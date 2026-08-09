package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/calculations"
	"github.com/babafemi99/daftar/backend/internal/mercury"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
	"github.com/babafemi99/daftar/backend/internal/pkg/tokoz"
	"github.com/babafemi99/daftar/backend/internal/service"
)

type fakeDocuments struct {
	document   model.Document
	result     calculations.DocumentResult
	calls      []string
	errByCall  map[string]error
	version    int64
	documentID string
	lineID     string
	input      model.DocumentInput
	line       calculations.LineInput
	filter     mercury.DocumentListFilter
	order      []string
	overrides  service.DuplicateDocumentInput
}

func (f *fakeDocuments) called(name string) error {
	f.calls = append(f.calls, name)
	return f.errByCall[name]
}
func (f *fakeDocuments) Create(_ context.Context, input model.DocumentInput) (*model.Document, error) {
	f.input = input
	if err := f.called("create"); err != nil {
		return nil, err
	}
	return &f.document, nil
}
func (f *fakeDocuments) Preview(_ context.Context, input service.CalculationPreviewInput) (*calculations.DocumentResult, error) {
	f.line = input.LineItems[0]
	if err := f.called("preview"); err != nil {
		return nil, err
	}
	return &f.result, nil
}
func (f *fakeDocuments) GetByID(_ context.Context, id string) (*model.Document, error) {
	f.documentID = id
	if err := f.called("get"); err != nil {
		return nil, err
	}
	return &f.document, nil
}
func (f *fakeDocuments) List(_ context.Context, filter mercury.DocumentListFilter) (mercury.DocumentPage, error) {
	f.filter = filter
	if err := f.called("list"); err != nil {
		return mercury.DocumentPage{}, err
	}
	return mercury.DocumentPage{Documents: []model.Document{f.document}, Page: 1, PageSize: 20, TotalItems: 1, TotalPages: 1}, nil
}
func (f *fakeDocuments) ReplaceDraft(_ context.Context, id string, version int64, input model.DocumentInput) (*model.Document, error) {
	f.documentID, f.version, f.input = id, version, input
	if err := f.called("replace"); err != nil {
		return nil, err
	}
	return &f.document, nil
}
func (f *fakeDocuments) AddLine(_ context.Context, id string, version int64, line calculations.LineInput) (*model.Document, error) {
	f.documentID, f.version, f.line = id, version, line
	if err := f.called("add-line"); err != nil {
		return nil, err
	}
	return &f.document, nil
}
func (f *fakeDocuments) UpdateLine(_ context.Context, id, lineID string, version int64, line calculations.LineInput) (*model.Document, error) {
	f.documentID, f.lineID, f.version, f.line = id, lineID, version, line
	if err := f.called("update-line"); err != nil {
		return nil, err
	}
	return &f.document, nil
}
func (f *fakeDocuments) DeleteLine(_ context.Context, id, lineID string, version int64) (*model.Document, error) {
	f.documentID, f.lineID, f.version = id, lineID, version
	if err := f.called("delete-line"); err != nil {
		return nil, err
	}
	return &f.document, nil
}
func (f *fakeDocuments) ReorderLines(_ context.Context, id string, version int64, order []string) (*model.Document, error) {
	f.documentID, f.version, f.order = id, version, order
	if err := f.called("reorder"); err != nil {
		return nil, err
	}
	return &f.document, nil
}
func (f *fakeDocuments) Archive(_ context.Context, id string, version int64) (*model.Document, error) {
	f.documentID, f.version = id, version
	if err := f.called("archive"); err != nil {
		return nil, err
	}
	return &f.document, nil
}
func (f *fakeDocuments) Restore(_ context.Context, id string, version int64) (*model.Document, error) {
	f.documentID, f.version = id, version
	if err := f.called("restore"); err != nil {
		return nil, err
	}
	return &f.document, nil
}
func (f *fakeDocuments) Finalize(_ context.Context, id string, version int64) (*model.Document, error) {
	f.documentID, f.version = id, version
	if err := f.called("finalize"); err != nil {
		return nil, err
	}
	return &f.document, nil
}
func (f *fakeDocuments) Duplicate(_ context.Context, id string, overrides service.DuplicateDocumentInput) (*model.Document, error) {
	f.documentID, f.overrides = id, overrides
	if err := f.called("duplicate"); err != nil {
		return nil, err
	}
	return &f.document, nil
}

