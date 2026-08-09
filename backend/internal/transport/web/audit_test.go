package web

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
)

type fakeAudits struct {
	events     []model.AuditEvent
	documentID string
	limit      int64
	err        error
}

func (audits *fakeAudits) ListDocument(_ context.Context, documentID string, limit int64) ([]model.AuditEvent, error) {
	audits.documentID, audits.limit = documentID, limit
	return audits.events, audits.err
}

func TestListDocumentAuditEvents(t *testing.T) {
	document := webTestDocument(t)
	event, err := model.NewAuditEvent(document.OwnerID, document.OwnerID, lid.NewRequest(), document, model.AuditDocumentCreated, model.AuditMetadata{ChangedFields: []string{"title"}}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	audits := &fakeAudits{events: []model.AuditEvent{event}}
	api := configuredDocumentAPI(t, &fakeDocuments{document: document})
	if err := api.ConfigureAudits(audits); err != nil {
		t.Fatal(err)
	}
	response := serveDocumentRequest(t, api, http.MethodGet, "/api/v1/documents/"+document.ID+"/audit-events?limit=10", "", "")
	if response.Code != http.StatusOK || audits.documentID != document.ID || audits.limit != 10 {
		t.Fatalf("status=%d document=%s limit=%d body=%s", response.Code, audits.documentID, audits.limit, response.Body.String())
	}
	var envelope struct {
		Data []auditEventResponse `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data) != 1 || envelope.Data[0].Action != model.AuditDocumentCreated {
		t.Fatalf("response=%+v", envelope)
	}
}

func TestListDocumentAuditEventsRejectsInvalidLimitAndPreservesErrors(t *testing.T) {
	audits := &fakeAudits{}
	api := configuredDocumentAPI(t, &fakeDocuments{document: webTestDocument(t)})
	if err := api.ConfigureAudits(audits); err != nil {
		t.Fatal(err)
	}
	invalid := serveDocumentRequest(t, api, http.MethodGet, "/api/v1/documents/"+lid.NewDocument()+"/audit-events?limit=101", "", "")
	assertErrorResponse(t, invalid, http.StatusUnprocessableEntity, buhari.CodeInvalidFilter)
	if audits.documentID != "" {
		t.Fatal("repository was called for invalid input")
	}

	audits.err = buhari.New(buhari.CodeNotFound, "Document not found.")
	notFound := serveDocumentRequest(t, api, http.MethodGet, "/api/v1/documents/"+lid.NewDocument()+"/audit-events", "", "")
	assertErrorResponse(t, notFound, http.StatusNotFound, buhari.CodeNotFound)
}
