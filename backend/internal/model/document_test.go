package model

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/calculations"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
)

func TestNewDocumentBuildsValidCalculatedDraft(t *testing.T) {
	now := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	document, err := NewDocument(lid.NewUser(), "DOC-2026-000001", validDocumentInput(), now)
	if err != nil {
		t.Fatal(err)
	}
	if document.Status != DocumentDraft || document.Version != 1 {
		t.Fatalf("unexpected lifecycle: status=%s version=%d", document.Status, document.Version)
	}
	if lid.Validate(document.ID, lid.Document) != nil || lid.Validate(document.LineItems[0].ID, lid.Line) != nil {
		t.Fatal("document and line IDs were not generated")
	}
	if document.Totals != (calculations.Totals{SubtotalMinor: 20_000, DiscountMinor: 2_000, TaxMinor: 900, GrandTotalMinor: 18_900}) {
		t.Fatalf("unexpected totals: %+v", document.Totals)
	}
	if err := document.Validate(); err != nil {
		t.Fatalf("created document is invalid: %v", err)
	}
}

func TestDraftLifecycle(t *testing.T) {
	created := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	document, err := NewDocument(lid.NewUser(), "DOC-2026-000001", validDocumentInput(), created)
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Restore(created.Add(time.Minute)); !hasDocumentCode(err, buhari.CodeDocumentNotArchived) {
		t.Fatalf("active restore error = %v", err)
	}

	archived := created.Add(time.Hour)
	if err := document.Archive(archived); err != nil {
		t.Fatal(err)
	}
	if document.ArchivedAt == nil || document.Version != 2 {
		t.Fatalf("document was not archived: %+v", document)
	}
	if err := document.ReplaceDraft(validDocumentInput(), archived.Add(time.Minute)); !hasDocumentCode(err, buhari.CodeDocumentArchived) {
		t.Fatalf("archived edit error = %v", err)
	}

	restored := archived.Add(2 * time.Hour)
	if err := document.Restore(restored); err != nil {
		t.Fatal(err)
	}
	if document.ArchivedAt != nil || document.Version != 3 {
		t.Fatalf("document was not restored: %+v", document)
	}

	finalized := restored.Add(time.Hour)
	if err := document.Finalize(finalized); err != nil {
		t.Fatal(err)
	}
	if document.Status != DocumentFinalized || document.FinalizedAt == nil || !document.FinalizedAt.Equal(finalized) || document.Version != 4 {
		t.Fatalf("document was not finalized: %+v", document)
	}
	if err := document.ReplaceDraft(validDocumentInput(), finalized.Add(time.Hour)); !hasDocumentCode(err, buhari.CodeDocumentFinalized) {
		t.Fatalf("finalized edit error = %v", err)
	}
	if err := document.Archive(finalized.Add(time.Hour)); !hasDocumentCode(err, buhari.CodeDocumentFinalized) {
		t.Fatalf("finalized archive error = %v", err)
	}
}

