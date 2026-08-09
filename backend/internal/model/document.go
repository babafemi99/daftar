package model

import (
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/calculations"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
)

const (
	DocumentCollection        = "documents"
	DocumentCounterCollection = "document_counters"
	maximumTitleLength        = 160
	maximumCustomerLength     = 200
)

type DocumentStatus string
type Currency string

const (
	DocumentDraft     DocumentStatus = "draft"
	DocumentFinalized DocumentStatus = "finalized"

	CurrencyUSD Currency = "USD"
	CurrencyAED Currency = "AED"
	CurrencySAR Currency = "SAR"
	CurrencyNGN Currency = "NGN"
	CurrencyGBP Currency = "GBP"
	CurrencyEUR Currency = "EUR"
)

var documentReference = regexp.MustCompile(`^DOC-[0-9]{4}-[0-9]{6}$`)

type Document struct {
	ID                       string                      `bson:"_id" json:"id"`
	OwnerID                  string                      `bson:"ownerId" json:"-"`
	Reference                string                      `bson:"reference" json:"reference"`
	Title                    string                      `bson:"title" json:"title"`
	Customer                 string                      `bson:"customer" json:"customer"`
	IssueDate                time.Time                   `bson:"issueDate" json:"issueDate"`
	Currency                 Currency                    `bson:"currency" json:"currency"`
	Status                   DocumentStatus              `bson:"status" json:"status"`
	LineItems                []calculations.LineResult   `bson:"lineItems" json:"lineItems"`
	Totals                   calculations.Totals         `bson:"totals" json:"totals"`
	TaxBreakdown             []calculations.TaxBreakdown `bson:"taxBreakdown" json:"taxBreakdown"`
	CalculationPolicyVersion string                      `bson:"calculationPolicyVersion" json:"calculationPolicyVersion"`
	Version                  int64                       `bson:"version" json:"version"`
	FinalizedAt              *time.Time                  `bson:"finalizedAt" json:"finalizedAt"`
	ArchivedAt               *time.Time                  `bson:"archivedAt" json:"archivedAt"`
	CreatedAt                time.Time                   `bson:"createdAt" json:"createdAt"`
	UpdatedAt                time.Time                   `bson:"updatedAt" json:"updatedAt"`
}

type DocumentInput struct {
	Title     string
	Customer  string
	IssueDate time.Time
	Currency  Currency
	LineItems []calculations.LineInput
}

func NewDocument(ownerID, reference string, input DocumentInput, now time.Time) (Document, error) {
	input.LineItems = freshLineInputs(input.LineItems)
	document := Document{
		ID:        lid.NewDocument(),
		OwnerID:   ownerID,
		Reference: strings.TrimSpace(reference),
		Status:    DocumentDraft,
		Version:   1,
		CreatedAt: now.UTC(),
		UpdatedAt: now.UTC(),
	}
	if err := document.applyInput(input); err != nil {
		return Document{}, err
	}
	if err := document.Validate(); err != nil {
		return Document{}, err
	}
	return document, nil
}

// ReplaceDraft replaces every client-editable field and recalculates the
// complete aggregate. Existing line IDs must be preserved by callers.
func (document *Document) ReplaceDraft(input DocumentInput, now time.Time) error {
	if err := document.requireActiveDraft(); err != nil {
		return err
	}
	lines, err := document.reconcileLineIDs(input.LineItems)
	if err != nil {
		return err
	}
	input.LineItems = lines
	copy := *document
	if err := copy.applyInput(input); err != nil {
		return err
	}
	copy.Version++
	copy.UpdatedAt = now.UTC()
	if err := copy.Validate(); err != nil {
		return err
	}
	*document = copy
	return nil
}

