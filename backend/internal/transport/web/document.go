package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/calculations"
	"github.com/babafemi99/daftar/backend/internal/mercury"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

var ErrDocumentServiceRequired = errors.New("web: document service is required")

type Documents interface {
	Create(context.Context, model.DocumentInput) (*model.Document, error)
	Preview(context.Context, service.CalculationPreviewInput) (*calculations.DocumentResult, error)
	GetByID(context.Context, string) (*model.Document, error)
	List(context.Context, mercury.DocumentListFilter) (mercury.DocumentPage, error)
	ReplaceDraft(context.Context, string, int64, model.DocumentInput) (*model.Document, error)
	AddLine(context.Context, string, int64, calculations.LineInput) (*model.Document, error)
	UpdateLine(context.Context, string, string, int64, calculations.LineInput) (*model.Document, error)
	DeleteLine(context.Context, string, string, int64) (*model.Document, error)
	ReorderLines(context.Context, string, int64, []string) (*model.Document, error)
	Archive(context.Context, string, int64) (*model.Document, error)
	Restore(context.Context, string, int64) (*model.Document, error)
	Finalize(context.Context, string, int64) (*model.Document, error)
	Duplicate(context.Context, string, service.DuplicateDocumentInput) (*model.Document, error)
}

func (a *API) ConfigureDocuments(documents Documents) error {
	if documents == nil {
		return ErrDocumentServiceRequired
	}
	a.documents = documents
	return nil
}

type documentRequest struct {
	Title     string         `json:"title"`
	Customer  string         `json:"customer"`
	IssueDate string         `json:"issueDate"`
	Currency  model.Currency `json:"currency"`
	LineItems []lineRequest  `json:"lineItems"`
}

type previewRequest struct {
	Currency  model.Currency `json:"currency"`
	LineItems []lineRequest  `json:"lineItems"`
}

type lineRequest struct {
	ID          string           `json:"id,omitempty"`
	Description string           `json:"description"`
	Quantity    int64            `json:"quantity"`
	UnitPrice   string           `json:"unitPrice"`
	Discount    *discountRequest `json:"discount,omitempty"`
	TaxRate     string           `json:"taxRate"`
}

type discountRequest struct {
	Type  calculations.DiscountType `json:"type"`
	Value string                    `json:"value"`
}

type reorderLinesRequest struct {
	LineItemIDs []string `json:"lineItemIds"`
}

type duplicateDocumentRequest struct {
	Title     *string `json:"title,omitempty"`
	IssueDate *string `json:"issueDate,omitempty"`
}

type documentResponse struct {
	ID                       string                 `json:"id"`
	Reference                string                 `json:"reference"`
	Title                    string                 `json:"title"`
	Customer                 string                 `json:"customer"`
	IssueDate                string                 `json:"issueDate"`
	Currency                 model.Currency         `json:"currency"`
	Status                   model.DocumentStatus   `json:"status"`
	Version                  int64                  `json:"version"`
	LineItems                []lineResponse         `json:"lineItems"`
	Totals                   totalsResponse         `json:"totals"`
	TaxBreakdown             []taxBreakdownResponse `json:"taxBreakdown"`
	CalculationPolicyVersion string                 `json:"calculationPolicyVersion"`
	FinalizedAt              *time.Time             `json:"finalizedAt"`
	ArchivedAt               *time.Time             `json:"archivedAt"`
	CreatedAt                time.Time              `json:"createdAt"`
	UpdatedAt                time.Time              `json:"updatedAt"`
}

type lineResponse struct {
	ID          string                 `json:"id,omitempty"`
	Description string                 `json:"description"`
	Quantity    calculations.Quantity  `json:"quantity"`
	UnitPrice   string                 `json:"unitPrice"`
	Discount    *discountResponse      `json:"discount,omitempty"`
	TaxRate     string                 `json:"taxRate"`
	Calculated  lineCalculatedResponse `json:"calculated"`
}

type discountResponse struct {
	Type  calculations.DiscountType `json:"type"`
	Value string                    `json:"value"`
}

