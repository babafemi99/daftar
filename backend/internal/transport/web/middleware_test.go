package web

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/cfg"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
	"github.com/babafemi99/daftar/backend/internal/pkg/tokoz"
)

func decodeLogRecord(t *testing.T, output *bytes.Buffer) map[string]any {
	t.Helper()
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &record); err != nil {
		t.Fatalf("decode log record: %v; output = %q", err, output.String())
	}
	return record
}

func TestRequestLoggingMiddlewareWritesStructuredCompletion(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	api := NewAPIWithLogger(cfg.HTTP{}, logger, time.Second)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	request.Header.Set(requestIDHeader, "req-structured-log")
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	record := decodeLogRecord(t, &output)
	if record["msg"] != "request completed" || record["requestId"] != "req-structured-log" || record["method"] != http.MethodGet {
		t.Fatalf("log record = %#v", record)
	}
	if record["route"] != "/api/v1/health/live" || record["status"] != float64(http.StatusOK) {
		t.Fatalf("log record = %#v", record)
	}
}

func TestRequestLoggingIncludesApplicationErrorAndRedactsCredentials(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	api := NewAPIWithLogger(cfg.HTTP{}, logger, time.Second)
	handler := api.RequestIDMiddleware(api.RequestLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ResponseError(r.Context(), w, buhari.New(buhari.CodeDocumentVersionConflict, "The document version is stale."))
	})))
	request := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewBufferString(`{"password":"never-log-me"}`))
	request.Header.Set("Authorization", "Bearer never-log-this-token")
	request.Header.Set("Cookie", "daftar_session=never-log-this-cookie")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	record := decodeLogRecord(t, &output)
	if record["errorCode"] != string(buhari.CodeDocumentVersionConflict) || record["status"] != float64(http.StatusConflict) {
		t.Fatalf("log record = %#v", record)
	}
	for _, secret := range []string{"never-log-me", "never-log-this-token", "never-log-this-cookie"} {
		if bytes.Contains(output.Bytes(), []byte(secret)) {
			t.Fatalf("log output contains secret %q: %s", secret, output.String())
		}
	}
}

func TestRequestLoggingIncludesAuthenticatedUserID(t *testing.T) {
	tokoz.Configure(tokoz.Config{
		Secret:    "test-secret-with-enough-entropy",
		Issuer:    "daftar-test",
		Audience:  "daftar-test-client",
		AccessTTL: time.Hour,
	})
	userID := lid.NewUser()
	token, err := tokoz.GenerateToken(userID, "user", "user@example.com")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	api := NewAPIWithLogger(cfg.HTTP{}, logger, time.Second)
	handler := api.RequestIDMiddleware(api.RequestLoggingMiddleware(api.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))))
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	record := decodeLogRecord(t, &output)
	if record["userId"] != userID {
		t.Fatalf("log record = %#v", record)
	}
}

func TestMiddlewareAddsRequestID(t *testing.T) {
	api := NewAPI(cfg.HTTP{})
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	if response.Header().Get(requestIDHeader) == "" {
		t.Fatal("response is missing X-Request-ID")
	}
}

func TestRequestIDMiddlewarePreservesSafeClientID(t *testing.T) {
	api := NewAPI(cfg.HTTP{})
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set(requestIDHeader, "client-request-123")
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	if got := response.Header().Get(requestIDHeader); got != "client-request-123" {
		t.Fatalf("X-Request-ID = %q", got)
	}
}

func TestAuthMiddleware(t *testing.T) {
	tokoz.Configure(tokoz.Config{
		Secret:    "test-secret-with-enough-entropy",
		Issuer:    "daftar-test",
		Audience:  "daftar-test-client",
		AccessTTL: time.Hour,
	})
	userID := lid.NewUser()
	token, err := tokoz.GenerateToken(userID, "user", "user@example.com")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	api := NewAPI(cfg.HTTP{})
	protected := api.RequestIDMiddleware(api.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executor, ok := ExecutorFromContext(r.Context())
		if !ok || string(executor.ID) != userID {
			t.Errorf("executor = %#v, ok = %v", executor, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	})))
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	protected.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestAuthMiddlewareRejectsMissingBearerToken(t *testing.T) {
	api := NewAPI(cfg.HTTP{})
	protected := api.RequestIDMiddleware(api.AuthMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler was called")
	})))
	response := httptest.NewRecorder()
	protected.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body ErrorEnvelope
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.RequestID == "" || body.Error.Code == "" {
		t.Fatalf("error response = %#v", body)
	}
}

func TestCORSExplicitOriginAllowsCredentials(t *testing.T) {
	api := NewAPI(cfg.HTTP{CORSAllowedOrigins: []string{"https://app.example.com"}})
	request := httptest.NewRequest(http.MethodOptions, "/health", nil)
	request.Header.Set("Origin", "https://app.example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	if response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q", response.Header().Get("Access-Control-Allow-Origin"))
	}
	if response.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q", response.Header().Get("Access-Control-Allow-Credentials"))
	}
}

func TestWildcardCORSDisablesCredentials(t *testing.T) {
	if allowsCredentials([]string{"*"}) {
		t.Fatal("allowsCredentials(*) = true")
	}
}
