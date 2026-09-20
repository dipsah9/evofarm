package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// RateLimitConfig holds the tunables for the rate limiter.
type RateLimitConfig struct {
	MaxRequests int           // Max requests allowed per window
	Window      time.Duration // Length of the window
}

// LoadRateLimitConfig reads configuration from environment with sensible defaults.
func LoadRateLimitConfig() RateLimitConfig {
	maxReqs := 10
	if v := os.Getenv("RATE_LIMIT_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxReqs = n
		}
	}

	windowSecs := 60
	if v := os.Getenv("RATE_LIMIT_WINDOW_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			windowSecs = n
		}
	}

	return RateLimitConfig{
		MaxRequests: maxReqs,
		Window:      time.Duration(windowSecs) * time.Second,
	}
}

// extractClientIP finds the real client IP, accounting for Fly.io's proxy.
func extractClientIP(r *http.Request) string {
	// Fly.io sets Fly-Client-IP with the real client address.
	if ip := r.Header.Get("Fly-Client-IP"); ip != "" {
		return ip
	}
	// X-Forwarded-For may contain a comma-separated list; take the first.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	// Fallback: connection remote address
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// checkRateLimit increments the counter for the given key and returns:
//   allowed    — true if the request should proceed
//   remaining  — requests left in the current window
//   retryAfter — seconds until the window resets (only meaningful if !allowed)
func checkRateLimit(ctx context.Context, rdb *redis.Client, key string, cfg RateLimitConfig) (bool, int, int) {
	// INCR is atomic — safe across multiple API machines
	count, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		// If Redis fails, fail open (allow the request). Better to serve
		// users than to lock everyone out due to an infrastructure issue.
		log.Printf("rate limit: INCR failed, failing open: %v", err)
		return true, cfg.MaxRequests, 0
	}

	// First request in this window: set the TTL.
	if count == 1 {
		rdb.Expire(ctx, key, cfg.Window)
	}

	remaining := cfg.MaxRequests - int(count)
	if remaining < 0 {
		remaining = 0
	}

	if int(count) > cfg.MaxRequests {
		// Read the remaining TTL so we can tell the client when to retry.
		ttl, err := rdb.TTL(ctx, key).Result()
		retryAfter := int(cfg.Window.Seconds())
		if err == nil && ttl > 0 {
			retryAfter = int(ttl.Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
		}
		return false, 0, retryAfter
	}

	return true, remaining, 0
}

// rateLimitMiddleware wraps a handler and enforces per-user (or per-IP) rate limits.
func rateLimitMiddleware(cfg RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prefer authenticated user ID for rate limiting.
			// Fall back to IP for unauthenticated requests.
			var key string
			if userID := getUserID(r); userID != "" {
				key = fmt.Sprintf("rate:job:user:%s", userID)
			} else {
				ip := extractClientIP(r)
				key = fmt.Sprintf("rate:job:ip:%s", ip)
			}

			allowed, remaining, retryAfter := checkRateLimit(r.Context(), rdb, key, cfg)

			// Always expose rate-limit headers so clients can self-throttle.
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.MaxRequests))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":       "rate_limit_exceeded",
					"message":     fmt.Sprintf("Too many job submissions. Limit is %d per %s.", cfg.MaxRequests, cfg.Window),
					"retry_after": retryAfter,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitStatusHandler exposes the current rate-limit state for the caller's IP.
// Useful for debugging and for a future "quota remaining" UI badge.
func rateLimitStatusHandler(cfg RateLimitConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := extractClientIP(r)
		key := fmt.Sprintf("rate:job:%s", ip)

		count, err := rdb.Get(r.Context(), key).Int()
		if err == redis.Nil {
			count = 0
		} else if err != nil {
			log.Printf("rate limit status: GET failed: %v", err)
			http.Error(w, "rate limit backend unavailable", http.StatusServiceUnavailable)
			return
		}

		ttl, _ := rdb.TTL(r.Context(), key).Result()
		retryAfter := 0
		if ttl > 0 {
			retryAfter = int(ttl.Seconds())
		}

		remaining := cfg.MaxRequests - count
		if remaining < 0 {
			remaining = 0
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip":           ip,
			"limit":        cfg.MaxRequests,
			"used":         count,
			"remaining":    remaining,
			"reset_in_sec": retryAfter,
		})
	}
}