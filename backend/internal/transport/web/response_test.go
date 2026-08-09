package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/pkg/requestctx"
)

func TestResponse(t *testing.T) {
	ctx := requestctx.WithRequestID(context.Background(), "req-test")
	recorder := httptest.NewRecorder()

	Response(ctx, recorder, http.StatusCreated, map[string]string{"id": "user-test"})

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d", recorder.Code)
	}
	var response ResponseEnvelope
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.RequestID != "req-test" {
		t.Fatalf("response = %#v", response)
	}
}

func TestResponseErrorMapsApplicationError(t *testing.T) {
	ctx := requestctx.WithRequestID(context.Background(), "req-test")
	recorder := httptest.NewRecorder()
	err := buhari.Validation(buhari.FieldError{Path: "email", Code: buhari.CodeInvalidEmail, Message: "Invalid email."})

	ResponseError(ctx, recorder, err)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", recorder.Code)
	}
	var response ErrorEnvelope
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error.Code != buhari.CodeValidationFailed || response.Error.RequestID != "req-test" || response.Error.Details == nil {
		t.Fatalf("response = %#v", response)
	}
}

func TestResponseErrorHidesUnknownError(t *testing.T) {
	recorder := httptest.NewRecorder()
	ResponseError(context.Background(), recorder, errors.New("database password leaked"))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", recorder.Code)
	}
	if body := recorder.Body.String(); body == "" || contains(body, "database password leaked") {
		t.Fatalf("response exposes cause: %s", body)
	}
}

func TestStatusForDocumentApplicationCodes(t *testing.T) {
	tests := map[buhari.Code]int{
		buhari.CodeNotFound:                   http.StatusNotFound,
		buhari.CodeDocumentAlreadyFinalized:   http.StatusConflict,
		buhari.CodeSourceDocumentNotFinalized: http.StatusConflict,
		buhari.CodeInvalidLineOrder:           http.StatusUnprocessableEntity,
		buhari.CodeInvalidCurrency:            http.StatusUnprocessableEntity,
		buhari.CodeInvalidDiscountType:        http.StatusUnprocessableEntity,
		buhari.CodeInvalidFilter:              http.StatusUnprocessableEntity,
		buhari.CodeDocumentNotArchived:        http.StatusConflict,
		buhari.CodeInternalError:              http.StatusInternalServerError,
	}
	for code, expected := range tests {
		if actual := statusFor(code); actual != expected {
			t.Errorf("statusFor(%s)=%d want=%d", code, actual, expected)
		}
	}
}

func contains(value, substring string) bool {
	for i := 0; i+len(substring) <= len(value); i++ {
		if value[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}