func (document Document) reconcileLineIDs(lines []calculations.LineInput) ([]calculations.LineInput, error) {
	existing := make(map[string]struct{}, len(document.LineItems))
	for _, line := range document.LineItems {
		existing[line.ID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(lines))
	result := cloneLineInputs(lines)
	for index, line := range result {
		if line.ID == "" {
			continue
		}
		path := "lineItems[" + indexString(index) + "].id"
		if _, duplicate := seen[line.ID]; duplicate {
			return nil, buhari.Validation(buhari.FieldError{Path: path, Code: buhari.CodeValidationFailed, Message: "Line item IDs must be unique."})
		}
		seen[line.ID] = struct{}{}
		if _, belongs := existing[line.ID]; !belongs {
			return nil, buhari.Validation(buhari.FieldError{Path: path, Code: buhari.CodeValidationFailed, Message: "Line item ID does not belong to this document."})
		}
	}
	return result, nil
}

func (document *Document) Finalize(now time.Time) error {
	if err := document.requireActiveDraft(); err != nil {
		return err
	}
	if len(document.LineItems) == 0 {
		return buhari.New(buhari.CodeDocumentRequiresLine, "A document requires at least one line item before finalization.")
	}
	CP := *document
	if err := CP.recalculate(lineInputs(CP.LineItems)); err != nil {
		return err
	}
	timestamp := now.UTC()
	CP.Status = DocumentFinalized
	CP.FinalizedAt = &timestamp
	CP.Version++
	CP.UpdatedAt = timestamp
	if err := CP.Validate(); err != nil {
		return err
	}
	*document = CP
	return nil
}

func (document *Document) Archive(now time.Time) error {
	if document.Status == DocumentFinalized {
		return buhari.New(buhari.CodeDocumentFinalized, "Finalized documents cannot be archived.")
	}
	if document.ArchivedAt != nil {
		return buhari.New(buhari.CodeDocumentArchived, "Document is already archived.")
	}
	CP := *document
	timestamp := now.UTC()
	CP.ArchivedAt = &timestamp
	CP.Version++
	CP.UpdatedAt = timestamp
	if err := CP.Validate(); err != nil {
		return err
	}
	*document = CP
	return nil
}

func (document *Document) Restore(now time.Time) error {
	if document.Status == DocumentFinalized {
		return buhari.New(buhari.CodeDocumentFinalized, "Finalized documents cannot be restored as drafts.")
	}
	if document.ArchivedAt == nil {
		return buhari.New(buhari.CodeDocumentNotArchived, "Document is not archived.")
	}
	copy := *document
	copy.ArchivedAt = nil
	copy.Version++
	copy.UpdatedAt = now.UTC()
	if err := copy.Validate(); err != nil {
		return err
	}
	*document = copy
	return nil
}

func (document *Document) Validate() error {
	var fields []buhari.FieldError
	if lid.Validate(document.ID, lid.Document) != nil {
		fields = append(fields, field("id", "Document ID is invalid."))
	}
	if lid.Validate(document.OwnerID, lid.User) != nil {
		fields = append(fields, field("ownerId", "Document owner is invalid."))
	}
	if !documentReference.MatchString(document.Reference) {
		fields = append(fields, field("reference", "Document reference is invalid."))
	}
	if !validText(document.Title, maximumTitleLength) {
		fields = append(fields, field("title", "Title must be between 1 and 160 characters."))
	}
	if !validText(document.Customer, maximumCustomerLength) {
		fields = append(fields, field("customer", "Customer must be between 1 and 200 characters."))
	}
	if !validIssueDate(document.IssueDate) {
		fields = append(fields, field("issueDate", "Issue date must be a UTC calendar date."))
	}
	if !document.Currency.Valid() {
		fields = append(fields, buhari.FieldError{Path: "currency", Code: buhari.CodeInvalidCurrency, Message: "Currency is not supported."})
	}
	if document.Status != DocumentDraft && document.Status != DocumentFinalized {
		fields = append(fields, field("status", "Document status is invalid."))
	}
	if document.Version < 1 {
		fields = append(fields, field("version", "Document version must be positive."))
	}
	if document.CreatedAt.IsZero() || document.UpdatedAt.IsZero() || document.UpdatedAt.Before(document.CreatedAt) {
		fields = append(fields, field("timestamps", "Document timestamps are invalid."))
	}
	if document.Status == DocumentFinalized && document.FinalizedAt == nil {
		fields = append(fields, field("finalizedAt", "Finalized documents require a finalized timestamp."))
	}
	if document.Status == DocumentDraft && document.FinalizedAt != nil {
		fields = append(fields, field("finalizedAt", "Draft documents cannot have a finalized timestamp."))
	}
	if document.Status == DocumentFinalized && document.ArchivedAt != nil {
		fields = append(fields, field("archivedAt", "Finalized documents cannot be archived."))
	}
	if len(fields) > 0 {
		return buhari.Validation(fields...)
	}

	seenLines := make(map[string]struct{}, len(document.LineItems))
	for index, line := range document.LineItems {
		if lid.Validate(line.ID, lid.Line) != nil {
			return buhari.Validation(buhari.FieldError{Path: "lineItems[" + indexString(index) + "].id", Code: buhari.CodeValidationFailed, Message: "Line item ID is invalid."})
		}
		if _, exists := seenLines[line.ID]; exists {
			return buhari.Validation(buhari.FieldError{Path: "lineItems[" + indexString(index) + "].id", Code: buhari.CodeValidationFailed, Message: "Line item IDs must be unique."})
		}
		seenLines[line.ID] = struct{}{}
	}

	recalculated, err := calculations.CalculateDocument(lineInputs(document.LineItems))
	if err != nil {
		return err
	}
	if document.CalculationPolicyVersion != calculations.PolicyVersion ||
		!reflect.DeepEqual(document.LineItems, recalculated.Lines) ||
		document.Totals != recalculated.Totals ||
		!reflect.DeepEqual(document.TaxBreakdown, recalculated.TaxBreakdown) {
		return buhari.New(buhari.CodeValidationFailed, "Stored document calculations do not match their inputs.")
	}
	return nil
}

func (currency Currency) Valid() bool {
	switch currency {
	case CurrencyUSD, CurrencyAED, CurrencySAR, CurrencyNGN, CurrencyGBP, CurrencyEUR:
		return true
	default:
		return false
	}
}

func (document *Document) applyInput(input DocumentInput) error {
	document.Title = strings.TrimSpace(input.Title)
	document.Customer = strings.TrimSpace(input.Customer)
	document.IssueDate = input.IssueDate.UTC()
	document.Currency = input.Currency
	lines := cloneLineInputs(input.LineItems)
	for index := range lines {
		lines[index].Description = strings.TrimSpace(lines[index].Description)
		if lines[index].ID == "" {
			lines[index].ID = lid.NewLine()
		}
	}
	return document.recalculate(lines)
}

func (document *Document) recalculate(lines []calculations.LineInput) error {
	result, err := calculations.CalculateDocument(lines)
	if err != nil {
		return err
	}
	document.LineItems = result.Lines
	document.Totals = result.Totals
	document.TaxBreakdown = result.TaxBreakdown
	document.CalculationPolicyVersion = result.PolicyVersion
	return nil
}

func (document *Document) requireActiveDraft() error {
	if document.Status == DocumentFinalized {
		return buhari.New(buhari.CodeDocumentFinalized, "Finalized documents are immutable.")
	}
	if document.ArchivedAt != nil {
		return buhari.New(buhari.CodeDocumentArchived, "Archived documents must be restored before editing.")
	}
	return nil
}

func lineInputs(lines []calculations.LineResult) []calculations.LineInput {
	inputs := make([]calculations.LineInput, len(lines))
	for index := range lines {
		inputs[index] = lines[index].LineInput
	}
	return inputs
}

func freshLineInputs(lines []calculations.LineInput) []calculations.LineInput {
	result := cloneLineInputs(lines)
	for index := range result {
		result[index].ID = ""
	}
	return result
}

func cloneLineInputs(lines []calculations.LineInput) []calculations.LineInput {
	result := make([]calculations.LineInput, len(lines))
	for index, line := range lines {
		result[index] = line
		if line.Discount != nil {
			discount := *line.Discount
			result[index].Discount = &discount
		}
	}
	return result
}

func validText(value string, maximum int) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed != "" && len([]rune(trimmed)) <= maximum
}

func validIssueDate(value time.Time) bool {
	if value.IsZero() || value.Location() != time.UTC {
		return false
	}
	return value.Hour() == 0 && value.Minute() == 0 && value.Second() == 0 && value.Nanosecond() == 0
}

func field(path, message string) buhari.FieldError {
	return buhari.FieldError{Path: path, Code: buhari.CodeValidationFailed, Message: message}
}

func indexString(value int) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	position := len(buffer)
	for value > 0 {
		position--
		buffer[position] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[position:])
}
