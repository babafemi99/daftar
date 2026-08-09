package model

import (
	"strings"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
)

const AuditCollection = "audit_events"

type AuditAction string

const (
	AuditDocumentCreated    AuditAction = "document.created"
	AuditDocumentUpdated    AuditAction = "document.updated"
	AuditDocumentFinalized  AuditAction = "document.finalized"
	AuditDocumentDuplicated AuditAction = "document.duplicated"
	AuditDocumentArchived   AuditAction = "document.archived"
	AuditDocumentRestored   AuditAction = "document.restored"
)

type AuditMetadata struct {
	ChangedFields            []string `bson:"changedFields,omitempty" json:"changedFields,omitempty"`
	SourceDocumentID         *string  `bson:"sourceDocumentId,omitempty" json:"sourceDocumentId,omitempty"`
	CalculationPolicyVersion string   `bson:"calculationPolicyVersion,omitempty" json:"calculationPolicyVersion,omitempty"`
}

type AuditEvent struct {
	ID                string        `bson:"_id" json:"id"`
	OwnerID           string        `bson:"ownerId" json:"-"`
	ActorID           string        `bson:"actorId" json:"actorId"`
	DocumentID        string        `bson:"documentId" json:"documentId"`
	DocumentReference string        `bson:"documentReference" json:"documentReference"`
	Action            AuditAction   `bson:"action" json:"action"`
	DocumentVersion   int64         `bson:"documentVersion" json:"documentVersion"`
	Metadata          AuditMetadata `bson:"metadata" json:"metadata"`
	RequestID         string        `bson:"requestId" json:"requestId"`
	OccurredAt        time.Time     `bson:"occurredAt" json:"occurredAt"`
}

func NewAuditEvent(ownerID, actorID, requestID string, document Document, action AuditAction, metadata AuditMetadata, now time.Time) (AuditEvent, error) {
	event := AuditEvent{ID: lid.NewAudit(), OwnerID: ownerID, ActorID: actorID, DocumentID: document.ID, DocumentReference: document.Reference, Action: action, DocumentVersion: document.Version, Metadata: metadata, RequestID: requestID, OccurredAt: now.UTC()}
	return event, event.Validate()
}

func (event AuditEvent) Validate() error {
	if lid.Validate(event.ID, lid.Audit) != nil || lid.Validate(event.OwnerID, lid.User) != nil || lid.Validate(event.ActorID, lid.User) != nil || lid.Validate(event.DocumentID, lid.Document) != nil || event.OwnerID != event.ActorID || event.DocumentVersion < 1 || event.OccurredAt.IsZero() || strings.TrimSpace(event.RequestID) == "" {
		return buhari.New(buhari.CodeValidationFailed, "Audit event identity or timestamps are invalid.")
	}
	switch event.Action {
	case AuditDocumentCreated, AuditDocumentUpdated, AuditDocumentFinalized, AuditDocumentDuplicated, AuditDocumentArchived, AuditDocumentRestored:
	default:
		return buhari.New(buhari.CodeValidationFailed, "Audit action is invalid.")
	}
	if event.Metadata.SourceDocumentID != nil && lid.Validate(*event.Metadata.SourceDocumentID, lid.Document) != nil {
		return buhari.New(buhari.CodeValidationFailed, "Audit source document is invalid.")
	}
	if len(event.Metadata.ChangedFields) > 12 {
		return buhari.New(buhari.CodeValidationFailed, "Audit metadata is too large.")
	}
	allowed := map[string]bool{"title": true, "customer": true, "issueDate": true, "currency": true, "lineItems": true, "totals": true, "taxBreakdown": true, "status": true, "archivedAt": true}
	for _, field := range event.Metadata.ChangedFields {
		if !allowed[field] {
			return buhari.New(buhari.CodeValidationFailed, "Audit changed field is not allowed.")
		}
	}
	return nil
}
