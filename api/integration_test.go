//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

// setupTestServer creates a fully wired test server with a real
// Postgres and Redis. It requires DATABASE_URL and REDIS_URL to be set.
func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")
	jwtSecret := os.Getenv("JWT_SECRET")

	if dbURL == "" || redisURL == "" || jwtSecret == "" {
		t.Skip("integration tests require DATABASE_URL, REDIS_URL, JWT_SECRET")
	}

	// Initialize globals
	var err error
	db, err = NewDB(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("could not connect to Postgres: %v", err)
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("could not parse REDIS_URL: %v", err)
	}
	rdb = redis.NewClient(opt)

	// Clean up any leftover test data
	ctx := context.Background()
	rdb.FlushDB(ctx)
	db.pool.Exec(ctx, "DELETE FROM jobs WHERE user_id IN (SELECT id FROM users WHERE email LIKE 'test-%@example.com')")
	db.pool.Exec(ctx, "DELETE FROM users WHERE email LIKE 'test-%@example.com'")

	// Build a router with the same setup as main.go
	rateCfg := RateLimitConfig{MaxRequests: 10, Window: 60 * time.Second}
	r := buildRouter(rateCfg)

	return httptest.NewServer(r)
}

// ---------- Helper functions ----------

func postJSON(t *testing.T, url string, body interface{}, token string) *http.Response {
	t.Helper()
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s failed: %v", url, err)
	}
	return resp
}

func getJSON(t *testing.T, url string, token string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body
}

// registerUser registers a fresh test user and returns (user_id, token).
func registerUser(t *testing.T, server *httptest.Server, suffix string) (string, string) {
	t.Helper()
	email := fmt.Sprintf("test-%s-%d@example.com", suffix, time.Now().UnixNano())
	body := map[string]string{
		"email":    email,
		"password": "testpassword123",
		"name":     "Test " + suffix,
	}
	resp := postJSON(t, server.URL+"/auth/register", body, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register failed: %d - %s", resp.StatusCode, readBody(t, resp))
	}
	var out map[string]string
	json.Unmarshal(readBody(t, resp), &out)
	return out["user_id"], out["token"]
}

// ---------- Tests ----------

func TestIntegration_RegisterAndLogin(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	email := fmt.Sprintf("test-login-%d@example.com", time.Now().UnixNano())

	// Register
	resp := postJSON(t, server.URL+"/auth/register", map[string]string{
		"email": email, "password": "testpassword123", "name": "Login Test",
	}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", resp.StatusCode)
	}

	// Login with correct password
	resp = postJSON(t, server.URL+"/auth/login", map[string]string{
		"email": email, "password": "testpassword123",
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: expected 200, got %d", resp.StatusCode)
	}

	// Login with wrong password
	resp = postJSON(t, server.URL+"/auth/login", map[string]string{
		"email": email, "password": "wrongpassword",
	}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong password: expected 401, got %d", resp.StatusCode)
	}
}

func TestIntegration_JobSubmissionRequiresAuth(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// No token
	resp := postJSON(t, server.URL+"/jobs", map[string]interface{}{
		"problem": "xor", "population_size": 50, "generations": 10,
	}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestIntegration_SubmitAndFetchJob(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	_, token := registerUser(t, server, "submit")

	// Submit a job
	resp := postJSON(t, server.URL+"/jobs", map[string]interface{}{
		"problem": "xor", "population_size": 50, "generations": 10,
	}, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("submit: expected 200, got %d - %s", resp.StatusCode, readBody(t, resp))
	}

	var submitted map[string]string
	json.Unmarshal(readBody(t, resp), &submitted)
	jobID := submitted["job_id"]
	if jobID == "" {
		t.Fatal("no job_id in response")
	}

	// Fetch the job
	resp = getJSON(t, server.URL+"/jobs/"+jobID, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fetch: expected 200, got %d", resp.StatusCode)
	}

	var job map[string]interface{}
	json.Unmarshal(readBody(t, resp), &job)
	if job["job_id"] != jobID {
		t.Fatalf("job_id mismatch: got %v", job["job_id"])
	}
}

func TestIntegration_CrossUserIsolation(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	_, tokenA := registerUser(t, server, "alice")
	_, tokenB := registerUser(t, server, "bob")

	// Alice submits a job
	resp := postJSON(t, server.URL+"/jobs", map[string]interface{}{
		"problem": "xor", "population_size": 50, "generations": 10,
	}, tokenA)
	var submitted map[string]string
	json.Unmarshal(readBody(t, resp), &submitted)
	jobID := submitted["job_id"]

	// Bob tries to fetch Alice's job
	resp = getJSON(t, server.URL+"/jobs/"+jobID, tokenB)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-user access: expected 404, got %d", resp.StatusCode)
	}
}

func TestIntegration_HistoryFilteredByUser(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	_, tokenA := registerUser(t, server, "history-a")
	_, tokenB := registerUser(t, server, "history-b")

	// Alice submits a job
	postJSON(t, server.URL+"/jobs", map[string]interface{}{
		"problem": "xor", "population_size": 50, "generations": 10,
	}, tokenA)

	// Bob's history should be empty
	resp := getJSON(t, server.URL+"/jobs/history", tokenB)
	var bobHistory []map[string]interface{}
	json.Unmarshal(readBody(t, resp), &bobHistory)
	if len(bobHistory) != 0 {
		t.Fatalf("Bob's history should be empty, got %d", len(bobHistory))
	}

	// Alice's history should have her job
	resp = getJSON(t, server.URL+"/jobs/history", tokenA)
	var aliceHistory []map[string]interface{}
	json.Unmarshal(readBody(t, resp), &aliceHistory)
	if len(aliceHistory) == 0 {
		t.Fatal("Alice's history should not be empty")
	}
}

func TestIntegration_RateLimit(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	_, token := registerUser(t, server, "ratelimit")

	// Submit 15 jobs rapidly (limit is 10/min)
	statuses := []int{}
	for i := 0; i < 15; i++ {
		resp := postJSON(t, server.URL+"/jobs", map[string]interface{}{
			"problem": "xor", "population_size": 10, "generations": 5,
		}, token)
		statuses = append(statuses, resp.StatusCode)
		resp.Body.Close()
	}

	// Count 429s
	limited := 0
	for _, s := range statuses {
		if s == http.StatusTooManyRequests {
			limited++
		}
	}
	if limited == 0 {
		t.Fatal("expected at least one 429, got none")
	}
	if limited < 3 {
		t.Fatalf("expected at least 3 rate limits, got %d", limited)
	}
}