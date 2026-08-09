package buhari

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(CodeNotFound, "The requested resource was not found.")

	if err.Code != CodeNotFound {
		t.Errorf("Code = %q, want %q", err.Code, CodeNotFound)
	}
	if err.Message != "The requested resource was not found." {
		t.Errorf("Message = %q", err.Message)
	}
	if got := err.Error(); got != "RESOURCE_NOT_FOUND: The requested resource was not found." {
		t.Errorf("Error() = %q", got)
	}
}

func TestWrapSupportsErrorInspection(t *testing.T) {
	cause := errors.New("database unavailable")
	err := Wrap(CodeInternalError, "An unexpected error occurred.", cause)

	if !errors.Is(err, cause) {
		t.Fatal("errors.Is() = false, want true for wrapped cause")
	}

	var appErr *Error
	if !errors.As(err, &appErr) {
		t.Fatal("errors.As() = false, want true for *Error")
	}
	if appErr.Cause != cause {
		t.Errorf("Cause = %v, want %v", appErr.Cause, cause)
	}
}

func TestValidation(t *testing.T) {
	fields := []FieldError{
		{
			Path:    "lineItems[0].quantity",
			Code:    CodeInvalidQuantity,
			Message: "Quantity must be greater than or equal to 1.",
		},
		{
			Path:    "lineItems[0].taxRate",
			Code:    CodeInvalidTaxRate,
			Message: "Tax rate must be between 0 and 100.",
		},
	}

	err := Validation(fields...)

	if err.Code != CodeValidationFailed {
		t.Errorf("Code = %q, want %q", err.Code, CodeValidationFailed)
	}
	if err.Message != validationMessage {
		t.Errorf("Message = %q, want %q", err.Message, validationMessage)
	}
	if len(err.Fields) != 2 {
		t.Fatalf("len(Fields) = %d, want 2", len(err.Fields))
	}
	if err.Fields[0] != fields[0] || err.Fields[1] != fields[1] {
		t.Errorf("Fields = %#v, want %#v", err.Fields, fields)
	}
}

func TestNilErrorMethods(t *testing.T) {
	var err *Error

	if got := err.Error(); got != "" {
		t.Errorf("Error() = %q, want empty string", got)
	}
	if got := err.Unwrap(); got != nil {
		t.Errorf("Unwrap() = %v, want nil", got)
	}
}
