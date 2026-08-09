package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/mercury"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
	"github.com/babafemi99/daftar/backend/internal/pkg/requestctx"
)

type fakeReports struct {
	ownerID string
	from    time.Time
	to      time.Time
	result  []mercury.CurrencySummary
	calls   int
}

func (repository *fakeReports) Summary(_ context.Context, ownerID string, from, to time.Time) ([]mercury.CurrencySummary, error) {
	repository.calls++
	repository.ownerID, repository.from, repository.to = ownerID, from, to
	return repository.result, nil
}

func TestReportServiceValidatesRangeBeforeRepositoryAccess(t *testing.T) {
	repository := &fakeReports{}
	reports, err := NewReportService(repository)
	if err != nil {
		t.Fatal(err)
	}
	ownerID := lid.NewUser()
	ctx := requestctx.WithExecutor(context.Background(), model.Executor{ID: model.UserID(ownerID)})
	valid := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		dateRange ReportRange
		path      string
	}{
		{name: "missing from", dateRange: ReportRange{To: valid}, path: "from"},
		{name: "missing to", dateRange: ReportRange{From: valid}, path: "to"},
		{name: "from after to", dateRange: ReportRange{From: valid.AddDate(0, 0, 1), To: valid}, path: "from"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := repository.calls
			_, err := reports.Summary(ctx, test.dateRange)
			var appErr *buhari.Error
			if !errors.As(err, &appErr) || appErr.Code != buhari.CodeInvalidFilter || len(appErr.Fields) == 0 || appErr.Fields[0].Path != test.path {
				t.Fatalf("error=%v", err)
			}
			if repository.calls != before {
				t.Fatalf("repository calls=%d want=%d", repository.calls, before)
			}
		})
	}
}

func TestReportServiceScopesOwnerAndCountsCurrencies(t *testing.T) {
	repository := &fakeReports{result: []mercury.CurrencySummary{{Currency: model.CurrencyUSD, DocumentCount: 2}, {Currency: model.CurrencyEUR, DocumentCount: 1}}}
	reports, err := NewReportService(repository)
	if err != nil {
		t.Fatal(err)
	}
	ownerID := lid.NewUser()
	ctx := requestctx.WithExecutor(context.Background(), model.Executor{ID: model.UserID(ownerID)})
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	result, err := reports.Summary(ctx, ReportRange{From: from, To: to})
	if err != nil {
		t.Fatal(err)
	}
	if repository.ownerID != ownerID || repository.from != from || repository.to != to || result.DocumentCount != 3 {
		t.Fatalf("repository=%+v result=%+v", repository, result)
	}
}
