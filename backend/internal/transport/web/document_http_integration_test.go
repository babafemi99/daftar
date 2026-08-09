//go:build integration

package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/cfg"
	"github.com/babafemi99/daftar/backend/internal/mercury"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
	"github.com/babafemi99/daftar/backend/internal/service"
	"github.com/testcontainers/testcontainers-go"
	mongotestcontainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

func TestDocumentHTTPWorkflow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := mongotestcontainer.Run(ctx, "mongo:8.0", mongotestcontainer.WithReplicaSet("rs0"))
	if err != nil {
		t.Fatalf("start MongoDB test container: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Errorf("terminate MongoDB test container: %v", err)
		}
	})
	uri, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatal(err)
	}
	client, err := mongo.Connect(options.Client().ApplyURI(uri).SetDirect(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := client.Disconnect(shutdownCtx); err != nil {
			t.Errorf("disconnect MongoDB: %v", err)
		}
	})
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		t.Fatal(err)
	}
	database := client.Database(strings.ReplaceAll("daftar_http_"+lid.NewRequest(), "-", "_"))
	core, err := mercury.NewCore(database)
	if err != nil {
		t.Fatal(err)
	}
	if err := core.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	users, err := service.NewUserService(core.User())
	if err != nil {
		t.Fatal(err)
	}
	documents, err := service.NewAuditedDocumentService(core.Document(), core.DocumentReference(), core.Audit(), core)
	if err != nil {
		t.Fatal(err)
	}
	api := NewAPI(cfg.HTTP{CORSAllowedOrigins: []string{"https://app.example.com"}})
	if err := api.ConfigureAuth(users, cfg.JWT{
		Secret: "01234567890123456789012345678901", Issuer: "daftar-http-integration",
		Audience: "daftar-http-integration", AccessTTL: time.Hour,
	}, cfg.Cookie{Name: "daftar_session", Path: "/", SameSite: "lax"}); err != nil {
		t.Fatal(err)
	}
	if err := api.ConfigureDocuments(documents); err != nil {
		t.Fatal(err)
	}
	audits, err := service.NewAuditService(core.Audit(), core.Document())
	if err != nil {
		t.Fatal(err)
	}
	if err := api.ConfigureAudits(audits); err != nil {
		t.Fatal(err)
	}
	reports, err := service.NewReportService(core.Report())
	if err != nil {
		t.Fatal(err)
	}
	if err := api.ConfigureReports(reports); err != nil {
		t.Fatal(err)
	}

	ownerCookie := registerHTTPUser(t, api, "owner@example.com")
	otherCookie := registerHTTPUser(t, api, "other@example.com")
	createBody := `{"title":"Mixed rates","customer":"CrossVal","issueDate":"2026-08-08","currency":"USD","lineItems":[{"description":"Percentage","quantity":2,"unitPrice":"100.00","discount":{"type":"percentage","value":"10"},"taxRate":"5"},{"description":"Fixed","quantity":1,"unitPrice":"75.00","discount":{"type":"fixed","value":"5.00"},"taxRate":"20"}]}`
	created := documentHTTPRequest(t, api, ownerCookie, http.MethodPost, "/api/v1/documents", createBody, "")
	createdDocument := decodeHTTPDocument(t, created, http.StatusCreated)
	if len(createdDocument.LineItems) != 2 || len(createdDocument.TaxBreakdown) != 2 || createdDocument.Totals.GrandTotalMinor == 0 {
		t.Fatalf("created document did not preserve mixed calculations: %+v", createdDocument)
	}

	retrieved := documentHTTPRequest(t, api, ownerCookie, http.MethodGet, "/api/v1/documents/"+createdDocument.ID, "", "")
	retrievedDocument := decodeHTTPDocument(t, retrieved, http.StatusOK)
	if retrievedDocument.ID != createdDocument.ID || retrievedDocument.Totals != createdDocument.Totals {
		t.Fatalf("retrieved document differs from create response")
	}

	lineBody := `{"description":"Additional","quantity":1,"unitPrice":"50.00","discount":{"type":"percentage","value":"2.5"},"taxRate":"10"}`
	added := documentHTTPRequest(t, api, ownerCookie, http.MethodPost, "/api/v1/documents/"+createdDocument.ID+"/line-items", lineBody, created.Header().Get("ETag"))
	addedDocument := decodeHTTPDocument(t, added, http.StatusCreated)
	if len(addedDocument.LineItems) != 3 || addedDocument.Version != 2 {
		t.Fatalf("line mutation result=%+v", addedDocument)
	}

	finalized := documentHTTPRequest(t, api, ownerCookie, http.MethodPost, "/api/v1/documents/"+createdDocument.ID+"/finalize", "", added.Header().Get("ETag"))
	finalizedDocument := decodeHTTPDocument(t, finalized, http.StatusOK)
	if finalizedDocument.Status != "finalized" || finalizedDocument.Version != 3 {
		t.Fatalf("finalized document=%+v", finalizedDocument)
	}

	rejected := documentHTTPRequest(t, api, ownerCookie, http.MethodPost, "/api/v1/documents/"+createdDocument.ID+"/line-items", lineBody, finalized.Header().Get("ETag"))
	assertHTTPApplicationError(t, rejected, http.StatusConflict, buhari.CodeDocumentFinalized)

	crossOwner := documentHTTPRequest(t, api, otherCookie, http.MethodGet, "/api/v1/documents/"+createdDocument.ID, "", "")
	assertHTTPApplicationError(t, crossOwner, http.StatusNotFound, buhari.CodeNotFound)

	auditResponse := documentHTTPRequest(t, api, ownerCookie, http.MethodGet, "/api/v1/documents/"+createdDocument.ID+"/audit-events", "", "")
	if auditResponse.Code != http.StatusOK {
		t.Fatalf("audit status=%d body=%s", auditResponse.Code, auditResponse.Body.String())
	}
	var auditEnvelope struct {
		Data []auditEventResponse `json:"data"`
	}
	if err := json.NewDecoder(auditResponse.Body).Decode(&auditEnvelope); err != nil {
		t.Fatal(err)
	}
	if len(auditEnvelope.Data) != 3 || auditEnvelope.Data[0].Action != "document.finalized" || auditEnvelope.Data[2].Action != "document.created" {
		t.Fatalf("audit events=%+v", auditEnvelope.Data)
	}
	crossOwnerAudit := documentHTTPRequest(t, api, otherCookie, http.MethodGet, "/api/v1/documents/"+createdDocument.ID+"/audit-events", "", "")
	assertHTTPApplicationError(t, crossOwnerAudit, http.StatusNotFound, buhari.CodeNotFound)

	reportResponse := documentHTTPRequest(t, api, ownerCookie, http.MethodGet, "/api/v1/reports/summary?from=2026-08-08&to=2026-08-08", "", "")
	if reportResponse.Code != http.StatusOK {
		t.Fatalf("report status=%d body=%s", reportResponse.Code, reportResponse.Body.String())
	}
	var reportEnvelope struct {
		Data summaryReportResponse `json:"data"`
	}
	if err := json.NewDecoder(reportResponse.Body).Decode(&reportEnvelope); err != nil {
		t.Fatal(err)
	}
	if reportEnvelope.Data.DocumentCount != 1 || len(reportEnvelope.Data.Currencies) != 1 || reportEnvelope.Data.Currencies[0].Currency != "USD" {
		t.Fatalf("report=%+v", reportEnvelope.Data)
	}

	otherReportResponse := documentHTTPRequest(t, api, otherCookie, http.MethodGet, "/api/v1/reports/summary?from=2026-08-08&to=2026-08-08", "", "")
	if otherReportResponse.Code != http.StatusOK {
		t.Fatalf("other report status=%d body=%s", otherReportResponse.Code, otherReportResponse.Body.String())
	}
	var otherReportEnvelope struct {
		Data summaryReportResponse `json:"data"`
	}
	if err := json.NewDecoder(otherReportResponse.Body).Decode(&otherReportEnvelope); err != nil {
		t.Fatal(err)
	}
	if otherReportEnvelope.Data.DocumentCount != 0 || len(otherReportEnvelope.Data.Currencies) != 0 {
		t.Fatalf("other owner report=%+v", otherReportEnvelope.Data)
	}
}

func registerHTTPUser(t *testing.T, api *API, email string) *http.Cookie {
	t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":"strong-password","first_name":"Ada","last_name":"Lovelace"}`, email)
	response := documentHTTPRequest(t, api, nil, http.MethodPost, "/api/v1/auth/register", body, "")
	if response.Code != http.StatusCreated {
		t.Fatalf("register %s: status=%d body=%s", email, response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("register %s cookies=%#v", email, cookies)
	}
	return cookies[0]
}

func documentHTTPRequest(t *testing.T, api *API, cookie *http.Cookie, method, target, body, ifMatch string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Origin", "https://app.example.com")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	return response
}

func decodeHTTPDocument(t *testing.T, response *httptest.ResponseRecorder, status int) documentResponse {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status=%d want=%d body=%s", response.Code, status, response.Body.String())
	}
	var envelope struct {
		Data documentResponse `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func assertHTTPApplicationError(t *testing.T, response *httptest.ResponseRecorder, status int, code buhari.Code) {
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
