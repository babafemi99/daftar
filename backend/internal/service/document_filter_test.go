package service

import (
	"errors"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/mercury"
	"github.com/babafemi99/daftar/backend/internal/model"
)

func TestValidateDocumentListFilter(t *testing.T) {
	unsupportedStatus := model.DocumentStatus("deleted")
	unsupportedCurrency := model.Currency("BTC")
	from := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		filter mercury.DocumentListFilter
		path   string
	}{
		{name: "unsupported status", filter: mercury.DocumentListFilter{Status: &unsupportedStatus}, path: "status"},
		{name: "unsupported currency", filter: mercury.DocumentListFilter{Currency: &unsupportedCurrency}, path: "currency"},
		{name: "reversed date range", filter: mercury.DocumentListFilter{IssueFrom: &from, IssueTo: &to}, path: "from"},
		{name: "negative limit", filter: mercury.DocumentListFilter{Limit: -1}, path: "limit"},
		{name: "limit too large", filter: mercury.DocumentListFilter{Limit: mercury.MaximumDocumentLimit + 1}, path: "limit"},
		{name: "negative page", filter: mercury.DocumentListFilter{Page: -1}, path: "page"},
		{name: "unsupported sort", filter: mercury.DocumentListFilter{Sort: "total_desc"}, path: "sort"},
		{name: "search too short", filter: mercury.DocumentListFilter{Search: "a"}, path: "search"},
		{name: "search too long", filter: mercury.DocumentListFilter{Search: string(make([]rune, 101))}, path: "search"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateDocumentListFilter(test.filter)
			var appErr *buhari.Error
			if !errors.As(err, &appErr) || appErr.Code != buhari.CodeInvalidFilter {
				t.Fatalf("error = %v, want %s", err, buhari.CodeInvalidFilter)
			}
			if len(appErr.Fields) != 1 || appErr.Fields[0].Path != test.path || appErr.Fields[0].Code != buhari.CodeInvalidFilter {
				t.Fatalf("fields = %+v", appErr.Fields)
			}
		})
	}
}

func TestValidateDocumentListFilterAcceptsDefaultsAndCurrentSort(t *testing.T) {
	status := model.DocumentDraft
	currency := model.CurrencyUSD
	from := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	tests := []mercury.DocumentListFilter{
		{},
		{Status: &status, Currency: &currency, IssueFrom: &from, IssueTo: &to, Limit: mercury.MaximumDocumentLimit, Sort: mercury.DocumentSortIssueDateDesc},
	}
	for _, filter := range tests {
		if err := validateDocumentListFilter(filter); err != nil {
			t.Fatalf("valid filter rejected: %v", err)
		}
	}
}

func TestDocumentServiceRejectsInvalidFilterBeforeRepositoryAccess(t *testing.T) {
	service, repository, _, ctx := newLineServiceDependencies(t, nil)
	status := model.DocumentStatus("deleted")
	_, err := service.List(ctx, mercury.DocumentListFilter{Status: &status})
	var appErr *buhari.Error
	if !errors.As(err, &appErr) || appErr.Code != buhari.CodeInvalidFilter {
		t.Fatalf("error = %v, want %s", err, buhari.CodeInvalidFilter)
	}
	if repository.listCalls != 0 {
		t.Fatalf("repository list calls = %d", repository.listCalls)
	}
}
