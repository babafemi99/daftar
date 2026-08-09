package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/calculations"
	"github.com/babafemi99/daftar/backend/internal/mercury"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/service"
)

type fakeReportService struct {
	dateRange service.ReportRange
	result    service.SummaryReport
	calls     int
}

func (reports *fakeReportService) Summary(_ context.Context, dateRange service.ReportRange) (*service.SummaryReport, error) {
	reports.calls++
	reports.dateRange = dateRange
	return &reports.result, nil
}

func TestReportSummaryRequiresAuthentication(t *testing.T) {
	api := configuredReportAPI(t, &fakeReportService{})
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/reports/summary?from=2026-08-01&to=2026-08-31", nil))
	assertErrorResponse(t, response, http.StatusUnauthorized, buhari.CodeUnauthorized)
}

func TestReportSummaryValidatesDates(t *testing.T) {
	tests := []struct {
		name  string
		query string
		paths []string
	}{
		{name: "both required", query: "", paths: []string{"from", "to"}},
		{name: "malformed from", query: "from=08-01-2026&to=2026-08-31", paths: []string{"from"}},
		{name: "malformed to", query: "from=2026-08-01&to=2026-02-30", paths: []string{"to"}},
		{name: "reversed", query: "from=2026-09-01&to=2026-08-31", paths: []string{"from"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeReportService{}
			response := serveDocumentRequest(t, configuredReportAPI(t, fake), http.MethodGet, "/api/v1/reports/summary?"+test.query, "", "")
			if response.Code != http.StatusUnprocessableEntity || fake.calls != 0 {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, fake.calls, response.Body.String())
			}
			var body struct {
				Error struct {
					Code    buhari.Code `json:"code"`
					Details struct {
						Fields []buhari.FieldError `json:"fields"`
					} `json:"details"`
				} `json:"error"`
			}
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Error.Code != buhari.CodeInvalidFilter || len(body.Error.Details.Fields) != len(test.paths) {
				t.Fatalf("error=%+v", body.Error)
			}
			for index, path := range test.paths {
				if body.Error.Details.Fields[index].Path != path {
					t.Fatalf("fields=%+v", body.Error.Details.Fields)
				}
			}
		})
	}
}

func TestReportSummaryFormatsCurrenciesAndEmptyResult(t *testing.T) {
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	fake := &fakeReportService{result: service.SummaryReport{
		From: from, To: to, DocumentCount: 2,
		Currencies: []mercury.CurrencySummary{{
			Currency: model.CurrencyUSD, DocumentCount: 2, Subtotal: 45_000, TotalDiscount: 4_000,
			TotalTax: 1_150, GrandTotal: 42_150,
			TaxBreakdown: []calculations.TaxBreakdown{{Rate: 50_000, TaxableAmountMinor: 23_000, TaxAmountMinor: 1_150}},
		}},
	}}
	response := serveDocumentRequest(t, configuredReportAPI(t, fake), http.MethodGet, "/api/v1/reports/summary?from=2026-08-01&to=2026-08-31", "", "")
	if response.Code != http.StatusOK || fake.calls != 1 || fake.dateRange.From != from || fake.dateRange.To != to {
		t.Fatalf("status=%d calls=%d range=%+v body=%s", response.Code, fake.calls, fake.dateRange, response.Body.String())
	}
	var envelope struct {
		Data summaryReportResponse `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data.Currencies) != 1 || envelope.Data.Currencies[0].Subtotal != "450.00" || envelope.Data.Currencies[0].TotalDiscount != "40.00" || envelope.Data.Currencies[0].TotalTax != "11.50" || envelope.Data.Currencies[0].GrandTotal != "421.50" {
		t.Fatalf("response=%+v", envelope.Data)
	}

	empty := &fakeReportService{result: service.SummaryReport{From: from, To: to, Currencies: []mercury.CurrencySummary{}}}
	emptyResponse := serveDocumentRequest(t, configuredReportAPI(t, empty), http.MethodGet, "/api/v1/reports/summary?from=2026-08-01&to=2026-08-31", "", "")
	var emptyEnvelope struct {
		Data summaryReportResponse `json:"data"`
	}
	if err := json.NewDecoder(emptyResponse.Body).Decode(&emptyEnvelope); err != nil {
		t.Fatal(err)
	}
	if emptyEnvelope.Data.DocumentCount != 0 || emptyEnvelope.Data.Currencies == nil || len(emptyEnvelope.Data.Currencies) != 0 {
		t.Fatalf("empty response=%+v", emptyEnvelope.Data)
	}
}

func configuredReportAPI(t *testing.T, reports Reports) *API {
	t.Helper()
	api := configuredDocumentAPI(t, &fakeDocuments{document: webTestDocument(t)})
	if err := api.ConfigureReports(reports); err != nil {
		t.Fatal(err)
	}
	return api
}
