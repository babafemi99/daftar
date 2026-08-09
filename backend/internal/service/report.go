package service

import (
	"context"
	"errors"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/mercury"
)

var ErrReportRepositoryRequired = errors.New("service: report repository is required")

type ReportRange struct {
	From time.Time
	To   time.Time
}

type SummaryReport struct {
	From          time.Time
	To            time.Time
	DocumentCount int64
	Currencies    []mercury.CurrencySummary
}

type ReportService struct {
	reports mercury.Reports
}

func NewReportService(reports mercury.Reports) (*ReportService, error) {
	if reports == nil {
		return nil, ErrReportRepositoryRequired
	}
	return &ReportService{reports: reports}, nil
}

func (service *ReportService) Summary(ctx context.Context, dateRange ReportRange) (*SummaryReport, error) {
	ownerID, err := authenticatedOwner(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateReportRange(dateRange); err != nil {
		return nil, err
	}
	currencies, err := service.reports.Summary(ctx, ownerID, dateRange.From, dateRange.To)
	if err != nil {
		return nil, err
	}
	result := &SummaryReport{From: dateRange.From, To: dateRange.To, Currencies: currencies}
	for _, summary := range currencies {
		result.DocumentCount += summary.DocumentCount
	}
	return result, nil
}

func validateReportRange(dateRange ReportRange) error {
	fields := make([]buhari.FieldError, 0, 2)
	if !calendarDate(dateRange.From) {
		fields = append(fields, reportField("from", "From must be a valid UTC calendar date."))
	}
	if !calendarDate(dateRange.To) {
		fields = append(fields, reportField("to", "To must be a valid UTC calendar date."))
	}
	if len(fields) == 0 && dateRange.From.After(dateRange.To) {
		fields = append(fields, reportField("from", "From cannot be after to."))
	}
	if len(fields) > 0 {
		return buhari.InvalidFilter(fields...)
	}
	return nil
}

func calendarDate(value time.Time) bool {
	return !value.IsZero() && value.Location() == time.UTC && value.Hour() == 0 && value.Minute() == 0 && value.Second() == 0 && value.Nanosecond() == 0
}

func reportField(path, message string) buhari.FieldError {
	return buhari.FieldError{Path: path, Code: buhari.CodeInvalidFilter, Message: message}
}
