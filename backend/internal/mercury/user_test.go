package mercury

import (
	"errors"
	"testing"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestMapUserReadError(t *testing.T) {
	err := mapUserReadError(mongo.ErrNoDocuments)

	var appErr *buhari.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error type = %T, want *buhari.Error", err)
	}
	if appErr.Code != buhari.CodeNotFound {
		t.Errorf("Code = %q, want %q", appErr.Code, buhari.CodeNotFound)
	}
}

func TestMapUserReadErrorWrapsCause(t *testing.T) {
	cause := errors.New("connection lost")
	err := mapUserReadError(cause)

	var appErr *buhari.Error
	if !errors.As(err, &appErr) || appErr.Code != buhari.CodeInternalError {
		t.Fatalf("error = %#v, want internal *buhari.Error", err)
	}
	if !errors.Is(err, cause) {
		t.Fatal("internal error does not wrap its cause")
	}
}

func TestMapUserWriteDuplicateError(t *testing.T) {
	err := mapUserWriteError(mongo.CommandError{Code: 11000, Message: "duplicate key"})

	var appErr *buhari.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error type = %T, want *buhari.Error", err)
	}
	if appErr.Code != buhari.CodeEmailAlreadyRegistered {
		t.Errorf("Code = %q, want %q", appErr.Code, buhari.CodeEmailAlreadyRegistered)
	}
}
