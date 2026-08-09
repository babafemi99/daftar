package web

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/mercury"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/service"
)

var ErrReportServiceRequired = errors.New("web: report service is required")

type Reports interface {
	Summary(context.Context, service.ReportRange) (*service.SummaryReport, error)
}

func (a *API) ConfigureReports(reports Reports) error {
	if reports == nil {
		return ErrReportServiceRequired
	}
	a.reports = reports
	return nil
}

type summaryReportResponse struct {
	From          string                    `json:"from"`
	To            string                    `json:"to"`
	DocumentCount int64                     `json:"documentCount"`
	Currencies    []currencySummaryResponse `json:"currencies"`
}

type currencySummaryResponse struct {
	Currency      model.Currency         `json:"currency"`
	DocumentCount int64                  `json:"documentCount"`
	Subtotal      string                 `json:"subtotal"`
	TotalDiscount string                 `json:"totalDiscount"`
	TotalTax      string                 `json:"totalTax"`
	GrandTotal    string                 `json:"grandTotal"`
	TaxBreakdown  []taxBreakdownResponse `json:"taxBreakdown"`
}

func (a *API) reportSummary(w http.ResponseWriter, r *http.Request) {
	dateRange, err := parseReportRange(r)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	report, err := a.reportService().Summary(r.Context(), dateRange)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	Response(r.Context(), w, http.StatusOK, newSummaryReportResponse(*report))
}

func parseReportRange(r *http.Request) (service.ReportRange, error) {
	query := r.URL.Query()
	from, fromErr := parseRequiredReportDate(query.Get("from"), "from")
	to, toErr := parseRequiredReportDate(query.Get("to"), "to")
	fields := make([]buhari.FieldError, 0, 2)
	if fromErr != nil {
		fields = append(fields, *fromErr)
	}
	if toErr != nil {
		fields = append(fields, *toErr)
	}
	if len(fields) == 0 && from.After(to) {
		fields = append(fields, buhari.FieldError{Path: "from", Code: buhari.CodeInvalidFilter, Message: "From cannot be after to."})
	}
	if len(fields) > 0 {
		return service.ReportRange{}, buhari.InvalidFilter(fields...)
	}
	return service.ReportRange{From: from, To: to}, nil
}

func parseRequiredReportDate(value, path string) (time.Time, *buhari.FieldError) {
	if value == "" {
		return time.Time{}, &buhari.FieldError{Path: path, Code: buhari.CodeInvalidFilter, Message: path + " is required."}
	}
	date, err := time.Parse("2006-01-02", value)
	if err != nil || date.Format("2006-01-02") != value {
		return time.Time{}, &buhari.FieldError{Path: path, Code: buhari.CodeInvalidFilter, Message: path + " must use YYYY-MM-DD format."}
	}
	return date.UTC(), nil
}

func newSummaryReportResponse(report service.SummaryReport) summaryReportResponse {
	response := summaryReportResponse{
		From: report.From.Format("2006-01-02"), To: report.To.Format("2006-01-02"),
		DocumentCount: report.DocumentCount,
		Currencies:    make([]currencySummaryResponse, len(report.Currencies)),
	}
	for index, summary := range report.Currencies {
		response.Currencies[index] = newCurrencySummaryResponse(summary)
	}
	return response
}

func newCurrencySummaryResponse(summary mercury.CurrencySummary) currencySummaryResponse {
	return currencySummaryResponse{
		Currency: summary.Currency, DocumentCount: summary.DocumentCount,
		Subtotal: formatMoney(summary.Subtotal), TotalDiscount: formatMoney(summary.TotalDiscount),
		TotalTax: formatMoney(summary.TotalTax), GrandTotal: formatMoney(summary.GrandTotal),
		TaxBreakdown: newTaxBreakdownResponses(summary.TaxBreakdown),
	}
}

type unavailableReports struct{}

func (unavailableReports) Summary(context.Context, service.ReportRange) (*service.SummaryReport, error) {
	return nil, errors.New("report service not configured")
}

func (a *API) reportService() Reports {
	if a.reports == nil {
		return unavailableReports{}
	}
	return a.reports
}