func TestDocumentRoutesRequireAuthentication(t *testing.T) {
	api := configuredDocumentAPI(t, &fakeDocuments{document: webTestDocument(t)})
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil))
	assertErrorResponse(t, response, http.StatusUnauthorized, buhari.CodeUnauthorized)
}

func TestCreateDocumentStrictlyDecodesAndConvertsInput(t *testing.T) {
	fake := &fakeDocuments{document: webTestDocument(t)}
	api := configuredDocumentAPI(t, fake)
	body := `{"title":"August","customer":"Acme","issueDate":"2026-08-08","currency":"USD","lineItems":[{"description":"Consulting","quantity":2,"unitPrice":"100.25","discount":{"type":"percentage","value":"10.5"},"taxRate":"5.25"}]}`
	response := serveDocumentRequest(t, api, http.MethodPost, "/api/v1/documents", body, "")
	if response.Code != http.StatusCreated || response.Header().Get("ETag") != `"1"` {
		t.Fatalf("status=%d etag=%q body=%s", response.Code, response.Header().Get("ETag"), response.Body.String())
	}
	if len(fake.calls) != 1 || fake.calls[0] != "create" || fake.input.LineItems[0].UnitPriceMinor != 10_025 || fake.input.LineItems[0].TaxRate != 52_500 || fake.input.LineItems[0].Discount.Value != 105_000 {
		t.Fatalf("converted input = %+v calls=%v", fake.input, fake.calls)
	}
	var envelope struct {
		Data documentResponse `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.LineItems[0].UnitPrice != "100.00" || envelope.Data.LineItems[0].TaxRate != "5" {
		t.Fatalf("response conversion = %+v", envelope.Data.LineItems[0])
	}
}

func TestDocumentJSONRejectsUnknownTrailingCalculatedAndClientID(t *testing.T) {
	tests := []string{
		`{"title":"A","customer":"B","issueDate":"2026-08-08","currency":"USD","lineItems":[],"unknown":true}`,
		`{"title":"A","customer":"B","issueDate":"2026-08-08","currency":"USD","lineItems":[]} {"extra":true}`,
		`{"title":"A","customer":"B","issueDate":"2026-08-08","currency":"USD","lineItems":[{"description":"x","quantity":1,"unitPrice":"1.00","taxRate":"0","calculated":{"lineTotalMinor":100}}]}`,
		`{"title":"A","customer":"B","issueDate":"2026-08-08","currency":"USD","lineItems":[{"id":"line-client","description":"x","quantity":1,"unitPrice":"1.00","taxRate":"0"}]}`,
	}
	for _, body := range tests {
		fake := &fakeDocuments{document: webTestDocument(t)}
		response := serveDocumentRequest(t, configuredDocumentAPI(t, fake), http.MethodPost, "/api/v1/documents", body, "")
		assertErrorResponse(t, response, http.StatusUnprocessableEntity, buhari.CodeValidationFailed)
		if len(fake.calls) != 0 {
			t.Fatalf("service calls=%v", fake.calls)
		}
	}
}

func TestDocumentLineConversionErrorsUseIndexedPaths(t *testing.T) {
	tests := []struct {
		name string
		body string
		path string
	}{
		{
			name: "unit price",
			body: `{"title":"A","customer":"B","issueDate":"2026-08-08","currency":"USD","lineItems":[{"description":"x","quantity":1,"unitPrice":"invalid","taxRate":"0"}]}`,
			path: "lineItems[0].unitPrice",
		},
		{
			name: "tax rate",
			body: `{"title":"A","customer":"B","issueDate":"2026-08-08","currency":"USD","lineItems":[{"description":"x","quantity":1,"unitPrice":"1.00","taxRate":"0"},{"description":"y","quantity":1,"unitPrice":"1.00","taxRate":"invalid"}]}`,
			path: "lineItems[1].taxRate",
		},
		{
			name: "discount value",
			body: `{"title":"A","customer":"B","issueDate":"2026-08-08","currency":"USD","lineItems":[{"description":"x","quantity":1,"unitPrice":"1.00","taxRate":"0"},{"description":"y","quantity":1,"unitPrice":"1.00","taxRate":"0"},{"description":"z","quantity":1,"unitPrice":"1.00","discount":{"type":"fixed","value":"invalid"},"taxRate":"0"}]}`,
			path: "lineItems[2].discount.value",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := serveDocumentRequest(t, configuredDocumentAPI(t, &fakeDocuments{document: webTestDocument(t)}), http.MethodPost, "/api/v1/documents", test.body, "")
			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			var envelope ErrorEnvelope
			if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
				t.Fatal(err)
			}
			details, ok := envelope.Error.Details.(map[string]any)
			if !ok {
				t.Fatalf("details=%#v", envelope.Error.Details)
			}
			fields, ok := details["fields"].([]any)
			if !ok || len(fields) != 1 {
				t.Fatalf("fields=%#v", details["fields"])
			}
			field, ok := fields[0].(map[string]any)
			if !ok || field["path"] != test.path {
				t.Fatalf("field=%#v want path=%q", fields[0], test.path)
			}
		})
	}
}

func TestListDocumentsUsesCollectionEnvelope(t *testing.T) {
	fake := &fakeDocuments{document: webTestDocument(t)}
	response := serveDocumentRequest(t, configuredDocumentAPI(t, fake), http.MethodGet, "/api/v1/documents?page=1&limit=20&search=Acme", "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var envelope struct {
		Data []documentResponse `json:"data"`
		Page struct {
			NextCursor *string `json:"nextCursor"`
			HasMore    bool    `json:"hasMore"`
			Number     int64   `json:"number"`
			Size       int64   `json:"size"`
			TotalItems int64   `json:"totalItems"`
			TotalPages int64   `json:"totalPages"`
		} `json:"page"`
		RequestID string `json:"requestId"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data) != 1 || envelope.Page.NextCursor != nil || envelope.Page.HasMore || envelope.Page.Number != 1 || envelope.Page.Size != 20 || envelope.Page.TotalItems != 1 || envelope.Page.TotalPages != 1 || envelope.RequestID == "" {
		t.Fatalf("envelope=%+v", envelope)
	}
	if fake.filter.Page != 1 || fake.filter.Limit != 20 || fake.filter.Search != "Acme" {
		t.Fatalf("filter=%+v", fake.filter)
	}
}

