package sys

import (
	"context"
	"errors"
	"testing"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/cfg"
	"github.com/babafemi99/daftar/backend/internal/model"
)

type fakeRegistrar struct {
	calls   int
	request model.CreateUserRequest
	err     error
}

func (f *fakeRegistrar) Register(_ context.Context, request *model.CreateUserRequest) (*model.User, error) {
	f.calls++
	f.request = *request
	return &model.User{}, f.err
}

func TestEnsureBootstrapUser(t *testing.T) {
	registrar := &fakeRegistrar{}
	bootstrap := cfg.BootstrapUser{
		Enabled: true, Email: "owner@example.com", Password: "strong-password", FirstName: "Ada", LastName: "Lovelace",
	}
	if err := ensureBootstrapUser(context.Background(), registrar, bootstrap); err != nil {
		t.Fatalf("ensureBootstrapUser() error = %v", err)
	}
	if registrar.calls != 1 || registrar.request.Email != bootstrap.Email {
		t.Fatalf("registrar = %#v", registrar)
	}
}

func TestEnsureBootstrapUserIsIdempotent(t *testing.T) {
	registrar := &fakeRegistrar{err: buhari.New(buhari.CodeEmailAlreadyRegistered, "already exists")}
	if err := ensureBootstrapUser(context.Background(), registrar, cfg.BootstrapUser{Enabled: true}); err != nil {
		t.Fatalf("ensureBootstrapUser() error = %v", err)
	}

	cause := errors.New("database unavailable")
	registrar.err = cause
	if err := ensureBootstrapUser(context.Background(), registrar, cfg.BootstrapUser{Enabled: true}); !errors.Is(err, cause) {
		t.Fatalf("ensureBootstrapUser() error = %v", err)
	}
}
