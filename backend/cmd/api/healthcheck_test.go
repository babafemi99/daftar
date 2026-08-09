package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunHealthcheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("DAFTAR_HEALTHCHECK_URL", server.URL)

	if status := runHealthcheck(); status != 0 {
		t.Fatalf("runHealthcheck() = %d, want 0", status)
	}
}

func TestRunHealthcheckRejectsUnhealthyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	t.Setenv("DAFTAR_HEALTHCHECK_URL", server.URL)

	if status := runHealthcheck(); status == 0 {
		t.Fatal("runHealthcheck() = 0, want failure")
	}
}
