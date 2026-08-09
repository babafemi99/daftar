package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
	"github.com/babafemi99/daftar/backend/internal/pkg/requestctx"
	"github.com/babafemi99/daftar/backend/internal/pkg/tokoz"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

const requestIDHeader = "X-Request-ID"

var safeRequestID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

func (a *API) applyMiddleware(router chi.Router) {
	router.Use(a.RequestIDMiddleware)
	router.Use(a.TrustedProxyMiddleware)
	router.Use(a.SecurityHeadersMiddleware)
	router.Use(a.RequestLoggingMiddleware)
	router.Use(a.RecoveryMiddleware)
	router.Use(middleware.Timeout(a.config.RequestTimeout))
	router.Use(a.OriginMiddleware)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   a.config.CORSAllowedOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Idempotency-Key", "If-Match", requestIDHeader},
		ExposedHeaders:   []string{requestIDHeader},
		AllowCredentials: allowsCredentials(a.config.CORSAllowedOrigins),
		MaxAge:           300,
	}))
	router.Use(httprate.LimitByIP(a.config.RateLimitPerMinute, time.Minute))
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status    int
	bytes     int
	errorCode buhari.Code
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *loggingResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	written, err := w.ResponseWriter.Write(body)
	w.bytes += written
	return written, err
}

func (w *loggingResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func setResponseErrorCode(w http.ResponseWriter, code buhari.Code) {
	if response, ok := w.(*loggingResponseWriter); ok {
		response.errorCode = code
	}
}

func (a *API) RequestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		requestID := RequestIDFromContext(r.Context())
		fields := &requestctx.RequestLog{}
		logger := a.logger.With("requestId", requestID)
		ctx := requestctx.WithLogger(r.Context(), logger)
		ctx = requestctx.WithRequestLog(ctx, fields)
		response := &loggingResponseWriter{ResponseWriter: w}

		next.ServeHTTP(response, r.WithContext(ctx))

		status := response.status
		if status == 0 {
			status = http.StatusOK
		}
		duration := time.Since(startedAt)
		route := r.URL.Path
		if pattern := chi.RouteContext(r.Context()).RoutePattern(); pattern != "" {
			route = pattern
		}
		attributes := []any{
			"method", r.Method,
			"route", route,
			"status", status,
			"durationMs", duration.Milliseconds(),
			"responseBytes", response.bytes,
		}
		if clientIP := remoteIP(r.RemoteAddr); clientIP != nil {
			attributes = append(attributes, "clientIp", clientIP.String())
		}
		if fields.UserID != "" {
			attributes = append(attributes, "userId", fields.UserID)
		}
		if response.errorCode != "" {
			attributes = append(attributes, "errorCode", response.errorCode)
		}

		switch {
		case status >= http.StatusInternalServerError:
			logger.Error("request completed", attributes...)
		case status >= http.StatusBadRequest || duration >= a.slowRequest:
			logger.Warn("request completed", attributes...)
		default:
			logger.Info("request completed", attributes...)
		}
	})
}

func (a *API) RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				requestctx.Logger(r.Context()).Error("request panic recovered",
					"panicType", fmt.Sprintf("%T", recovered),
					"stack", string(debug.Stack()),
				)
				ResponseError(r.Context(), w, buhari.New(buhari.CodeInternalError, "An unexpected error occurred."))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type trustedNetwork struct{ network *net.IPNet }

func parseTrustedProxies(values []string) []trustedNetwork {
	result := make([]trustedNetwork, 0, len(values))
	for _, value := range values {
		if ip := net.ParseIP(value); ip != nil {
			bits := 128
			if ip.To4() != nil {
				ip, bits = ip.To4(), 32
			}
			result = append(result, trustedNetwork{network: &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}})
			continue
		}
		if _, network, err := net.ParseCIDR(value); err == nil {
			result = append(result, trustedNetwork{network: network})
		}
	}
	return result
}

