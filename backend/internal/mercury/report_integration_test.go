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

func TestReportRepositorySummaryIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database := newMongoTestDatabase(t)
	documents := NewDocumentRepository(database)
	reports := NewReportRepository(database)
	if err := reports.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}

	ownerID := lid.NewUser()
	otherOwnerID := lid.NewUser()
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	records := []model.Document{
		reportTestDocument(t, ownerID, "DOC-2026-000001", from, model.CurrencyUSD, true, []calculations.LineInput{{Description: "percentage", Quantity: 2, UnitPriceMinor: 10_000, Discount: &calculations.Discount{Type: calculations.DiscountPercentage, Value: 100_000}, TaxRate: 50_000}}),
		reportTestDocument(t, ownerID, "DOC-2026-000002", to, model.CurrencyUSD, true, []calculations.LineInput{{Description: "fixed", Quantity: 1, UnitPriceMinor: 10_000, Discount: &calculations.Discount{Type: calculations.DiscountFixed, Value: 2_000}, TaxRate: 100_000}}),
		reportTestDocument(t, ownerID, "DOC-2026-000003", from.AddDate(0, 0, 15), model.CurrencyEUR, true, []calculations.LineInput{{Description: "euro", Quantity: 1, UnitPriceMinor: 5_000, TaxRate: 50_000}}),
		reportTestDocument(t, ownerID, "DOC-2026-000004", from.AddDate(0, 0, -1), model.CurrencyUSD, true, []calculations.LineInput{{Description: "before", Quantity: 1, UnitPriceMinor: 99_999}}),
		reportTestDocument(t, ownerID, "DOC-2026-000005", to.AddDate(0, 0, 1), model.CurrencyUSD, true, []calculations.LineInput{{Description: "after", Quantity: 1, UnitPriceMinor: 99_999}}),
		reportTestDocument(t, ownerID, "DOC-2026-000006", from.AddDate(0, 0, 10), model.CurrencyUSD, false, []calculations.LineInput{{Description: "draft", Quantity: 1, UnitPriceMinor: 99_999}}),
		reportTestDocument(t, otherOwnerID, "DOC-2026-000001", from.AddDate(0, 0, 10), model.CurrencyUSD, true, []calculations.LineInput{{Description: "other owner", Quantity: 1, UnitPriceMinor: 99_999}}),
	}
	for _, document := range records {
		if err := documents.Create(ctx, document); err != nil {
			t.Fatal(err)
		}
	}

	summaries, err := reports.Summary(ctx, ownerID, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 2 || summaries[0].Currency != model.CurrencyEUR || summaries[1].Currency != model.CurrencyUSD {
		t.Fatalf("currencies=%+v", summaries)
	}
	eur := summaries[0]
	if eur.DocumentCount != 1 || eur.Subtotal != 5_000 || eur.TotalDiscount != 0 || eur.TotalTax != 250 || eur.GrandTotal != 5_250 {
		t.Fatalf("EUR summary=%+v", eur)
	}
	usd := summaries[1]
	if usd.DocumentCount != 2 || usd.Subtotal != 30_000 || usd.TotalDiscount != 4_000 || usd.TotalTax != 1_700 || usd.GrandTotal != 27_700 {
		t.Fatalf("USD summary=%+v", usd)
	}
	if len(usd.TaxBreakdown) != 2 || usd.TaxBreakdown[0] != (calculations.TaxBreakdown{Rate: 50_000, TaxableAmountMinor: 18_000, TaxAmountMinor: 900}) || usd.TaxBreakdown[1] != (calculations.TaxBreakdown{Rate: 100_000, TaxableAmountMinor: 8_000, TaxAmountMinor: 800}) {
		t.Fatalf("USD tax breakdown=%+v", usd.TaxBreakdown)
	}

	empty, err := reports.Summary(ctx, ownerID, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC))
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty summary=%+v error=%v", empty, err)
	}
}

func reportTestDocument(t *testing.T, ownerID, reference string, issueDate time.Time, currency model.Currency, finalized bool, lines []calculations.LineInput) model.Document {
	t.Helper()
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	document, err := model.NewDocument(ownerID, reference, model.DocumentInput{
		Title: "Report document", Customer: "CrossVal", IssueDate: issueDate,
		Currency: currency, LineItems: lines,
	}, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	if finalized {
		if err := document.Finalize(createdAt.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	return document
}
