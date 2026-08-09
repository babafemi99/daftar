package mercury

import (
	"errors"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/calculations"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
)

func TestValidateMutationTarget(t *testing.T) {
	draft := repositoryTestDocument(t)

	updated := draft
	updated.Version++
	if err := validateMutationTarget(updated, mutationReplace); err != nil {
		t.Fatalf("active draft replacement rejected: %v", err)
	}

	archived := draft
	if err := archived.Archive(draft.UpdatedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := validateMutationTarget(archived, mutationArchive); err != nil {
		t.Fatalf("archived target rejected: %v", err)
	}
	if err := validateMutationTarget(archived, mutationReplace); !repositoryHasCode(err, buhari.CodeConflict) {
		t.Fatalf("archive accepted as replacement: %v", err)
	}

	finalized := draft
	if err := finalized.Finalize(draft.UpdatedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := validateMutationTarget(finalized, mutationFinalize); err != nil {
		t.Fatalf("finalized target rejected: %v", err)
	}
	if err := validateMutationTarget(finalized, mutationReplace); !repositoryHasCode(err, buhari.CodeConflict) {
		t.Fatalf("finalized target accepted as replacement: %v", err)
	}
}

func TestClassifyMutationConflictPriority(t *testing.T) {
	draft := repositoryTestDocument(t)
	if err := classifyMutationConflict(draft, draft.Version, mutationRestore); !repositoryHasCode(err, buhari.CodeDocumentNotArchived) {
		t.Fatalf("active restore error = %v", err)
	}

	stale := classifyMutationConflict(draft, draft.Version-1, mutationReplace)
	if !repositoryHasCode(stale, buhari.CodeDocumentVersionConflict) {
		t.Fatalf("stale error = %v", stale)
	}

	archived := draft
	if err := archived.Archive(draft.UpdatedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := classifyMutationConflict(archived, draft.Version-1, mutationReplace); !repositoryHasCode(err, buhari.CodeDocumentArchived) {
		t.Fatalf("archived error = %v", err)
	}

	finalized := draft
	if err := finalized.Finalize(draft.UpdatedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := classifyMutationConflict(finalized, draft.Version-1, mutationReplace); !repositoryHasCode(err, buhari.CodeDocumentFinalized) {
		t.Fatalf("finalized error = %v", err)
	}
}

func repositoryTestDocument(t *testing.T) model.Document {
	t.Helper()
	now := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	document, err := model.NewDocument(lid.NewUser(), "DOC-2026-000001", model.DocumentInput{
		Title:     "Repository test",
		Customer:  "Acme Limited",
		IssueDate: time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC),
		Currency:  model.CurrencyUSD,
		LineItems: []calculations.LineInput{{Description: "Consulting", Quantity: 1, UnitPriceMinor: 10_000, TaxRate: 50_000}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func repositoryHasCode(err error, code buhari.Code) bool {
	var appErr *buhari.Error
	return errors.As(err, &appErr) && appErr.Code == code
}
