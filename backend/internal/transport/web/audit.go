package web

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/mercury"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/go-chi/chi/v5"
)

var ErrAuditServiceRequired = errors.New("web: audit service is required")

type Audits interface {
	ListDocument(context.Context, string, int64) ([]model.AuditEvent, error)
}

func (a *API) ConfigureAudits(audits Audits) error {
	if audits == nil {
		return ErrAuditServiceRequired
	}
	a.audits = audits
	return nil
}

type auditEventResponse struct {
	ID                string              `json:"id"`
	ActorID           string              `json:"actorId"`
	DocumentID        string              `json:"documentId"`
	DocumentReference string              `json:"documentReference"`
	Action            model.AuditAction   `json:"action"`
	DocumentVersion   int64               `json:"documentVersion"`
	Metadata          model.AuditMetadata `json:"metadata"`
	RequestID         string              `json:"requestId"`
	OccurredAt        string              `json:"occurredAt"`
}

func (a *API) listDocumentAuditEvents(w http.ResponseWriter, r *http.Request) {
	limit := int64(0)
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed < 1 || parsed > mercury.MaximumAuditLimit {
			ResponseError(r.Context(), w, buhari.InvalidFilter(buhari.FieldError{Path: "limit", Code: buhari.CodeInvalidFilter, Message: "Limit must be an integer between 1 and 100."}))
			return
		}
		limit = parsed
	}
	events, err := a.auditService().ListDocument(r.Context(), chi.URLParam(r, "documentId"), limit)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	result := make([]auditEventResponse, len(events))
	for index, event := range events {
		result[index] = auditEventResponse{ID: event.ID, ActorID: event.ActorID, DocumentID: event.DocumentID, DocumentReference: event.DocumentReference, Action: event.Action, DocumentVersion: event.DocumentVersion, Metadata: event.Metadata, RequestID: event.RequestID, OccurredAt: event.OccurredAt.Format("2006-01-02T15:04:05.999999999Z07:00")}
	}
	CollectionResponse(r.Context(), w, http.StatusOK, result)
}

type unavailableAudits struct{}

func (unavailableAudits) ListDocument(context.Context, string, int64) ([]model.AuditEvent, error) {
	return nil, errors.New("audit service not configured")
}
func (a *API) auditService() Audits {
	if a.audits == nil {
		return unavailableAudits{}
	}
	return a.audits
}
