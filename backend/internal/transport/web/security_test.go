package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/babafemi99/daftar/backend/internal/cfg"
)

func TestSecurityHeaders(t *testing.T) {
	api := NewAPI(cfg.HTTP{EnableHSTS: true})
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	for _, header := range []string{"X-Content-Type-Options", "Referrer-Policy", "X-Frame-Options", "Content-Security-Policy", "Strict-Transport-Security"} {
		if response.Header().Get(header) == "" {
			t.Errorf("missing security header %s", header)
		}
	}
}

func TestUntrustedPeerCannotSpoofForwardedIP(t *testing.T) {
	api := NewAPI(cfg.HTTP{TrustedProxies: []string{"10.0.0.0/8"}})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "203.0.113.10:1234"
	request.Header.Set("X-Forwarded-For", "198.51.100.20")

	var got string
	api.TrustedProxyMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = r.RemoteAddr
	})).ServeHTTP(httptest.NewRecorder(), request)

	if got != "203.0.113.10:1234" || request.Header.Get("X-Forwarded-For") != "" {
		t.Fatalf("RemoteAddr = %q, XFF = %q", got, request.Header.Get("X-Forwarded-For"))
	}
}

func TestTrustedPeerSetsClientIP(t *testing.T) {
	api := NewAPI(cfg.HTTP{TrustedProxies: []string{"10.0.0.0/8"}})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "10.0.0.2:1234"
	request.Header.Set("X-Forwarded-For", "198.51.100.20, 10.0.0.3")

	var got string
	api.TrustedProxyMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = r.RemoteAddr
	})).ServeHTTP(httptest.NewRecorder(), request)

	if got != "198.51.100.20" {
		t.Fatalf("RemoteAddr = %q", got)
	}
}
