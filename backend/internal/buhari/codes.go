package buhari

// Code is a stable, machine-readable error identifier shared across domain
// packages. Code values form part of the public API contract.
type Code string

const (
	CodeValidationFailed       Code = "VALIDATION_FAILED"
	CodeInternalError          Code = "INTERNAL_ERROR"
	CodeNotFound               Code = "RESOURCE_NOT_FOUND"
	CodeUnauthorized           Code = "UNAUTHORIZED"
	CodeForbidden              Code = "FORBIDDEN"
	CodeConflict               Code = "CONFLICT"
	CodeEmailAlreadyRegistered Code = "EMAIL_ALREADY_REGISTERED"
	CodeInvalidEmail           Code = "INVALID_EMAIL"
	CodeInvalidPassword        Code = "INVALID_PASSWORD"
	CodeInvalidFirstName       Code = "INVALID_FIRST_NAME"
	CodeInvalidLastName        Code = "INVALID_LAST_NAME"
	CodeInvalidCurrency        Code = "INVALID_CURRENCY"
	CodeInvalidDiscountType    Code = "INVALID_DISCOUNT_TYPE"
	CodeInvalidFilter          Code = "INVALID_FILTER"
	CodeInvalidIfMatch         Code = "INVALID_IF_MATCH"

	CodeInvalidMoneyFormat Code = "INVALID_MONEY_FORMAT"
	CodeInvalidQuantity    Code = "INVALID_QUANTITY"
	CodeInvalidTaxRate     Code = "INVALID_TAX_RATE"
	CodeInvalidDiscount    Code = "INVALID_DISCOUNT_RATE"
	CodeDiscountTooLarge   Code = "FIXED_DISCOUNT_EXCEEDS_SUBTOTAL"
	CodeMonetaryOverflow   Code = "MONETARY_OVERFLOW"

	CodeDocumentFinalized          Code = "DOCUMENT_FINALIZED"
	CodeDocumentAlreadyFinalized   Code = "DOCUMENT_ALREADY_FINALIZED"
	CodeDocumentArchived           Code = "DOCUMENT_ARCHIVED"
	CodeDocumentVersionConflict    Code = "DOCUMENT_VERSION_CONFLICT"
	CodeDocumentRequiresLine       Code = "DOCUMENT_REQUIRES_LINE"
	CodeInvalidLineOrder           Code = "INVALID_LINE_ORDER"
	CodeSourceDocumentNotFinalized Code = "SOURCE_DOCUMENT_NOT_FINALIZED"
	CodeDocumentNotArchived        Code = "DOCUMENT_NOT_ARCHIVED"

	CodeIdempotencyKeyRequired Code = "IDEMPOTENCY_KEY_REQUIRED"
	CodeIdempotencyKeyReused   Code = "IDEMPOTENCY_KEY_REUSED"
)
