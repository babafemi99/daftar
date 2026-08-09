package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/calculations"
	"github.com/babafemi99/daftar/backend/internal/cfg"
	"github.com/babafemi99/daftar/backend/internal/mercury"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
	"github.com/babafemi99/daftar/backend/internal/pkg/requestctx"
	"github.com/babafemi99/daftar/backend/internal/sys"
)

type seedLifecycle uint8

const (
	seedDraft seedLifecycle = iota
	seedFinalized
	seedArchived
)

type seedDocument struct {
	input     model.DocumentInput
	lifecycle seedLifecycle
}

type seedRuntime interface {
	EnsureUser(context.Context, cfg.BootstrapUser) (model.User, error)
	ListDocuments(context.Context, model.User) ([]model.Document, error)
	CreateDocument(context.Context, model.User, model.DocumentInput) (model.Document, error)
	FinalizeDocument(context.Context, model.User, model.Document) (model.Document, error)
	ArchiveDocument(context.Context, model.User, model.Document) (model.Document, error)
}

func seedDemo(ctx context.Context, dependencies *sys.Dependencies, account cfg.BootstrapUser) error {
	if dependencies == nil || dependencies.UserService == nil || dependencies.DocumentService == nil || dependencies.Repositories == nil {
		return errors.New("seed: application dependencies are unavailable")
	}
	return runSeed(ctx, &applicationSeedRuntime{dependencies: dependencies}, account, time.Now().UTC())
}

func runSeed(ctx context.Context, runtime seedRuntime, account cfg.BootstrapUser, now time.Time) error {
	if runtime == nil {
		return errors.New("seed: runtime is required")
	}
	if err := validateSeedAccount(account); err != nil {
		return err
	}
	user, err := runtime.EnsureUser(ctx, account)
	if err != nil {
		return fmt.Errorf("seed: ensure reviewer user: %w", err)
	}
	existing, err := runtime.ListDocuments(ctx, user)
	if err != nil {
		return fmt.Errorf("seed: list reviewer documents: %w", err)
	}
	byTitle := make(map[string]model.Document, len(existing))
	for _, document := range existing {
		byTitle[document.Title] = document
	}

	for _, specification := range demoDocuments(now) {
		document, exists := byTitle[specification.input.Title]
		if !exists {
			document, err = runtime.CreateDocument(ctx, user, specification.input)
			if err != nil {
				return fmt.Errorf("seed: create %q: %w", specification.input.Title, err)
			}
		}
		switch specification.lifecycle {
		case seedDraft:
			if document.Status != model.DocumentDraft || document.ArchivedAt != nil {
				return fmt.Errorf("seed: %q exists but is not an active draft", document.Title)
			}
		case seedFinalized:
			if document.Status == model.DocumentDraft && document.ArchivedAt == nil {
				if _, err := runtime.FinalizeDocument(ctx, user, document); err != nil {
					return fmt.Errorf("seed: finalize %q: %w", document.Title, err)
				}
			} else if document.Status != model.DocumentFinalized {
				return fmt.Errorf("seed: %q exists in an incompatible lifecycle state", document.Title)
			}
		case seedArchived:
			if document.Status != model.DocumentDraft {
				return fmt.Errorf("seed: %q exists but is not a draft", document.Title)
			}
			if document.ArchivedAt == nil {
				if _, err := runtime.ArchiveDocument(ctx, user, document); err != nil {
					return fmt.Errorf("seed: archive %q: %w", document.Title, err)
				}
			}
		}
	}
	return nil
}

func validateSeedAccount(account cfg.BootstrapUser) error {
	if strings.TrimSpace(account.Email) == "" || len(account.Password) < 8 || strings.TrimSpace(account.FirstName) == "" || strings.TrimSpace(account.LastName) == "" {
		return errors.New("seed: DAFTAR_BOOTSTRAP_USER_EMAIL, PASSWORD, FIRST_NAME, and LAST_NAME are required")
	}
	return nil
}

