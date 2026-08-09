//go:build integration

package mercury

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/calculations"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
)

func TestDocumentRepositoryIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	database := newMongoTestDatabase(t)

	repository := NewDocumentRepository(database)
	if err := repository.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	references := NewDocumentReferenceRepository(database)
	if err := references.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	referenceOwner := lid.NewUser()
	firstReference, err := references.Allocate(ctx, referenceOwner, 2026, time.Now().UTC())
	if err != nil || firstReference != "DOC-2026-000001" {
		t.Fatalf("first allocated reference=%q error=%v", firstReference, err)
	}
	secondReference, err := references.Allocate(ctx, referenceOwner, 2026, time.Now().UTC())
	if err != nil || secondReference != "DOC-2026-000002" {
		t.Fatalf("second allocated reference=%q error=%v", secondReference, err)
	}
	document := repositoryTestDocument(t)
	if err := repository.Create(ctx, document); err != nil {
		t.Fatal(err)
	}

	loaded, err := repository.FindByID(ctx, document.OwnerID, document.ID)
	if err != nil || loaded.ID != document.ID {
		t.Fatalf("FindByID() document=%s error=%v", loaded.ID, err)
	}
	if _, err := repository.FindByID(ctx, lid.NewUser(), document.ID); !repositoryHasCode(err, buhari.CodeNotFound) {
		t.Fatalf("cross-owner lookup error = %v", err)
	}
	page, err := repository.List(ctx, document.OwnerID, DocumentListFilter{})
	if err != nil || len(page.Documents) != 1 || page.TotalItems != 1 {
		t.Fatalf("List() page=%+v error=%v", page, err)
	}
	searched, err := repository.List(ctx, document.OwnerID, DocumentListFilter{Search: "repository"})
	if err != nil || len(searched.Documents) != 1 || searched.Documents[0].ID != document.ID {
		t.Fatalf("search page=%+v error=%v", searched, err)
	}
	crossOwnerSearch, err := repository.List(ctx, lid.NewUser(), DocumentListFilter{Search: "repository"})
	if err != nil || len(crossOwnerSearch.Documents) != 0 {
		t.Fatalf("cross-owner search=%+v error=%v", crossOwnerSearch, err)
	}

	input := model.DocumentInput{
		Title: document.Title, Customer: document.Customer, IssueDate: document.IssueDate,
		Currency: document.Currency, LineItems: lineInputsForRepository(document),
	}
	input.Title = "Updated repository test"
	if err := document.ReplaceDraft(input, document.UpdatedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repository.ReplaceDraft(ctx, document, 1); err != nil {
		t.Fatal(err)
	}
	if err := repository.ReplaceDraft(ctx, document, 1); !repositoryHasCode(err, buhari.CodeDocumentVersionConflict) {
		t.Fatalf("stale replacement error = %v", err)
	}

	expected := document.Version
	if err := document.Archive(document.UpdatedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repository.Archive(ctx, document, expected); err != nil {
		t.Fatal(err)
	}
	expected = document.Version
	if err := document.Restore(document.UpdatedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repository.Restore(ctx, document, expected); err != nil {
		t.Fatal(err)
	}
	expected = document.Version
	if err := document.Finalize(document.UpdatedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repository.Finalize(ctx, document, expected); err != nil {
		t.Fatal(err)
	}

	loaded, err = repository.FindByReference(ctx, document.OwnerID, document.Reference)
	if err != nil || loaded.Status != model.DocumentFinalized {
		t.Fatalf("finalized lookup status=%s error=%v", loaded.Status, err)
	}
}

func TestDocumentRepositoryConditionalWritesLeaveStoredDocumentUnchanged(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repository := NewDocumentRepository(newMongoTestDatabase(t))

	assertUnchanged := func(t *testing.T, ownerID, documentID string, before model.Document) {
		t.Helper()
		after, err := repository.FindByID(ctx, ownerID, documentID)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(after, before) {
			t.Fatalf("failed conditional write changed stored document:\nbefore=%+v\nafter=%+v", before, after)
		}
	}
	create := func(t *testing.T) model.Document {
		t.Helper()
		document := repositoryTestDocument(t)
		if err := repository.Create(ctx, document); err != nil {
			t.Fatal(err)
		}
		stored, err := repository.FindByID(ctx, document.OwnerID, document.ID)
		if err != nil {
			t.Fatal(err)
		}
		return stored
	}
	advanceDraft := func(t *testing.T, document model.Document, count int) model.Document {
		t.Helper()
		for index := 0; index < count; index++ {
			input := model.DocumentInput{Title: document.Title, Customer: document.Customer, IssueDate: document.IssueDate, Currency: document.Currency, LineItems: lineInputsForRepository(document)}
			input.Title = document.Title + " updated"
			if err := document.ReplaceDraft(input, document.UpdatedAt.Add(time.Hour)); err != nil {
				t.Fatal(err)
			}
		}
		return document
	}

	t.Run("wrong owner", func(t *testing.T) {
		stored := create(t)
		candidate := advanceDraft(t, stored, 1)
		candidate.OwnerID = lid.NewUser()
		if err := repository.ReplaceDraft(ctx, candidate, stored.Version); !repositoryHasCode(err, buhari.CodeNotFound) {
			t.Fatalf("wrong-owner write error = %v", err)
		}
		assertUnchanged(t, stored.OwnerID, stored.ID, stored)
	})

	t.Run("stale version", func(t *testing.T) {
		stored := create(t)
		winner := advanceDraft(t, stored, 1)
		if err := repository.ReplaceDraft(ctx, winner, stored.Version); err != nil {
			t.Fatal(err)
		}
		beforeFailure, err := repository.FindByID(ctx, stored.OwnerID, stored.ID)
		if err != nil {
			t.Fatal(err)
		}
		stale := advanceDraft(t, stored, 1)
		if err := repository.ReplaceDraft(ctx, stale, stored.Version); !repositoryHasCode(err, buhari.CodeDocumentVersionConflict) {
			t.Fatalf("stale write error = %v", err)
		}
		assertUnchanged(t, stored.OwnerID, stored.ID, beforeFailure)
	})

	t.Run("finalized status", func(t *testing.T) {
		stored := create(t)
		finalized := stored
		if err := finalized.Finalize(finalized.UpdatedAt.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		if err := repository.Finalize(ctx, finalized, stored.Version); err != nil {
			t.Fatal(err)
		}
		beforeFailure, err := repository.FindByID(ctx, stored.OwnerID, stored.ID)
		if err != nil {
			t.Fatal(err)
		}
		candidate := advanceDraft(t, stored, 2)
		if err := repository.ReplaceDraft(ctx, candidate, beforeFailure.Version); !repositoryHasCode(err, buhari.CodeDocumentFinalized) {
			t.Fatalf("finalized write error = %v", err)
		}
		assertUnchanged(t, stored.OwnerID, stored.ID, beforeFailure)
	})

	t.Run("archived state", func(t *testing.T) {
		stored := create(t)
		archived := stored
		if err := archived.Archive(archived.UpdatedAt.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		if err := repository.Archive(ctx, archived, stored.Version); err != nil {
			t.Fatal(err)
		}
		beforeFailure, err := repository.FindByID(ctx, stored.OwnerID, stored.ID)
		if err != nil {
			t.Fatal(err)
		}
		candidate := advanceDraft(t, stored, 2)
		if err := repository.ReplaceDraft(ctx, candidate, beforeFailure.Version); !repositoryHasCode(err, buhari.CodeDocumentArchived) {
			t.Fatalf("archived write error = %v", err)
		}
		assertUnchanged(t, stored.OwnerID, stored.ID, beforeFailure)
	})
}

func lineInputsForRepository(document model.Document) []calculations.LineInput {
	inputs := make([]calculations.LineInput, len(document.LineItems))
	for index := range document.LineItems {
		inputs[index] = document.LineItems[index].LineInput
	}
	return inputs
}