type lineCalculatedResponse struct {
	SubtotalMinor         calculations.Money `json:"subtotalMinor"`
	Subtotal              string             `json:"subtotal"`
	DiscountAmountMinor   calculations.Money `json:"discountAmountMinor"`
	DiscountAmount        string             `json:"discountAmount"`
	DiscountedAmountMinor calculations.Money `json:"discountedAmountMinor"`
	DiscountedAmount      string             `json:"discountedAmount"`
	TaxAmountMinor        calculations.Money `json:"taxAmountMinor"`
	TaxAmount             string             `json:"taxAmount"`
	LineTotalMinor        calculations.Money `json:"lineTotalMinor"`
	LineTotal             string             `json:"lineTotal"`
}

type totalsResponse struct {
	SubtotalMinor   calculations.Money `json:"subtotalMinor"`
	Subtotal        string             `json:"subtotal"`
	DiscountMinor   calculations.Money `json:"discountMinor"`
	Discount        string             `json:"discount"`
	TaxMinor        calculations.Money `json:"taxMinor"`
	Tax             string             `json:"tax"`
	GrandTotalMinor calculations.Money `json:"grandTotalMinor"`
	GrandTotal      string             `json:"grandTotal"`
}

type taxBreakdownResponse struct {
	Rate          string `json:"rate"`
	TaxableAmount string `json:"taxableAmount"`
	TaxAmount     string `json:"taxAmount"`
}

type previewResponse struct {
	LineItems                []lineResponse         `json:"lineItems"`
	Totals                   totalsResponse         `json:"totals"`
	TaxBreakdown             []taxBreakdownResponse `json:"taxBreakdown"`
	CalculationPolicyVersion string                 `json:"calculationPolicyVersion"`
}

func (a *API) createDocument(w http.ResponseWriter, r *http.Request) {
	var request documentRequest
	if err := decodeJSON(w, r, &request); err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	input, err := request.documentInput(false)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	document, err := a.documentService().Create(r.Context(), input)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	respondDocument(r.Context(), w, http.StatusCreated, document)
}

func (a *API) getDocument(w http.ResponseWriter, r *http.Request) {
	document, err := a.documentService().GetByID(r.Context(), chi.URLParam(r, "documentId"))
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	respondDocument(r.Context(), w, http.StatusOK, document)
}

func (a *API) listDocuments(w http.ResponseWriter, r *http.Request) {
	filter, err := parseDocumentListFilter(r)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	page, err := a.documentService().List(r.Context(), filter)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	result := make([]documentResponse, len(page.Documents))
	for index := range page.Documents {
		result[index] = newDocumentResponse(page.Documents[index])
	}
	CollectionResponseWithPage(r.Context(), w, http.StatusOK, result, CollectionPage{Number: page.Page, Size: page.PageSize, TotalItems: page.TotalItems, TotalPages: page.TotalPages, HasMore: page.Page < page.TotalPages})
}

func (a *API) replaceDocument(w http.ResponseWriter, r *http.Request) {
	version, err := parseIfMatch(r)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	var request documentRequest
	if err := decodeJSON(w, r, &request); err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	input, err := request.documentInput(true)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	document, err := a.documentService().ReplaceDraft(r.Context(), chi.URLParam(r, "documentId"), version, input)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	respondDocument(r.Context(), w, http.StatusOK, document)
}

func (a *API) previewDocumentCalculation(w http.ResponseWriter, r *http.Request) {
	var request previewRequest
	if err := decodeJSON(w, r, &request); err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	lines, err := parseLineRequests(request.LineItems, false)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	result, err := a.documentService().Preview(r.Context(), service.CalculationPreviewInput{Currency: request.Currency, LineItems: lines})
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	Response(r.Context(), w, http.StatusOK, newPreviewResponse(*result))
}

func (a *API) addDocumentLine(w http.ResponseWriter, r *http.Request) {
	a.mutateLine(w, r, http.StatusCreated, func(version int64, input calculations.LineInput) (*model.Document, error) {
		return a.documentService().AddLine(r.Context(), chi.URLParam(r, "documentId"), version, input)
	})
}

func (a *API) updateDocumentLine(w http.ResponseWriter, r *http.Request) {
	a.mutateLine(w, r, http.StatusOK, func(version int64, input calculations.LineInput) (*model.Document, error) {
		return a.documentService().UpdateLine(r.Context(), chi.URLParam(r, "documentId"), chi.URLParam(r, "lineItemId"), version, input)
	})
}