func TestDocumentMutationsRequireAndPropagateIfMatch(t *testing.T) {
	body := `{"description":"Line","quantity":1,"unitPrice":"10.00","taxRate":"0"}`
	for _, header := range []string{"", "4", `"0"`, `"abc"`} {
		fake := &fakeDocuments{document: webTestDocument(t)}
		response := serveDocumentRequest(t, configuredDocumentAPI(t, fake), http.MethodPost, "/api/v1/documents/doc-test/line-items", body, header)
		assertErrorResponse(t, response, http.StatusBadRequest, buhari.CodeInvalidIfMatch)
		if len(fake.calls) != 0 {
			t.Fatalf("header=%q calls=%v", header, fake.calls)
		}
	}
	fake := &fakeDocuments{document: webTestDocument(t)}
	response := serveDocumentRequest(t, configuredDocumentAPI(t, fake), http.MethodPost, "/api/v1/documents/doc-test/line-items", body, `"4"`)
	if response.Code != http.StatusCreated || fake.version != 4 || fake.documentID != "doc-test" {
		t.Fatalf("status=%d version=%d document=%q body=%s", response.Code, fake.version, fake.documentID, response.Body.String())
	}
}

func TestDocumentOwnershipAndLifecycleErrorsUseStandardEnvelope(t *testing.T) {
	t.Run("ownership", func(t *testing.T) {
		fake := &fakeDocuments{document: webTestDocument(t), errByCall: map[string]error{"get": buhari.New(buhari.CodeNotFound, "Document not found.")}}
		response := serveDocumentRequest(t, configuredDocumentAPI(t, fake), http.MethodGet, "/api/v1/documents/doc-other", "", "")
		assertErrorResponse(t, response, http.StatusNotFound, buhari.CodeNotFound)
	})
	t.Run("lifecycle", func(t *testing.T) {
		fake := &fakeDocuments{document: webTestDocument(t), errByCall: map[string]error{"finalize": buhari.New(buhari.CodeDocumentAlreadyFinalized, "Document is already finalized.")}}
		response := serveDocumentRequest(t, configuredDocumentAPI(t, fake), http.MethodPost, "/api/v1/documents/doc-test/finalize", "", `"1"`)
		assertErrorResponse(t, response, http.StatusConflict, buhari.CodeDocumentAlreadyFinalized)
	})
}

