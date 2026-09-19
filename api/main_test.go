package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHealthCheck verifies that /health returns 200 and the correct body.
func TestHealthCheck(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	healthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body["status"] != "healthy" {
		t.Errorf("expected status=healthy, got %q", body["status"])
	}
}

// TestListProblems verifies the problem catalog is returned.
func TestListProblems(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/problems", nil)
	rec := httptest.NewRecorder()

	listProblems(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var catalog map[string]map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&catalog); err != nil {
		t.Fatalf("could not decode catalog: %v", err)
	}

	// We expect at least these three problems.
	for _, name := range []string{"xor", "portfolio", "nurse_rostering"} {
		if _, ok := catalog[name]; !ok {
			t.Errorf("expected problem %q in catalog", name)
		}
	}
}