func (a *API) TrustedProxyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer := remoteIP(r.RemoteAddr)
		if !a.isTrustedProxy(peer) {
			r.Header.Del("X-Forwarded-For")
			r.Header.Del("X-Real-IP")
			next.ServeHTTP(w, r)
			return
		}

		forwarded := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
		if len(forwarded) == 1 && strings.TrimSpace(forwarded[0]) == "" {
			forwarded = []string{r.Header.Get("X-Real-IP")}
		}
		for i := len(forwarded) - 1; i >= 0; i-- {
			candidate := net.ParseIP(strings.TrimSpace(forwarded[i]))
			if candidate != nil && !a.isTrustedProxy(candidate) {
				r.RemoteAddr = candidate.String()
				break
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) isTrustedProxy(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, trusted := range a.trustedProxies {
		if trusted.network.Contains(ip) {
			return true
		}
	}
	return false
}

func remoteIP(address string) net.IP {
	host, _, err := net.SplitHostPort(address)
	if err == nil {
		return net.ParseIP(host)
	}
	return net.ParseIP(address)
}

func (a *API) SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		if a.config.EnableHSTS {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// RequestIDMiddleware accepts a safe caller-provided request ID or generates a
// prefixed ULID, then makes it available to handlers and response logs.
func (a *API) RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get(requestIDHeader))
		if !safeRequestID.MatchString(requestID) {
			requestID = lid.NewRequest()
		}

		w.Header().Set(requestIDHeader, requestID)
		ctx := requestctx.WithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := requestctx.RequestID(ctx)
	return requestID
}

// AuthMiddleware authenticates a user access token and places its claims in a
// small executor model. Refresh tokens are rejected by ValidateAccessToken.
func (a *API) AuthMiddleware(next http.Handler) http.Handler {
	return a.authenticateRole("user", next)
}

func (a *API) AdminMiddleware(next http.Handler) http.Handler {
	return a.authenticateRole("admin", next)
}

// AuthenticateAdmin is retained as a readable route-level alias.
func (a *API) AuthenticateAdmin(next http.Handler) http.Handler {
	return a.AdminMiddleware(next)
}

func (a *API) authenticateRole(requiredRole string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := a.accessToken(r)
		if !ok {
			ResponseError(r.Context(), w, buhari.New(buhari.CodeUnauthorized, "A valid Bearer access token is required."))
			return
		}

		claims, err := tokoz.ValidateAccessToken(token)
		if err != nil {
			ResponseError(r.Context(), w, buhari.New(buhari.CodeUnauthorized, "The access token is invalid or expired."))
			return
		}
		if err := lid.Validate(claims.Subject, lid.User); err != nil {
			ResponseError(r.Context(), w, buhari.New(buhari.CodeUnauthorized, "The access token subject is invalid."))
			return
		}
		if !strings.EqualFold(claims.Role, requiredRole) {
			ResponseError(r.Context(), w, buhari.New(buhari.CodeForbidden, requiredRole+" access is required."))
			return
		}

		executor := model.Executor{
			ID:    model.UserID(claims.Subject),
			Email: claims.Email,
			Role:  strings.ToLower(claims.Role),
		}
		ctx := requestctx.WithExecutor(r.Context(), executor)
		if fields, ok := requestctx.RequestLogFields(ctx); ok {
			fields.UserID = string(executor.ID)
		}
		ctx = requestctx.WithLogger(ctx, requestctx.Logger(ctx).With("userId", executor.ID))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *API) accessToken(r *http.Request) (string, bool) {
	if cookie, err := r.Cookie(a.cookie.Name); err == nil && cookie.Value != "" {
		return cookie.Value, true
	}
	return bearerToken(r.Header.Get("Authorization"))
}

func (a *API) OriginMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		for _, allowed := range a.config.CORSAllowedOrigins {
			if allowed != "*" && origin == allowed {
				next.ServeHTTP(w, r)
				return
			}
		}
		ResponseError(r.Context(), w, buhari.New(buhari.CodeForbidden, "Request origin is not allowed."))
	})
}

func ExecutorFromContext(ctx context.Context) (model.Executor, bool) {
	return requestctx.Executor(ctx)
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func allowsCredentials(origins []string) bool {
	for _, origin := range origins {
		if origin == "*" {
			return false
		}
	}
	return len(origins) > 0
}
