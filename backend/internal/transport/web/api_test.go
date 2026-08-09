package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/babafemi99/daftar/backend/internal/cfg"
)

func TestHealth(t *testing.T) {
	api := NewAPI(cfg.HTTP{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q", contentType)
	}

	var body healthResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("health response = %#v", body)
	}
}

func TestHealthAlias(t *testing.T) {
	api := NewAPI(cfg.HTTP{})
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}
