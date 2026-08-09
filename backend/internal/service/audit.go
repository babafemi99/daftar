package service

import (
	"context"
	"errors"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/mercury"
	"github.com/babafemi99/daftar/backend/internal/model"
)

var ErrAuditDocumentsRequired = errors.New("service: document repository is required for audits")

type AuditService struct {
	audits    mercury.Audits
	documents mercury.Documents
}

func NewAuditService(audits mercury.Audits, documents mercury.Documents) (*AuditService, error) {
	if audits == nil {
		return nil, ErrAuditRepositoryRequired
	}
	if documents == nil {
		return nil, ErrAuditDocumentsRequired
	}
	return &AuditService{audits: audits, documents: documents}, nil
}

func (service *AuditService) ListDocument(ctx context.Context, documentID string, limit int64) ([]model.AuditEvent, error) {
	ownerID, err := authenticatedOwner(ctx)
	if err != nil {
		return nil, err
	}
	if limit < 0 || limit > mercury.MaximumAuditLimit {
		return nil, buhari.InvalidFilter(buhari.FieldError{Path: "limit", Code: buhari.CodeInvalidFilter, Message: "Limit must be between 1 and 100 when provided."})
	}
	if _, err := service.documents.FindByID(ctx, ownerID, documentID); err != nil {
		return nil, err
	}
	return service.audits.ListDocument(ctx, ownerID, documentID, limit)
}
