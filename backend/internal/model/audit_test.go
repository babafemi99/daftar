package model

import (
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/calculations"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
)

func TestAuditEventValidation(t *testing.T) {
	ownerID := lid.NewUser()
	now := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
	document, err := NewDocument(ownerID, "DOC-2026-000001", DocumentInput{
		Title: "Audit", Customer: "Acme", IssueDate: now, Currency: CurrencyUSD,
		LineItems: []calculations.LineInput{{Description: "Service", Quantity: 1, UnitPriceMinor: 1000}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	event, err := NewAuditEvent(ownerID, ownerID, lid.NewRequest(), document, AuditDocumentCreated, AuditMetadata{ChangedFields: []string{"title", "totals"}}, now)
	if err != nil || event.Validate() != nil {
		t.Fatalf("valid event: event=%+v err=%v", event, err)
	}

	event.Action = "document.deleted"
	if event.Validate() == nil {
		t.Fatal("unsupported audit action was accepted")
	}
	event.Action = AuditDocumentCreated
	event.Metadata.ChangedFields = []string{"unitPrice"}
	if event.Validate() == nil {
		t.Fatal("unsafe audit metadata was accepted")
	}
}