func (a *API) mutateLine(w http.ResponseWriter, r *http.Request, status int, command func(int64, calculations.LineInput) (*model.Document, error)) {
	version, err := parseIfMatch(r)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	var request lineRequest
	if err := decodeJSON(w, r, &request); err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	if request.ID != "" {
		ResponseError(r.Context(), w, invalidInput("id", "Line item IDs are assigned by the server."))
		return
	}
	input, err := request.lineInput("", "")
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	document, err := command(version, input)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	respondDocument(r.Context(), w, status, document)
}

func (a *API) deleteDocumentLine(w http.ResponseWriter, r *http.Request) {
	version, err := parseIfMatch(r)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	document, err := a.documentService().DeleteLine(r.Context(), chi.URLParam(r, "documentId"), chi.URLParam(r, "lineItemId"), version)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	respondDocument(r.Context(), w, http.StatusOK, document)
}

func (a *API) reorderDocumentLines(w http.ResponseWriter, r *http.Request) {
	version, err := parseIfMatch(r)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	var request reorderLinesRequest
	if err := decodeJSON(w, r, &request); err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	document, err := a.documentService().ReorderLines(r.Context(), chi.URLParam(r, "documentId"), version, request.LineItemIDs)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	respondDocument(r.Context(), w, http.StatusOK, document)
}

func (a *API) archiveDocument(w http.ResponseWriter, r *http.Request) {
	a.mutateDocumentWithoutBody(w, r, a.documentService().Archive)
}

func (a *API) restoreDocument(w http.ResponseWriter, r *http.Request) {
	a.mutateDocumentWithoutBody(w, r, a.documentService().Restore)
}

func (a *API) finalizeDocument(w http.ResponseWriter, r *http.Request) {
	a.mutateDocumentWithoutBody(w, r, a.documentService().Finalize)
}

func (a *API) mutateDocumentWithoutBody(w http.ResponseWriter, r *http.Request, command func(context.Context, string, int64) (*model.Document, error)) {
	version, err := parseIfMatch(r)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	document, err := command(r.Context(), chi.URLParam(r, "documentId"), version)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	respondDocument(r.Context(), w, http.StatusOK, document)
}

func (a *API) duplicateDocument(w http.ResponseWriter, r *http.Request) {
	var request duplicateDocumentRequest
	if err := decodeOptionalJSON(w, r, &request); err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	overrides := service.DuplicateDocumentInput{Title: request.Title}
	if request.IssueDate != nil {
		date, err := parseDate(*request.IssueDate, "issueDate")
		if err != nil {
			ResponseError(r.Context(), w, err)
			return
		}
		overrides.IssueDate = &date
	}
	document, err := a.documentService().Duplicate(r.Context(), chi.URLParam(r, "documentId"), overrides)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	respondDocument(r.Context(), w, http.StatusCreated, document)
}

func (a *API) documentService() Documents {
	if a.documents == nil {
		return unavailableDocuments{}
	}
	return a.documents
}

type unavailableDocuments struct{}

func (unavailableDocuments) failure() error { return errors.New("document service not configured") }
func (u unavailableDocuments) Create(context.Context, model.DocumentInput) (*model.Document, error) {
	return nil, u.failure()
}
func (u unavailableDocuments) Preview(context.Context, service.CalculationPreviewInput) (*calculations.DocumentResult, error) {
	return nil, u.failure()
}
func (u unavailableDocuments) GetByID(context.Context, string) (*model.Document, error) {
	return nil, u.failure()
}
func (u unavailableDocuments) List(context.Context, mercury.DocumentListFilter) (mercury.DocumentPage, error) {
	return mercury.DocumentPage{}, u.failure()
}
func (u unavailableDocuments) ReplaceDraft(context.Context, string, int64, model.DocumentInput) (*model.Document, error) {
	return nil, u.failure()
}
func (u unavailableDocuments) AddLine(context.Context, string, int64, calculations.LineInput) (*model.Document, error) {
	return nil, u.failure()
}
func (u unavailableDocuments) UpdateLine(context.Context, string, string, int64, calculations.LineInput) (*model.Document, error) {
	return nil, u.failure()
}
func (u unavailableDocuments) DeleteLine(context.Context, string, string, int64) (*model.Document, error) {
	return nil, u.failure()
}
func (u unavailableDocuments) ReorderLines(context.Context, string, int64, []string) (*model.Document, error) {
	return nil, u.failure()
}
func (u unavailableDocuments) Archive(context.Context, string, int64) (*model.Document, error) {
	return nil, u.failure()
}
func (u unavailableDocuments) Restore(context.Context, string, int64) (*model.Document, error) {
	return nil, u.failure()
}
func (u unavailableDocuments) Finalize(context.Context, string, int64) (*model.Document, error) {
	return nil, u.failure()
}
func (u unavailableDocuments) Duplicate(context.Context, string, service.DuplicateDocumentInput) (*model.Document, error) {
	return nil, u.failure()
}

