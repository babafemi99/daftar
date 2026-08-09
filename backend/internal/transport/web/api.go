package web

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/babafemi99/daftar/backend/internal/cfg"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

type API struct {
	Server         *http.Server
	config         cfg.HTTP
	cookie         cfg.Cookie
	jwt            cfg.JWT
	users          AuthUsers
	documents      Documents
	reports        Reports
	audits         Audits
	sessions       Sessions
	router         chi.Router
	trustedProxies []trustedNetwork
	logger         *slog.Logger
	slowRequest    time.Duration
}

func NewAPI(config cfg.HTTP) *API {
	return NewAPIWithLogger(config, slog.Default(), cfg.Default().Logging.SlowRequestThreshold)
}

func NewAPIWithLogger(config cfg.HTTP, logger *slog.Logger, slowRequest time.Duration) *API {
	config = config.WithDefaults()
	if logger == nil {
		logger = slog.Default()
	}
	if slowRequest <= 0 {
		slowRequest = cfg.Default().Logging.SlowRequestThreshold
	}

	api := &API{config: config, cookie: cfg.Default().Cookie, trustedProxies: parseTrustedProxies(config.TrustedProxies), logger: logger, slowRequest: slowRequest}
	api.router = api.routes()
	api.Server = &http.Server{
		Addr:              config.Address,
		Handler:           api.router,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		MaxHeaderBytes:    config.MaxHeaderBytes,
	}

	return api
}

func (a *API) routes() chi.Router {
	router := chi.NewRouter()
	a.applyMiddleware(router)

	router.Get("/health", a.health)
	router.Get("/api/v1/health/live", a.health)
	router.Route("/api/v1", func(router chi.Router) {
		router.With(httprate.LimitByIP(a.config.RegisterRatePerMinute, time.Minute)).Post("/auth/register", a.register)
		router.With(httprate.LimitByIP(a.config.LoginRatePerMinute, time.Minute)).Post("/auth/login", a.login)
		router.With(httprate.LimitByIP(a.config.LoginRatePerMinute, time.Minute)).Post("/auth/refresh", a.refresh)
		router.Post("/auth/logout", a.logout)
		router.With(a.AuthMiddleware).Get("/me", a.me)
		documents := router.With(a.AuthMiddleware)
		documents.Post("/documents", a.createDocument)
		documents.Get("/documents", a.listDocuments)
		documents.Post("/documents/preview-calculation", a.previewDocumentCalculation)
		documents.Get("/documents/{documentId}", a.getDocument)
		documents.Patch("/documents/{documentId}", a.replaceDocument)
		documents.Delete("/documents/{documentId}", a.archiveDocument)
		documents.Post("/documents/{documentId}/restore", a.restoreDocument)
		documents.Post("/documents/{documentId}/finalize", a.finalizeDocument)
		documents.Post("/documents/{documentId}/duplicate", a.duplicateDocument)
		documents.Post("/documents/{documentId}/line-items", a.addDocumentLine)
		documents.Post("/documents/{documentId}/line-items/reorder", a.reorderDocumentLines)
		documents.Patch("/documents/{documentId}/line-items/{lineItemId}", a.updateDocumentLine)
		documents.Delete("/documents/{documentId}/line-items/{lineItemId}", a.deleteDocumentLine)
		documents.Get("/documents/{documentId}/audit-events", a.listDocumentAuditEvents)
		router.With(a.AuthMiddleware).Get("/reports/summary", a.reportSummary)
	})

	return router
}

func (a *API) Handler() http.Handler {
	return a.router
}

func (a *API) Serve() error {
	if err := a.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (a *API) Shutdown(ctx context.Context) error {
	return a.Server.Shutdown(ctx)
}

type healthResponse struct {
	Status string `json:"status"`
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
}