func TestFinalizeRequiresLine(t *testing.T) {
	input := validDocumentInput()
	input.LineItems = nil
	document, err := NewDocument(lid.NewUser(), "DOC-2026-000001", input, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Finalize(time.Now().UTC()); !hasDocumentCode(err, buhari.CodeDocumentRequiresLine) {
		t.Fatalf("finalize error = %v", err)
	}
}

func TestReplaceDraftIsAtomicAndPreservesExistingLineID(t *testing.T) {
	now := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	document, err := NewDocument(lid.NewUser(), "DOC-2026-000001", validDocumentInput(), now)
	if err != nil {
		t.Fatal(err)
	}
	lineID := document.LineItems[0].ID

	input := validDocumentInput()
	input.Title = "Updated title"
	input.LineItems[0].ID = lineID
	input.LineItems[0].Quantity = 3
	if err := document.ReplaceDraft(input, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if document.LineItems[0].ID != lineID || document.Version != 2 || document.Totals.SubtotalMinor != 30_000 {
		t.Fatalf("unexpected update: %+v", document)
	}

	before := document
	invalid := input
	invalid.LineItems[0].Quantity = 0
	if err := document.ReplaceDraft(invalid, now.Add(2*time.Hour)); err == nil {
		t.Fatal("expected invalid update to fail")
	}
	if !reflect.DeepEqual(document, before) {
		t.Fatal("failed update mutated the document")
	}
}

func TestDocumentValidateRejectsTamperedCalculatedValues(t *testing.T) {
	document, err := NewDocument(lid.NewUser(), "DOC-2026-000001", validDocumentInput(), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	document.Totals.GrandTotalMinor++
	if err := document.Validate(); !hasDocumentCode(err, buhari.CodeValidationFailed) {
		t.Fatalf("validation error = %v", err)
	}
}

func TestDocumentRejectsUnsupportedCurrencyAndReplacesClientLineIDs(t *testing.T) {
	input := validDocumentInput()
	input.Currency = Currency("BTC")
	_, err := NewDocument(lid.NewUser(), "DOC-2026-000001", input, time.Now().UTC())
	if !hasDocumentCode(err, buhari.CodeValidationFailed) {
		t.Fatalf("currency error = %v", err)
	}

	lineID := lid.NewLine()
	input = validDocumentInput()
	input.LineItems = append(input.LineItems, input.LineItems[0])
	input.LineItems[0].ID = lineID
	input.LineItems[1].ID = lineID
	document, err := NewDocument(lid.NewUser(), "DOC-2026-000001", input, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if document.LineItems[0].ID == lineID || document.LineItems[1].ID == lineID || document.LineItems[0].ID == document.LineItems[1].ID {
		t.Fatalf("creation retained client line IDs: %+v", document.LineItems)
	}
}

func TestReplaceDraftReconcilesLineIDs(t *testing.T) {
	now := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	document, err := NewDocument(lid.NewUser(), "DOC-2026-000001", validDocumentInput(), now)
	if err != nil {
		t.Fatal(err)
	}
	existingID := document.LineItems[0].ID
	input := validDocumentInput()
	input.LineItems = append(input.LineItems, input.LineItems[0])
	input.LineItems[0].ID = existingID
	input.LineItems[1].ID = ""
	if err := document.ReplaceDraft(input, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if document.LineItems[0].ID != existingID || document.LineItems[1].ID == "" || document.LineItems[1].ID == existingID {
		t.Fatalf("line IDs were not reconciled: %+v", document.LineItems)
	}

	before := document
	unknown := input
	unknown.LineItems = append([]calculations.LineInput(nil), input.LineItems...)
	unknown.LineItems[0].ID = lid.NewLine()
	if err := document.ReplaceDraft(unknown, now.Add(2*time.Hour)); !hasDocumentCode(err, buhari.CodeValidationFailed) {
		t.Fatalf("unknown line ID error = %v", err)
	}
	if !reflect.DeepEqual(document, before) {
		t.Fatal("unknown line ID mutated document")
	}

	duplicate := input
	duplicate.LineItems = append([]calculations.LineInput(nil), input.LineItems...)
	duplicate.LineItems[0].ID = existingID
	duplicate.LineItems[1].ID = existingID
	if err := document.ReplaceDraft(duplicate, now.Add(3*time.Hour)); !hasDocumentCode(err, buhari.CodeValidationFailed) {
		t.Fatalf("duplicate line ID error = %v", err)
	}
	if !reflect.DeepEqual(document, before) {
		t.Fatal("duplicate line ID mutated document")
	}
}

func TestDocumentDoesNotRetainCallerDiscountOnCreate(t *testing.T) {
	discount := &calculations.Discount{Type: calculations.DiscountPercentage, Value: 100_000}
	input := validDocumentInput()
	input.LineItems[0].Discount = discount
	document, err := NewDocument(lid.NewUser(), "DOC-2026-000001", input, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	beforeLine := document.LineItems[0]
	beforeTotals := document.Totals
	beforeBreakdown := append([]calculations.TaxBreakdown(nil), document.TaxBreakdown...)
	discount.Value = 500_000
	if !reflect.DeepEqual(document.LineItems[0], beforeLine) {
		t.Fatalf("caller mutation changed stored line: before=%+v after=%+v", beforeLine, document.LineItems[0])
	}
	if document.Totals != beforeTotals || !reflect.DeepEqual(document.TaxBreakdown, beforeBreakdown) {
		t.Fatalf("caller mutation changed calculations: totals=%+v breakdown=%+v", document.Totals, document.TaxBreakdown)
	}
}

func TestDocumentDoesNotRetainCallerDiscountOnReplacement(t *testing.T) {
	now := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	document, err := NewDocument(lid.NewUser(), "DOC-2026-000001", validDocumentInput(), now)
	if err != nil {
		t.Fatal(err)
	}
	discount := &calculations.Discount{Type: calculations.DiscountFixed, Value: 1_000}
	input := validDocumentInput()
	input.LineItems[0].ID = document.LineItems[0].ID
	input.LineItems[0].Discount = discount
	if err := document.ReplaceDraft(input, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	beforeLine := document.LineItems[0]
	beforeTotals := document.Totals
	beforeBreakdown := append([]calculations.TaxBreakdown(nil), document.TaxBreakdown...)
	discount.Value = 5_000
	if !reflect.DeepEqual(document.LineItems[0], beforeLine) {
		t.Fatalf("caller mutation changed stored line: before=%+v after=%+v", beforeLine, document.LineItems[0])
	}
	if document.Totals != beforeTotals || !reflect.DeepEqual(document.TaxBreakdown, beforeBreakdown) {
		t.Fatalf("caller mutation changed calculations: totals=%+v breakdown=%+v", document.Totals, document.TaxBreakdown)
	}
}

func validDocumentInput() DocumentInput {
	return DocumentInput{
		Title:     "August Software Services",
		Customer:  "Acme Limited",
		IssueDate: time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC),
		Currency:  CurrencyUSD,
		LineItems: []calculations.LineInput{{
			Description:    "Widget A",
			Quantity:       2,
			UnitPriceMinor: 10_000,
			Discount:       &calculations.Discount{Type: calculations.DiscountPercentage, Value: 100_000},
			TaxRate:        50_000,
		}},
	}
}

func hasDocumentCode(err error, code buhari.Code) bool {
	var appErr *buhari.Error
	return errors.As(err, &appErr) && appErr.Code == code
}
