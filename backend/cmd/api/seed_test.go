package main

import (
	"context"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/cfg"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
)

type fakeSeedRuntime struct {
	user          model.User
	documents     map[string]model.Document
	created       int
	finalized     int
	archived      int
	ensureUserRun int
}

func newFakeSeedRuntime() *fakeSeedRuntime {
	return &fakeSeedRuntime{user: model.User{ID: lid.NewUser(), Email: "reviewer@daftar.local"}, documents: make(map[string]model.Document)}
}

func (runtime *fakeSeedRuntime) EnsureUser(context.Context, cfg.BootstrapUser) (model.User, error) {
	runtime.ensureUserRun++
	return runtime.user, nil
}

func (runtime *fakeSeedRuntime) ListDocuments(context.Context, model.User) ([]model.Document, error) {
	documents := make([]model.Document, 0, len(runtime.documents))
	for _, document := range runtime.documents {
		documents = append(documents, document)
	}
	return documents, nil
}

func (runtime *fakeSeedRuntime) CreateDocument(_ context.Context, user model.User, input model.DocumentInput) (model.Document, error) {
	runtime.created++
	document := model.Document{ID: lid.NewDocument(), OwnerID: user.ID, Title: input.Title, Status: model.DocumentDraft, Version: 1}
	runtime.documents[input.Title] = document
	return document, nil
}

func (runtime *fakeSeedRuntime) FinalizeDocument(_ context.Context, _ model.User, document model.Document) (model.Document, error) {
	runtime.finalized++
	now := time.Now().UTC()
	document.Status, document.FinalizedAt, document.Version = model.DocumentFinalized, &now, document.Version+1
	runtime.documents[document.Title] = document
	return document, nil
}

func (runtime *fakeSeedRuntime) ArchiveDocument(_ context.Context, _ model.User, document model.Document) (model.Document, error) {
	runtime.archived++
	now := time.Now().UTC()
	document.ArchivedAt, document.Version = &now, document.Version+1
	runtime.documents[document.Title] = document
	return document, nil
}

func TestRunSeedIsIdempotent(t *testing.T) {
	runtime := newFakeSeedRuntime()
	account := cfg.BootstrapUser{Email: runtime.user.Email, Password: "strong-password", FirstName: "Ada", LastName: "Reviewer"}
	now := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)

	if err := runSeed(context.Background(), runtime, account, now); err != nil {
		t.Fatalf("runSeed() first call: %v", err)
	}
	if err := runSeed(context.Background(), runtime, account, now); err != nil {
		t.Fatalf("runSeed() second call: %v", err)
	}
	if runtime.created != 4 || len(runtime.documents) != 4 {
		t.Fatalf("created=%d documents=%d, want 4", runtime.created, len(runtime.documents))
	}
	if runtime.finalized != 1 || runtime.archived != 1 {
		t.Fatalf("finalized=%d archived=%d, want 1 each", runtime.finalized, runtime.archived)
	}
	if runtime.ensureUserRun != 2 {
		t.Fatalf("ensure user calls=%d, want 2", runtime.ensureUserRun)
	}
}

func TestRunSeedRequiresReviewerCredentials(t *testing.T) {
	if err := runSeed(context.Background(), newFakeSeedRuntime(), cfg.BootstrapUser{}, time.Now()); err == nil {
		t.Fatal("runSeed() error = nil")
	}
}
