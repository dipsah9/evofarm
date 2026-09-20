package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ---------- Config ----------

const (
	tokenLifetime = 24 * time.Hour
	bcryptCost    = 12
)

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is not set")
	}
	return []byte(secret)
}

// ---------- Request/Response types ----------

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name,omitempty"`
	Token  string `json:"token"`
}

// ---------- User model ----------

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         *string
}

// ---------- Password helpers ----------

func hashPassword(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	return string(bytes), err
}

func checkPassword(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// ---------- JWT helpers ----------

func generateToken(userID, email string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"iat":   now.Unix(),
		"exp":   now.Add(tokenLifetime).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

func parseToken(tokenStr string) (userID, email string, err error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return getJWTSecret(), nil
	})
	if err != nil {
		return "", "", err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", "", errors.New("invalid token")
	}
	sub, _ := claims["sub"].(string)
	emailClaim, _ := claims["email"].(string)
	if sub == "" {
		return "", "", errors.New("missing subject")
	}
	return sub, emailClaim, nil
}

// extractBearerToken pulls the token from "Authorization: Bearer <token>".
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// ---------- DB helpers ----------

func createUser(ctx context.Context, email, passwordHash, name string) (*User, error) {
	var user User
	var namePtr *string
	if name != "" {
		namePtr = &name
	}

	err := db.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, name
	`, email, passwordHash, namePtr).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func findUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := db.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, name
		FROM users
		WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func findUserByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := db.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, name
		FROM users
		WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ---------- Handlers ----------

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "auth not available", http.StatusServiceUnavailable)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Normalize email
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Validate
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Check for existing user
	if _, err := findUserByEmail(r.Context(), req.Email); err == nil {
		http.Error(w, "email already registered", http.StatusConflict)
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("register: lookup failed: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Hash password
	hash, err := hashPassword(req.Password)
	if err != nil {
		log.Printf("register: bcrypt failed: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Create user
	user, err := createUser(r.Context(), req.Email, hash, req.Name)
	if err != nil {
		log.Printf("register: create failed: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Generate token
	token, err := generateToken(user.ID, user.Email)
	if err != nil {
		log.Printf("register: token failed: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	name := ""
	if user.Name != nil {
		name = *user.Name
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(AuthResponse{
		UserID: user.ID,
		Email:  user.Email,
		Name:   name,
		Token:  token,
	})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "auth not available", http.StatusServiceUnavailable)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}

	user, err := findUserByEmail(r.Context(), req.Email)
	if err != nil {
		// Don't reveal whether the email exists.
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if !checkPassword(req.Password, user.PasswordHash) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := generateToken(user.ID, user.Email)
	if err != nil {
		log.Printf("login: token failed: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	name := ""
	if user.Name != nil {
		name = *user.Name
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		UserID: user.ID,
		Email:  user.Email,
		Name:   name,
		Token:  token,
	})
}

func meHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "auth not available", http.StatusServiceUnavailable)
		return
	}

	tokenStr := extractBearerToken(r)
	if tokenStr == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	userID, _, err := parseToken(tokenStr)
	if err != nil {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	user, err := findUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	name := ""
	if user.Name != nil {
		name = *user.Name
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"user_id": user.ID,
		"email":   user.Email,
		"name":    name,
	})
}

// ---------- Auth middleware (used later) ----------

// authMiddleware validates the JWT and injects the user ID into the request context.
// Use this to protect routes that require login.
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractBearerToken(r)
		if tokenStr == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		userID, _, err := parseToken(tokenStr)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Context key for the authenticated user ID.
type contextKey string

const userIDKey contextKey = "user_id"

// getUserID returns the authenticated user ID from the request context, if present.
func getUserID(r *http.Request) string {
	if v := r.Context().Value(userIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}