func (request documentRequest) documentInput(allowIDs bool) (model.DocumentInput, error) {
	issueDate, err := parseDate(request.IssueDate, "issueDate")
	if err != nil {
		return model.DocumentInput{}, err
	}
	lines, err := parseLineRequests(request.LineItems, allowIDs)
	if err != nil {
		return model.DocumentInput{}, err
	}
	return model.DocumentInput{Title: request.Title, Customer: request.Customer, IssueDate: issueDate, Currency: request.Currency, LineItems: lines}, nil
}

func parseLineRequests(requests []lineRequest, allowIDs bool) ([]calculations.LineInput, error) {
	lines := make([]calculations.LineInput, len(requests))
	for index, request := range requests {
		if !allowIDs && request.ID != "" {
			return nil, invalidInput("lineItems["+strconv.Itoa(index)+"].id", "Line item IDs are assigned by the server.")
		}
		line, err := request.lineInput(request.ID, "lineItems["+strconv.Itoa(index)+"]")
		if err != nil {
			return nil, err
		}
		lines[index] = line
	}
	return lines, nil
}

func (request lineRequest) lineInput(id, prefix string) (calculations.LineInput, error) {
	unitPrice, err := parseMoney(request.UnitPrice, nestedPath(prefix, "unitPrice"))
	if err != nil {
		return calculations.LineInput{}, err
	}
	taxRate, err := parseRate(request.TaxRate, nestedPath(prefix, "taxRate"))
	if err != nil {
		return calculations.LineInput{}, err
	}
	line := calculations.LineInput{ID: id, Description: request.Description, Quantity: calculations.Quantity(request.Quantity), UnitPriceMinor: unitPrice, TaxRate: taxRate}
	if request.Discount != nil {
		value, err := parseDiscountValue(*request.Discount, prefix)
		if err != nil {
			return calculations.LineInput{}, err
		}
		line.Discount = &calculations.Discount{Type: request.Discount.Type, Value: value}
	}
	return line, nil
}

func parseDiscountValue(discount discountRequest, prefix string) (int64, error) {
	switch discount.Type {
	case calculations.DiscountFixed:
		value, err := parseMoney(discount.Value, nestedPath(prefix, "discount.value"))
		return int64(value), err
	case calculations.DiscountPercentage:
		value, err := parsePercentage(discount.Value, nestedPath(prefix, "discount.value"), buhari.CodeInvalidDiscount)
		return int64(value), err
	default:
		return 0, buhari.Validation(buhari.FieldError{Path: nestedPath(prefix, "discount.type"), Code: buhari.CodeInvalidDiscountType, Message: "Discount type must be fixed or percentage."})
	}
}

func nestedPath(prefix, field string) string {
	if prefix == "" {
		return field
	}
	return prefix + "." + field
}

func parseMoney(value, path string) (calculations.Money, error) {
	minor, ok := parseUnsignedDecimal(value, 2, 100)
	if !ok || minor > math.MaxInt64 {
		return 0, buhari.Validation(buhari.FieldError{Path: path, Code: buhari.CodeInvalidMoneyFormat, Message: "Money must be a nonnegative decimal string with at most two decimal places."})
	}
	return calculations.Money(minor), nil
}

func parseRate(value, path string) (calculations.Rate, error) {
	return parsePercentage(value, path, buhari.CodeInvalidTaxRate)
}

func parsePercentage(value, path string, code buhari.Code) (calculations.Rate, error) {
	rate, ok := parseUnsignedDecimal(value, 4, 10_000)
	if !ok || rate > uint64(calculations.RateScale) {
		return 0, buhari.Validation(buhari.FieldError{Path: path, Code: code, Message: "Rate must be a decimal percentage between 0 and 100 with at most four decimal places."})
	}
	return calculations.Rate(rate), nil
}

