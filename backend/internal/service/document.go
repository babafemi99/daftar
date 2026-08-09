package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/calculations"
	"github.com/babafemi99/daftar/backend/internal/mercury"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
	"github.com/babafemi99/daftar/backend/internal/pkg/requestctx"
)

var (
	ErrDocumentRepositoryRequired = errors.New("service: document repository is required")
	ErrDocumentReferencesRequired = errors.New("service: document reference repository is required")
	ErrAuditRepositoryRequired    = errors.New("service: audit repository is required")
	ErrTransactionRunnerRequired  = errors.New("service: transaction runner is required")
)

type transactionRunner interface {
	RunInTx(context.Context, mercury.Transaction) error
}

type DocumentService struct {
	documents  mercury.Documents
	references mercury.DocumentReferences
	audits     mercury.Audits
	tx         transactionRunner
}

func NewAuditedDocumentService(documents mercury.Documents, references mercury.DocumentReferences, audits mercury.Audits, tx transactionRunner) (*DocumentService, error) {
	service, err := NewDocumentService(documents, references)
	if err != nil {
		return nil, err
	}
	if audits == nil {
		return nil, ErrAuditRepositoryRequired
	}
	if tx == nil {
		return nil, ErrTransactionRunnerRequired
	}
	service.audits, service.tx = audits, tx
	return service, nil
}

type DuplicateDocumentInput struct {
	Title     *string
	IssueDate *time.Time
}

type CalculationPreviewInput struct {
	Currency  model.Currency
	LineItems []calculations.LineInput
}

func NewDocumentService(documents mercury.Documents, references mercury.DocumentReferences) (*DocumentService, error) {
	if documents == nil {
		return nil, ErrDocumentRepositoryRequired
	}
	if references == nil {
		return nil, ErrDocumentReferencesRequired
	}
	return &DocumentService{documents: documents, references: references}, nil
}

func (service *DocumentService) Create(ctx context.Context, input model.DocumentInput) (*model.Document, error) {
	ownerID, err := authenticatedOwner(ctx)
	if err != nil {
		return nil, err
	}
	now := pkg.UTCNow()
	reference, err := service.references.Allocate(ctx, ownerID, input.IssueDate.Year(), now)
	if err != nil {
		return nil, err
	}
	document, err := model.NewDocument(ownerID, reference, input, now)
	if err != nil {
		return nil, err
	}
	if err := service.persistWithAudit(ctx, document, model.AuditDocumentCreated, model.AuditMetadata{ChangedFields: []string{"title", "customer", "issueDate", "currency", "lineItems", "totals", "taxBreakdown"}}, func(txCtx context.Context) error { return service.documents.Create(txCtx, document) }); err != nil {
		return nil, err
	}
	return &document, nil
}

