//go:build integration

package mercury

import (
	"context"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/calculations"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
)

func TestAuditRepositoryAppendAndOwnerScopedListIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repository := NewAuditRepository(newMongoTestDatabase(t))
	if err := repository.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	ownerID, otherOwnerID := lid.NewUser(), lid.NewUser()
	document := auditTestDocument(t, ownerID)
	older, err := model.NewAuditEvent(ownerID, ownerID, lid.NewRequest(), document, model.AuditDocumentCreated, model.AuditMetadata{}, time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	newer, err := model.NewAuditEvent(ownerID, ownerID, lid.NewRequest(), document, model.AuditDocumentUpdated, model.AuditMetadata{ChangedFields: []string{"lineItems", "totals"}}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []model.AuditEvent{older, newer} {
		if err := repository.Append(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	events, err := repository.ListDocument(ctx, ownerID, document.ID, 10)
	if err != nil || len(events) != 2 || events[0].ID != newer.ID || events[1].ID != older.ID {
		t.Fatalf("events=%+v err=%v", events, err)
	}
	isolated, err := repository.ListDocument(ctx, otherOwnerID, document.ID, 10)
	if err != nil || len(isolated) != 0 {
		t.Fatalf("cross-owner events=%+v err=%v", isolated, err)
	}
}

func auditTestDocument(t *testing.T, ownerID string) model.Document {
	t.Helper()
	now := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
	document, err := model.NewDocument(ownerID, "DOC-2026-000001", model.DocumentInput{
		Title: "Audit", Customer: "Acme", IssueDate: now, Currency: model.CurrencyUSD,
		LineItems: []calculations.LineInput{{Description: "Service", Quantity: 1, UnitPriceMinor: 1000}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	return document
}
