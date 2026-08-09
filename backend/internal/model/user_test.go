package model

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func validCreateUserRequest() CreateUserRequest {
	return CreateUserRequest{
		Email:     " Finance@Example.COM ",
		Password:  "strong-password",
		FirstName: " Finance ",
		LastName:  " User ",
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail(" Finance@Example.COM "); got != "finance@example.com" {
		t.Fatalf("NormalizeEmail() = %q", got)
	}
}

func TestCreateUserRequestValidationCollectsFields(t *testing.T) {
	err := (CreateUserRequest{}).Validate()
	var appErr *buhari.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("Validate() error = %T, want *buhari.Error", err)
	}
	if appErr.Code != buhari.CodeValidationFailed || len(appErr.Fields) != 4 {
		t.Fatalf("Validate() = %#v, want four field errors", appErr)
	}
}

func TestNewUser(t *testing.T) {
	now := time.Date(2026, time.August, 8, 10, 0, 0, 0, time.FixedZone("WAT", 60*60))
	user, err := NewUser(validCreateUserRequest(), "argon2id-hash", now)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	if err := lid.Validate(user.ID, lid.User); err != nil {
		t.Errorf("ID = %q: %v", user.ID, err)
	}
	if user.Email != "finance@example.com" || user.FirstName != "Finance" || user.LastName != "User" {
		t.Errorf("user = %#v", user)
	}
	if !user.CreatedAt.Equal(now.UTC()) || user.CreatedAt.Location() != time.UTC {
		t.Errorf("CreatedAt = %v", user.CreatedAt)
	}
}

func TestNewUserRequiresPasswordHash(t *testing.T) {
	_, err := NewUser(validCreateUserRequest(), "", time.Now())
	if !errors.Is(err, ErrPasswordHashRequired) {
		t.Fatalf("NewUser() error = %v, want ErrPasswordHashRequired", err)
	}
}

func TestUserPersistenceShapeAndJSONSafety(t *testing.T) {
	user, err := NewUser(validCreateUserRequest(), "secret-hash", time.Now())
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	encodedBSON, err := bson.Marshal(user)
	if err != nil {
		t.Fatalf("bson.Marshal() error = %v", err)
	}
	var stored bson.M
	if err := bson.Unmarshal(encodedBSON, &stored); err != nil {
		t.Fatalf("bson.Unmarshal() error = %v", err)
	}
	if stored["_id"] != user.ID || stored["email"] != "finance@example.com" || stored["passwordHash"] != user.PasswordHash {
		t.Fatalf("stored user = %#v", stored)
	}

	encodedJSON, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(encodedJSON), "secret-hash") {
		t.Fatalf("JSON exposes password hash: %s", encodedJSON)
	}
}