func parseUnsignedDecimal(value string, maximumFraction, scale int) (uint64, bool) {
	if value == "" || strings.TrimSpace(value) != value || strings.HasPrefix(value, "+") || strings.HasPrefix(value, "-") {
		return 0, false
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" || (len(parts) == 2 && (parts[1] == "" || len(parts[1]) > maximumFraction)) {
		return 0, false
	}
	for _, part := range parts {
		for _, character := range part {
			if character < '0' || character > '9' {
				return 0, false
			}
		}
	}
	whole, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || whole > math.MaxUint64/uint64(scale) {
		return 0, false
	}
	result := whole * uint64(scale)
	if len(parts) == 2 {
		fraction := parts[1] + strings.Repeat("0", maximumFraction-len(parts[1]))
		fractionValue, err := strconv.ParseUint(fraction, 10, 64)
		if err != nil || result > math.MaxUint64-fractionValue {
			return 0, false
		}
		result += fractionValue
	}
	return result, true
}

func parseDate(value, path string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", value)
	if err != nil || date.Format("2006-01-02") != value {
		return time.Time{}, invalidInput(path, "Date must use YYYY-MM-DD format.")
	}
	return date.UTC(), nil
}

func parseIfMatch(r *http.Request) (int64, error) {
	value := strings.TrimSpace(r.Header.Get("If-Match"))
	if len(value) < 3 || value[0] != '"' || value[len(value)-1] != '"' {
		return 0, buhari.New(buhari.CodeInvalidIfMatch, "If-Match must contain a quoted positive document version.")
	}
	version, err := strconv.ParseInt(value[1:len(value)-1], 10, 64)
	if err != nil || version < 1 {
		return 0, buhari.New(buhari.CodeInvalidIfMatch, "If-Match must contain a quoted positive document version.")
	}
	return version, nil
}

func parseDocumentListFilter(r *http.Request) (mercury.DocumentListFilter, error) {
	query := r.URL.Query()
	filter := mercury.DocumentListFilter{Sort: query.Get("sort")}
	filter.Search = strings.TrimSpace(query.Get("search"))
	if value := query.Get("status"); value != "" {
		status := model.DocumentStatus(value)
		filter.Status = &status
	}
	if value := query.Get("currency"); value != "" {
		currency := model.Currency(value)
		filter.Currency = &currency
	}
	var err error
	if value := query.Get("from"); value != "" {
		date, parseErr := parseDate(value, "from")
		if parseErr != nil {
			return filter, invalidFilter("from", "From date must use YYYY-MM-DD format.")
		}
		filter.IssueFrom = &date
	}
	if value := query.Get("to"); value != "" {
		date, parseErr := parseDate(value, "to")
		if parseErr != nil {
			return filter, invalidFilter("to", "To date must use YYYY-MM-DD format.")
		}
		filter.IssueTo = &date
	}
	if value := query.Get("limit"); value != "" {
		filter.Limit, err = strconv.ParseInt(value, 10, 64)
		if err != nil || filter.Limit < 1 || filter.Limit > mercury.MaximumDocumentLimit {
			return filter, invalidFilter("limit", "Limit must be an integer between 1 and 100.")
		}
	}
	if value := query.Get("page"); value != "" {
		filter.Page, err = strconv.ParseInt(value, 10, 64)
		if err != nil || filter.Page < 1 {
			return filter, invalidFilter("page", "Page must be a positive integer.")
		}
	}
	if value := query.Get("archived"); value != "" {
		archived, parseErr := strconv.ParseBool(value)
		if parseErr != nil {
			return filter, invalidFilter("archived", "Archived must be true or false.")
		}
		filter.Archived = &archived
	}
	for _, unsupported := range []string{"cursor"} {
		if query.Get(unsupported) != "" {
			return filter, invalidFilter(unsupported, "Filter is not supported yet.")
		}
	}
	return filter, nil
}

func decodeOptionalJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return buhari.New(buhari.CodeValidationFailed, "The request body is invalid.")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return buhari.New(buhari.CodeValidationFailed, "The request body must contain exactly one JSON value.")
	}
	return nil
}

func invalidInput(path, message string) error {
	return buhari.Validation(buhari.FieldError{Path: path, Code: buhari.CodeValidationFailed, Message: message})
}

func invalidFilter(path, message string) error {
	return buhari.InvalidFilter(buhari.FieldError{Path: path, Code: buhari.CodeInvalidFilter, Message: message})
}

func respondDocument(ctx context.Context, w http.ResponseWriter, status int, document *model.Document) {
	w.Header().Set("ETag", strconv.Quote(strconv.FormatInt(document.Version, 10)))
	Response(ctx, w, status, newDocumentResponse(*document))
}