func demoDocuments(now time.Time) []seedDocument {
	issueDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	fixed := calculations.Discount{Type: calculations.DiscountFixed, Value: 1_000}
	percentage := calculations.Discount{Type: calculations.DiscountPercentage, Value: 100_000}
	mixedLines := []calculations.LineInput{
		{Description: "Site assessment and planning", Quantity: 2, UnitPriceMinor: 10_000, Discount: &fixed, TaxRate: 50_000},
		{Description: "Materials and installation", Quantity: 3, UnitPriceMinor: 7_500, Discount: &percentage, TaxRate: 75_000},
		{Description: "Project handover", Quantity: 1, UnitPriceMinor: 5_000, TaxRate: 0},
	}
	return []seedDocument{
		{input: model.DocumentInput{Title: "Daftar Demo — Empty Draft", Customer: "Daftar Demo Company", IssueDate: issueDate, Currency: model.CurrencyUSD}, lifecycle: seedDraft},
		{input: model.DocumentInput{Title: "Daftar Demo — Mixed Taxes & Discounts", Customer: "Daftar Demo Company", IssueDate: issueDate, Currency: model.CurrencyUSD, LineItems: cloneSeedLines(mixedLines)}, lifecycle: seedDraft},
		{input: model.DocumentInput{Title: "Daftar Demo — Finalized Invoice", Customer: "CrossVal Reviewer", IssueDate: issueDate, Currency: model.CurrencyUSD, LineItems: cloneSeedLines(mixedLines)}, lifecycle: seedFinalized},
		{input: model.DocumentInput{Title: "Daftar Demo — Archived Draft", Customer: "Daftar Demo Company", IssueDate: issueDate, Currency: model.CurrencyNGN, LineItems: []calculations.LineInput{{Description: "Archived demonstration line", Quantity: 1, UnitPriceMinor: 25_000, TaxRate: 75_000}}}, lifecycle: seedArchived},
	}
}

func cloneSeedLines(lines []calculations.LineInput) []calculations.LineInput {
	result := make([]calculations.LineInput, len(lines))
	for index, line := range lines {
		result[index] = line
		if line.Discount != nil {
			discount := *line.Discount
			result[index].Discount = &discount
		}
	}
	return result
}

type applicationSeedRuntime struct{ dependencies *sys.Dependencies }

func (runtime *applicationSeedRuntime) EnsureUser(ctx context.Context, account cfg.BootstrapUser) (model.User, error) {
	user, err := runtime.dependencies.Repositories.User().FindByEmail(ctx, account.Email)
	if err == nil {
		_, loginErr := runtime.dependencies.UserService.Login(ctx, &model.LoginRequest{Email: account.Email, Password: account.Password})
		if loginErr != nil {
			return model.User{}, errors.New("seed account already exists with a different password")
		}
		return user, nil
	}
	var appErr *buhari.Error
	if !errors.As(err, &appErr) || appErr.Code != buhari.CodeNotFound {
		return model.User{}, err
	}
	created, err := runtime.dependencies.UserService.Register(ctx, &model.CreateUserRequest{Email: account.Email, Password: account.Password, FirstName: account.FirstName, LastName: account.LastName})
	if err != nil {
		return model.User{}, err
	}
	return *created, nil
}

func (runtime *applicationSeedRuntime) ListDocuments(ctx context.Context, user model.User) ([]model.Document, error) {
	result := make([]model.Document, 0, 4)
	for _, archived := range []bool{false, true} {
		page, err := runtime.dependencies.DocumentService.List(seedContext(ctx, user), mercury.DocumentListFilter{Search: "Daftar Demo", Archived: &archived, Limit: mercury.MaximumDocumentLimit})
		if err != nil {
			return nil, err
		}
		result = append(result, page.Documents...)
	}
	return result, nil
}

func (runtime *applicationSeedRuntime) CreateDocument(ctx context.Context, user model.User, input model.DocumentInput) (model.Document, error) {
	document, err := runtime.dependencies.DocumentService.Create(seedContext(ctx, user), input)
	if err != nil {
		return model.Document{}, err
	}
	return *document, nil
}

func (runtime *applicationSeedRuntime) FinalizeDocument(ctx context.Context, user model.User, document model.Document) (model.Document, error) {
	finalized, err := runtime.dependencies.DocumentService.Finalize(seedContext(ctx, user), document.ID, document.Version)
	if err != nil {
		return model.Document{}, err
	}
	return *finalized, nil
}

func (runtime *applicationSeedRuntime) ArchiveDocument(ctx context.Context, user model.User, document model.Document) (model.Document, error) {
	archived, err := runtime.dependencies.DocumentService.Archive(seedContext(ctx, user), document.ID, document.Version)
	if err != nil {
		return model.Document{}, err
	}
	return *archived, nil
}

func seedContext(ctx context.Context, user model.User) context.Context {
	ctx = requestctx.WithExecutor(ctx, model.Executor{ID: model.UserID(user.ID), Email: user.Email, Role: "user"})
	return requestctx.WithRequestID(ctx, lid.NewRequest())
}