func TestDocumentSuccessfulWorkflowRoutes(t *testing.T) {
	documentBody := `{"title":"A","customer":"B","issueDate":"2026-08-08","currency":"USD","lineItems":[]}`
	lineBody := `{"description":"Line","quantity":1,"unitPrice":"10.00","taxRate":"0"}`
	tests := []struct {
		name, method, path, body, ifMatch, call string
		status                                  int
	}{
		{"list", http.MethodGet, "/api/v1/documents?status=draft&sort=issue_date_desc", "", "", "list", 200},
		{"get", http.MethodGet, "/api/v1/documents/doc-test", "", "", "get", 200},
		{"replace", http.MethodPatch, "/api/v1/documents/doc-test", documentBody, `"1"`, "replace", 200},
		{"preview", http.MethodPost, "/api/v1/documents/preview-calculation", `{"currency":"USD","lineItems":[` + lineBody + `]}`, "", "preview", 200},
		{"add line", http.MethodPost, "/api/v1/documents/doc-test/line-items", lineBody, `"1"`, "add-line", 201},
		{"update line", http.MethodPatch, "/api/v1/documents/doc-test/line-items/line-test", lineBody, `"1"`, "update-line", 200},
		{"delete line", http.MethodDelete, "/api/v1/documents/doc-test/line-items/line-test", "", `"1"`, "delete-line", 200},
		{"reorder", http.MethodPost, "/api/v1/documents/doc-test/line-items/reorder", `{"lineItemIds":["line-test"]}`, `"1"`, "reorder", 200},
		{"archive", http.MethodDelete, "/api/v1/documents/doc-test", "", `"1"`, "archive", 200},
		{"restore", http.MethodPost, "/api/v1/documents/doc-test/restore", "", `"1"`, "restore", 200},
		{"finalize", http.MethodPost, "/api/v1/documents/doc-test/finalize", "", `"1"`, "finalize", 200},
		{"duplicate", http.MethodPost, "/api/v1/documents/doc-test/duplicate", `{"title":"Copy","issueDate":"2026-09-01"}`, "", "duplicate", 201},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeDocuments{document: webTestDocument(t), result: webTestCalculation(t)}
			response := serveDocumentRequest(t, configuredDocumentAPI(t, fake), test.method, test.path, test.body, test.ifMatch)
			if response.Code != test.status || len(fake.calls) != 1 || fake.calls[0] != test.call {
				t.Fatalf("status=%d calls=%v body=%s", response.Code, fake.calls, response.Body.String())
			}
		})
	}
}

func configuredDocumentAPI(t *testing.T, documents Documents) *API {
	t.Helper()
	user := model.User{ID: lid.NewUser(), Email: "user@example.com", FirstName: "Ada", LastName: "Lovelace"}
	api := configuredAuthAPI(t, &fakeAuthUsers{user: user})
	if err := api.ConfigureDocuments(documents); err != nil {
		t.Fatal(err)
	}
	return api
}

func serveDocumentRequest(t *testing.T, api *API, method, target, body, ifMatch string) *httptest.ResponseRecorder {
	t.Helper()
	token, err := tokoz.GenerateToken(lid.NewUser(), "user", "user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Origin", "https://app.example.com")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	return response
}

func webTestDocument(t *testing.T) model.Document {
	t.Helper()
	document, err := model.NewDocument(lid.NewUser(), "DOC-2026-000001", model.DocumentInput{
		Title: "Test", Customer: "Acme", IssueDate: time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC), Currency: model.CurrencyUSD,
		LineItems: []calculations.LineInput{{Description: "Line", Quantity: 1, UnitPriceMinor: 10_000, TaxRate: 50_000}},
	}, time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func webTestCalculation(t *testing.T) calculations.DocumentResult {
	t.Helper()
	result, err := calculations.CalculateDocument([]calculations.LineInput{{Description: "Line", Quantity: 1, UnitPriceMinor: 1_000}})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func assertErrorResponse(t *testing.T, response *httptest.ResponseRecorder, status int, code buhari.Code) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status=%d want=%d body=%s", response.Code, status, response.Body.String())
	}
	var envelope ErrorEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error.Code != code {
		t.Fatalf("code=%s want=%s body=%s", envelope.Error.Code, code, response.Body.String())
	}
}