func newDocumentResponse(document model.Document) documentResponse {
	return documentResponse{
		ID: document.ID, Reference: document.Reference, Title: document.Title, Customer: document.Customer,
		IssueDate: document.IssueDate.Format("2006-01-02"), Currency: document.Currency, Status: document.Status,
		Version: document.Version, LineItems: newLineResponses(document.LineItems), Totals: newTotalsResponse(document.Totals),
		TaxBreakdown: newTaxBreakdownResponses(document.TaxBreakdown), CalculationPolicyVersion: document.CalculationPolicyVersion,
		FinalizedAt: document.FinalizedAt, ArchivedAt: document.ArchivedAt, CreatedAt: document.CreatedAt, UpdatedAt: document.UpdatedAt,
	}
}

func newPreviewResponse(result calculations.DocumentResult) previewResponse {
	return previewResponse{LineItems: newLineResponses(result.Lines), Totals: newTotalsResponse(result.Totals), TaxBreakdown: newTaxBreakdownResponses(result.TaxBreakdown), CalculationPolicyVersion: result.PolicyVersion}
}

func newLineResponses(lines []calculations.LineResult) []lineResponse {
	result := make([]lineResponse, len(lines))
	for index, line := range lines {
		calculated := line.Calculated
		response := lineResponse{
			ID: line.ID, Description: line.Description, Quantity: line.Quantity, UnitPrice: formatMoney(line.UnitPriceMinor), TaxRate: formatRate(line.TaxRate),
			Calculated: lineCalculatedResponse{
				SubtotalMinor: calculated.SubtotalMinor, Subtotal: formatMoney(calculated.SubtotalMinor),
				DiscountAmountMinor: calculated.DiscountAmountMinor, DiscountAmount: formatMoney(calculated.DiscountAmountMinor),
				DiscountedAmountMinor: calculated.DiscountedAmountMinor, DiscountedAmount: formatMoney(calculated.DiscountedAmountMinor),
				TaxAmountMinor: calculated.TaxAmountMinor, TaxAmount: formatMoney(calculated.TaxAmountMinor),
				LineTotalMinor: calculated.LineTotalMinor, LineTotal: formatMoney(calculated.LineTotalMinor),
			},
		}
		if line.Discount != nil {
			value := formatRate(calculations.Rate(line.Discount.Value))
			if line.Discount.Type == calculations.DiscountFixed {
				value = formatMoney(calculations.Money(line.Discount.Value))
			}
			response.Discount = &discountResponse{Type: line.Discount.Type, Value: value}
		}
		result[index] = response
	}
	return result
}

func newTotalsResponse(totals calculations.Totals) totalsResponse {
	return totalsResponse{
		SubtotalMinor: totals.SubtotalMinor, Subtotal: formatMoney(totals.SubtotalMinor),
		DiscountMinor: totals.DiscountMinor, Discount: formatMoney(totals.DiscountMinor),
		TaxMinor: totals.TaxMinor, Tax: formatMoney(totals.TaxMinor),
		GrandTotalMinor: totals.GrandTotalMinor, GrandTotal: formatMoney(totals.GrandTotalMinor),
	}
}

func newTaxBreakdownResponses(rows []calculations.TaxBreakdown) []taxBreakdownResponse {
	result := make([]taxBreakdownResponse, len(rows))
	for index, row := range rows {
		result[index] = taxBreakdownResponse{Rate: formatRate(row.Rate), TaxableAmount: formatMoney(row.TaxableAmountMinor), TaxAmount: formatMoney(row.TaxAmountMinor)}
	}
	return result
}

func formatMoney(value calculations.Money) string {
	integer := int64(value)
	return strconv.FormatInt(integer/100, 10) + "." + twoDigits(integer%100)
}

func formatRate(value calculations.Rate) string {
	integer := int64(value)
	whole := integer / 10_000
	fraction := strconv.FormatInt(integer%10_000+10_000, 10)[1:]
	fraction = strings.TrimRight(fraction, "0")
	if fraction == "" {
		return strconv.FormatInt(whole, 10)
	}
	return strconv.FormatInt(whole, 10) + "." + fraction
}

func twoDigits(value int64) string {
	if value < 10 {
		return "0" + strconv.FormatInt(value, 10)
	}
	return strconv.FormatInt(value, 10)
}
