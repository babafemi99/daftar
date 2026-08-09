package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/babafemi99/daftar/backend/internal/buhari"
)

type ResponseEnvelope struct {
	Data      any    `json:"data"`
	RequestID string `json:"requestId"`
}

type CollectionResponseEnvelope struct {
	Data      any            `json:"data"`
	Page      CollectionPage `json:"page"`
	RequestID string         `json:"requestId"`
}

type CollectionPage struct {
	NextCursor *string `json:"nextCursor"`
	HasMore    bool    `json:"hasMore"`
	Number     int64   `json:"number,omitempty"`
	Size       int64   `json:"size,omitempty"`
	TotalItems int64   `json:"totalItems"`
	TotalPages int64   `json:"totalPages"`
}

type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code      buhari.Code `json:"code"`
	Message   string      `json:"message"`
	Details   any         `json:"details,omitempty"`
	RequestID string      `json:"requestId"`
}

// Response writes the standard successful API envelope.
func Response(ctx context.Context, w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, ResponseEnvelope{
		Data:      data,
		RequestID: RequestIDFromContext(ctx),
	})
}

// CollectionResponse writes the standard collection envelope. The take-home
// document list is bounded but not cursor-paginated, so its page is terminal.
func CollectionResponse(ctx context.Context, w http.ResponseWriter, status int, data any) {
	CollectionResponseWithPage(ctx, w, status, data, CollectionPage{})
}

func CollectionResponseWithPage(ctx context.Context, w http.ResponseWriter, status int, data any, page CollectionPage) {
	writeJSON(w, status, CollectionResponseEnvelope{
		Data:      data,
		Page:      page,
		RequestID: RequestIDFromContext(ctx),
	})
}

// ResponseError converts an application error to its HTTP representation.
// Causes are intentionally omitted from the response.
func ResponseError(ctx context.Context, w http.ResponseWriter, err error) {
	appErr := asApplicationError(err)
	setResponseErrorCode(w, appErr.Code)
	body := ErrorBody{
		Code:      appErr.Code,
		Message:   appErr.Message,
		RequestID: RequestIDFromContext(ctx),
	}
	if len(appErr.Fields) > 0 {
		body.Details = map[string]any{"fields": appErr.Fields}
	}

	writeJSON(w, statusFor(appErr.Code), ErrorEnvelope{Error: body})
}

func asApplicationError(err error) *buhari.Error {
	if appErr, ok := errors.AsType[*buhari.Error](err); ok {
		return appErr
	}
	return buhari.Wrap(buhari.CodeInternalError, "An unexpected error occurred.", err)
}

func statusFor(code buhari.Code) int {
	switch code {
	case buhari.CodeUnauthorized:
		return http.StatusUnauthorized
	case buhari.CodeForbidden:
		return http.StatusForbidden
	case buhari.CodeNotFound:
		return http.StatusNotFound
	case buhari.CodeIdempotencyKeyRequired:
		return http.StatusBadRequest
	case buhari.CodeInvalidIfMatch:
		return http.StatusBadRequest
	case buhari.CodeConflict,
		buhari.CodeEmailAlreadyRegistered,
		buhari.CodeDocumentFinalized,
		buhari.CodeDocumentAlreadyFinalized,
		buhari.CodeDocumentArchived,
		buhari.CodeDocumentVersionConflict,
		buhari.CodeDocumentNotArchived,
		buhari.CodeSourceDocumentNotFinalized,
		buhari.CodeIdempotencyKeyReused:
		return http.StatusConflict
	case buhari.CodeValidationFailed,
		buhari.CodeInvalidFilter,
		buhari.CodeInvalidEmail,
		buhari.CodeInvalidPassword,
		buhari.CodeInvalidFirstName,
		buhari.CodeInvalidLastName,
		buhari.CodeInvalidCurrency,
		buhari.CodeInvalidMoneyFormat,
		buhari.CodeInvalidQuantity,
		buhari.CodeInvalidTaxRate,
		buhari.CodeInvalidDiscount,
		buhari.CodeInvalidDiscountType,
		buhari.CodeDiscountTooLarge,
		buhari.CodeMonetaryOverflow,
		buhari.CodeDocumentRequiresLine,
		buhari.CodeInvalidLineOrder:
		return http.StatusUnprocessableEntity
	case buhari.CodeInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