func (service *DocumentService) Preview(ctx context.Context, input CalculationPreviewInput) (*calculations.DocumentResult, error) {
	if _, err := authenticatedOwner(ctx); err != nil {
		return nil, err
	}
	if !input.Currency.Valid() {
		return nil, buhari.Validation(buhari.FieldError{Path: "currency", Code: buhari.CodeInvalidCurrency, Message: "Currency is not supported."})
	}
	result, err := calculations.CalculateDocument(input.LineItems)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (service *DocumentService) GetByID(ctx context.Context, documentID string) (*model.Document, error) {
	ownerID, err := authenticatedOwner(ctx)
	if err != nil {
		return nil, err
	}
	document, err := service.documents.FindByID(ctx, ownerID, documentID)
	if err != nil {
		return nil, err
	}
	return &document, nil
}

func (service *DocumentService) List(ctx context.Context, filter mercury.DocumentListFilter) (mercury.DocumentPage, error) {
	ownerID, err := authenticatedOwner(ctx)
	if err != nil {
		return mercury.DocumentPage{}, err
	}
	if err := validateDocumentListFilter(filter); err != nil {
		return mercury.DocumentPage{}, err
	}
	return service.documents.List(ctx, ownerID, filter)
}

func (service *DocumentService) ReplaceDraft(ctx context.Context, documentID string, expectedVersion int64, input model.DocumentInput) (*model.Document, error) {
	document, err := service.loadForMutation(ctx, documentID, expectedVersion)
	if err != nil {
		return nil, err
	}
	if err := document.ReplaceDraft(input, pkg.UTCNow()); err != nil {
		return nil, err
	}
	if err := service.persistWithAudit(ctx, *document, model.AuditDocumentUpdated, documentChangedMetadata(), func(txCtx context.Context) error {
		return service.documents.ReplaceDraft(txCtx, *document, expectedVersion)
	}); err != nil {
		return nil, err
	}
	return document, nil
}

func (service *DocumentService) AddLine(ctx context.Context, documentID string, expectedVersion int64, input calculations.LineInput) (*model.Document, error) {
	document, err := service.loadForMutation(ctx, documentID, expectedVersion)
	if err != nil {
		return nil, err
	}
	if err := requireActiveDraft(*document); err != nil {
		return nil, err
	}
	input = cloneLineInput(input)
	input.ID = ""
	documentInput := editableDocumentInput(*document)
	documentInput.LineItems = append(documentInput.LineItems, input)
	return service.replaceLoadedDraft(ctx, document, expectedVersion, documentInput)
}

func (service *DocumentService) UpdateLine(ctx context.Context, documentID, lineID string, expectedVersion int64, input calculations.LineInput) (*model.Document, error) {
	document, err := service.loadForMutation(ctx, documentID, expectedVersion)
	if err != nil {
		return nil, err
	}
	if err := requireActiveDraft(*document); err != nil {
		return nil, err
	}
	documentInput := editableDocumentInput(*document)
	lineIndex := findLine(documentInput.LineItems, lineID)
	if lineIndex < 0 {
		return nil, lineItemNotFound()
	}
	input = cloneLineInput(input)
	input.ID = lineID
	documentInput.LineItems[lineIndex] = input
	return service.replaceLoadedDraft(ctx, document, expectedVersion, documentInput)
}

func (service *DocumentService) DeleteLine(ctx context.Context, documentID, lineID string, expectedVersion int64) (*model.Document, error) {
	document, err := service.loadForMutation(ctx, documentID, expectedVersion)
	if err != nil {
		return nil, err
	}
	if err := requireActiveDraft(*document); err != nil {
		return nil, err
	}
	documentInput := editableDocumentInput(*document)
	lineIndex := findLine(documentInput.LineItems, lineID)
	if lineIndex < 0 {
		return nil, lineItemNotFound()
	}
	documentInput.LineItems = append(documentInput.LineItems[:lineIndex], documentInput.LineItems[lineIndex+1:]...)
	return service.replaceLoadedDraft(ctx, document, expectedVersion, documentInput)
}

func (service *DocumentService) ReorderLines(ctx context.Context, documentID string, expectedVersion int64, orderedLineIDs []string) (*model.Document, error) {
	document, err := service.loadForMutation(ctx, documentID, expectedVersion)
	if err != nil {
		return nil, err
	}
	if err := requireActiveDraft(*document); err != nil {
		return nil, err
	}
	if len(orderedLineIDs) != len(document.LineItems) {
		return nil, invalidLineOrder()
	}
	byID := make(map[string]calculations.LineInput, len(document.LineItems))
	for _, line := range document.LineItems {
		byID[line.ID] = cloneLineInput(line.LineInput)
	}
	ordered := make([]calculations.LineInput, len(orderedLineIDs))
	seen := make(map[string]struct{}, len(orderedLineIDs))
	for index, lineID := range orderedLineIDs {
		line, exists := byID[lineID]
		if !exists {
			return nil, invalidLineOrder()
		}
		if _, duplicate := seen[lineID]; duplicate {
			return nil, invalidLineOrder()
		}
		seen[lineID] = struct{}{}
		ordered[index] = line
	}
	documentInput := editableDocumentInput(*document)
	documentInput.LineItems = ordered
	return service.replaceLoadedDraft(ctx, document, expectedVersion, documentInput)
}

func (service *DocumentService) Archive(ctx context.Context, documentID string, expectedVersion int64) (*model.Document, error) {
	document, err := service.loadForMutation(ctx, documentID, expectedVersion)
	if err != nil {
		return nil, err
	}
	if err := document.Archive(pkg.UTCNow()); err != nil {
		return nil, err
	}
	if err := service.persistWithAudit(ctx, *document, model.AuditDocumentArchived, model.AuditMetadata{ChangedFields: []string{"archivedAt"}}, func(txCtx context.Context) error { return service.documents.Archive(txCtx, *document, expectedVersion) }); err != nil {
		return nil, err
	}
	return document, nil
}

func (service *DocumentService) Restore(ctx context.Context, documentID string, expectedVersion int64) (*model.Document, error) {
	document, err := service.loadForMutation(ctx, documentID, expectedVersion)
	if err != nil {
		return nil, err
	}
	if err := document.Restore(pkg.UTCNow()); err != nil {
		return nil, err
	}
	if err := service.persistWithAudit(ctx, *document, model.AuditDocumentRestored, model.AuditMetadata{ChangedFields: []string{"archivedAt"}}, func(txCtx context.Context) error { return service.documents.Restore(txCtx, *document, expectedVersion) }); err != nil {
		return nil, err
	}
	return document, nil
}

func (service *DocumentService) Finalize(ctx context.Context, documentID string, expectedVersion int64) (*model.Document, error) {
	document, err := service.GetByID(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if document.Status == model.DocumentFinalized {
		return nil, buhari.New(buhari.CodeDocumentAlreadyFinalized, "Document is already finalized.")
	}
	if expectedVersion < 1 || document.Version != expectedVersion {
		return nil, buhari.New(buhari.CodeDocumentVersionConflict, "Document was changed by another request.")
	}
	if err := document.Finalize(pkg.UTCNow()); err != nil {
		return nil, err
	}
	if err := service.persistWithAudit(ctx, *document, model.AuditDocumentFinalized, model.AuditMetadata{ChangedFields: []string{"status", "totals", "taxBreakdown"}, CalculationPolicyVersion: document.CalculationPolicyVersion}, func(txCtx context.Context) error {
		return service.documents.Finalize(txCtx, *document, expectedVersion)
	}); err != nil {
		return nil, err
	}
	return document, nil
}

func (service *DocumentService) Duplicate(ctx context.Context, sourceDocumentID string, overrides DuplicateDocumentInput) (*model.Document, error) {
	ownerID, err := authenticatedOwner(ctx)
	if err != nil {
		return nil, err
	}
	source, err := service.documents.FindByID(ctx, ownerID, sourceDocumentID)
	if err != nil {
		return nil, err
	}
	if source.Status != model.DocumentFinalized {
		return nil, buhari.New(buhari.CodeSourceDocumentNotFinalized, "Only finalized documents can be duplicated.")
	}

	input := model.DocumentInput{
		Title: source.Title, Customer: source.Customer, IssueDate: source.IssueDate,
		Currency: source.Currency, LineItems: duplicateLineInputs(source.LineItems),
	}
	if overrides.Title != nil {
		input.Title = *overrides.Title
	}
	if overrides.IssueDate != nil {
		input.IssueDate = overrides.IssueDate.UTC()
	}
	now := pkg.UTCNow()
	reference, err := service.references.Allocate(ctx, ownerID, input.IssueDate.Year(), now)
	if err != nil {
		return nil, err
	}
	document, err := model.NewDocument(ownerID, reference, input, now)
	if err != nil {
		return nil, err
	}
	sourceID := source.ID
	if err := service.persistWithAudit(ctx, document, model.AuditDocumentDuplicated, model.AuditMetadata{ChangedFields: []string{"title", "customer", "issueDate", "currency", "lineItems", "totals", "taxBreakdown"}, SourceDocumentID: &sourceID}, func(txCtx context.Context) error { return service.documents.Create(txCtx, document) }); err != nil {
		return nil, err
	}
	return &document, nil
}

func (service *DocumentService) loadForMutation(ctx context.Context, documentID string, expectedVersion int64) (*model.Document, error) {
	if expectedVersion < 1 {
		return nil, buhari.New(buhari.CodeDocumentVersionConflict, "A positive expected document version is required.")
	}
	document, err := service.GetByID(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if document.Version != expectedVersion {
		return nil, buhari.New(buhari.CodeDocumentVersionConflict, "Document was changed by another request.")
	}
	return document, nil
}

func (service *DocumentService) replaceLoadedDraft(ctx context.Context, document *model.Document, expectedVersion int64, input model.DocumentInput) (*model.Document, error) {
	if err := document.ReplaceDraft(input, pkg.UTCNow()); err != nil {
		return nil, err
	}
	if err := service.persistWithAudit(ctx, *document, model.AuditDocumentUpdated, documentChangedMetadata(), func(txCtx context.Context) error {
		return service.documents.ReplaceDraft(txCtx, *document, expectedVersion)
	}); err != nil {
		return nil, err
	}
	return document, nil
}

func documentChangedMetadata() model.AuditMetadata {
	return model.AuditMetadata{ChangedFields: []string{"title", "customer", "issueDate", "currency", "lineItems", "totals", "taxBreakdown"}}
}

func (service *DocumentService) persistWithAudit(ctx context.Context, document model.Document, action model.AuditAction, metadata model.AuditMetadata, persist func(context.Context) error) error {
	if service.audits == nil || service.tx == nil {
		return persist(ctx)
	}
	executor, ok := requestctx.Executor(ctx)
	requestID, hasRequestID := requestctx.RequestID(ctx)
	if !ok || !hasRequestID {
		return buhari.New(buhari.CodeInternalError, "Audit context is unavailable.")
	}
	event, err := model.NewAuditEvent(document.OwnerID, string(executor.ID), requestID, document, action, metadata, pkg.UTCNow())
	if err != nil {
		return err
	}
	return service.tx.RunInTx(ctx, func(txCtx context.Context) error {
		if err := persist(txCtx); err != nil {
			return err
		}
		return service.audits.Append(txCtx, event)
	})
}

func authenticatedOwner(ctx context.Context) (string, error) {
	executor, ok := requestctx.Executor(ctx)
	if !ok || lid.Validate(string(executor.ID), lid.User) != nil {
		return "", buhari.New(buhari.CodeUnauthorized, "Authenticated user context is required.")
	}
	return string(executor.ID), nil
}

func duplicateLineInputs(lines []calculations.LineResult) []calculations.LineInput {
	inputs := make([]calculations.LineInput, len(lines))
	for index, line := range lines {
		inputs[index] = line.LineInput
		inputs[index].ID = ""
		if line.Discount != nil {
			discount := *line.Discount
			inputs[index].Discount = &discount
		}
	}
	return inputs
}

func editableDocumentInput(document model.Document) model.DocumentInput {
	return model.DocumentInput{
		Title: document.Title, Customer: document.Customer, IssueDate: document.IssueDate,
		Currency: document.Currency, LineItems: cloneLineResults(document.LineItems),
	}
}

func cloneLineResults(lines []calculations.LineResult) []calculations.LineInput {
	inputs := make([]calculations.LineInput, len(lines))
	for index, line := range lines {
		inputs[index] = cloneLineInput(line.LineInput)
	}
	return inputs
}

func cloneLineInput(input calculations.LineInput) calculations.LineInput {
	copy := input
	if input.Discount != nil {
		discount := *input.Discount
		copy.Discount = &discount
	}
	return copy
}

func findLine(lines []calculations.LineInput, lineID string) int {
	for index := range lines {
		if lines[index].ID == lineID {
			return index
		}
	}
	return -1
}

func lineItemNotFound() error {
	return buhari.New(buhari.CodeNotFound, "Line item not found.")
}

func invalidLineOrder() error {
	return buhari.New(buhari.CodeInvalidLineOrder, "Line order must contain every existing line item exactly once.")
}

func validateDocumentListFilter(filter mercury.DocumentListFilter) error {
	fields := make([]buhari.FieldError, 0, 6)
	if filter.Status != nil && *filter.Status != model.DocumentDraft && *filter.Status != model.DocumentFinalized {
		fields = append(fields, buhari.FieldError{Path: "status", Code: buhari.CodeInvalidFilter, Message: "Status must be draft or finalized."})
	}
	if filter.Currency != nil && !filter.Currency.Valid() {
		fields = append(fields, buhari.FieldError{Path: "currency", Code: buhari.CodeInvalidFilter, Message: "Currency is not supported."})
	}
	if filter.IssueFrom != nil && filter.IssueTo != nil && filter.IssueFrom.After(*filter.IssueTo) {
		fields = append(fields, buhari.FieldError{Path: "from", Code: buhari.CodeInvalidFilter, Message: "From date cannot be after to date."})
	}
	if filter.Limit < 0 || filter.Limit > mercury.MaximumDocumentLimit {
		fields = append(fields, buhari.FieldError{Path: "limit", Code: buhari.CodeInvalidFilter, Message: "Limit must be between 1 and 100 when provided."})
	}
	if filter.Page < 0 {
		fields = append(fields, buhari.FieldError{Path: "page", Code: buhari.CodeInvalidFilter, Message: "Page must be a positive integer when provided."})
	}
	if filter.Sort != "" && filter.Sort != mercury.DocumentSortIssueDateDesc {
		fields = append(fields, buhari.FieldError{Path: "sort", Code: buhari.CodeInvalidFilter, Message: "Sort is not supported."})
	}
	searchLength := len([]rune(strings.TrimSpace(filter.Search)))
	if searchLength == 1 || searchLength > 100 {
		fields = append(fields, buhari.FieldError{Path: "search", Code: buhari.CodeInvalidFilter, Message: "Search must contain between 2 and 100 characters."})
	}
	if len(fields) > 0 {
		return buhari.InvalidFilter(fields...)
	}
	return nil
}

func requireActiveDraft(document model.Document) error {
	if document.Status == model.DocumentFinalized {
		return buhari.New(buhari.CodeDocumentFinalized, "Finalized documents are immutable.")
	}
	if document.ArchivedAt != nil {
		return buhari.New(buhari.CodeDocumentArchived, "Archived documents must be restored before editing.")
	}
	return nil
}
