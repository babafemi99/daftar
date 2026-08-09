package service

import (
	"context"
	"errors"
	"testing"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/argon"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
	"github.com/babafemi99/daftar/backend/internal/pkg/requestctx"
)

type fakeUsers struct {
	created      model.User
	byID         model.User
	byEmail      model.User
	createErr    error
	findByIDErr  error
	findEmailErr error
}

func (f *fakeUsers) Create(_ context.Context, user model.User) error {
	f.created = user
	return f.createErr
}
func (f *fakeUsers) FindByID(context.Context, string) (model.User, error) {
	return f.byID, f.findByIDErr
}
func (f *fakeUsers) FindByEmail(context.Context, string) (model.User, error) {
	return f.byEmail, f.findEmailErr
}

func validRequest() model.CreateUserRequest {
	return model.CreateUserRequest{
		Email: " User@Example.COM ", Password: "strong-password", FirstName: "Ada", LastName: "Lovelace",
	}
}

func notFound() error {
	return buhari.New(buhari.CodeNotFound, "User not found.")
}

func TestNewUserServiceRequiresRepository(t *testing.T) {
	service, err := NewUserService(nil)
	if !errors.Is(err, ErrUserRepositoryRequired) || service != nil {
		t.Fatalf("NewUserService(nil) = %#v, %v", service, err)
	}
}

func TestUserServiceRegister(t *testing.T) {
	repository := &fakeUsers{findEmailErr: notFound()}
	service, _ := NewUserService(repository)
	request := validRequest()

	user, err := service.Register(context.Background(), &request)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if user.Email != "user@example.com" || user.PasswordHash == "" || repository.created != *user {
		t.Fatalf("user = %#v, created = %#v", user, repository.created)
	}
	if matched, err := argon.Verify(request.Password, user.PasswordHash); err != nil || !matched {
		t.Fatalf("stored password hash did not verify: matched=%v err=%v", matched, err)
	}
}

func TestUserServiceRegisterRejectsNilAndExistingUser(t *testing.T) {
	service, _ := NewUserService(&fakeUsers{})
	if _, err := service.Register(context.Background(), nil); !hasCode(err, buhari.CodeValidationFailed) {
		t.Fatalf("Register(nil) error = %v", err)
	}

	request := validRequest()
	if _, err := service.Register(context.Background(), &request); !hasCode(err, buhari.CodeEmailAlreadyRegistered) {
		t.Fatalf("Register(existing) error = %v", err)
	}
}

func TestUserServiceLogin(t *testing.T) {
	hash, err := argon.Hash("strong-password")
	if err != nil {
		t.Fatalf("argon.Hash() error = %v", err)
	}
	user := model.User{ID: lid.NewUser(), Email: "user@example.com", PasswordHash: hash, FirstName: "Ada", LastName: "Lovelace"}
	service, _ := NewUserService(&fakeUsers{byEmail: user})
	request := model.LoginRequest{Email: "USER@example.com", Password: "strong-password"}

	got, err := service.Login(context.Background(), &request)
	if err != nil || got == nil || *got != user {
		t.Fatalf("Login() = %#v, %v", got, err)
	}
}

func TestUserServiceLoginRejectsNilAndInvalidCredentials(t *testing.T) {
	service, _ := NewUserService(&fakeUsers{findEmailErr: notFound()})
	if _, err := service.Login(context.Background(), nil); !hasCode(err, buhari.CodeValidationFailed) {
		t.Fatalf("Login(nil) error = %v", err)
	}

	request := model.LoginRequest{Email: "user@example.com", Password: "wrong-password"}
	if _, err := service.Login(context.Background(), &request); !hasCode(err, buhari.CodeUnauthorized) {
		t.Fatalf("Login() error = %v", err)
	}
}

func TestUserServiceGetByIDEnforcesJWTSubject(t *testing.T) {
	ownerID := lid.NewUser()
	otherID := lid.NewUser()
	user := model.User{ID: ownerID, Email: "user@example.com"}
	service, _ := NewUserService(&fakeUsers{byID: user})

	ctx := requestctx.WithExecutor(context.Background(), model.Executor{ID: model.UserID(ownerID), Email: "user@example.com", Role: "user"})
	got, err := service.GetByID(ctx, ownerID)
	if err != nil || got == nil || got.ID != ownerID {
		t.Fatalf("GetByID(owner) = %#v, %v", got, err)
	}
	if _, err := service.GetByID(ctx, otherID); !hasCode(err, buhari.CodeNotFound) {
		t.Fatalf("GetByID(other) error = %v", err)
	}
	if _, err := service.GetByID(context.Background(), ownerID); !hasCode(err, buhari.CodeUnauthorized) {
		t.Fatalf("GetByID(missing executor) error = %v", err)
	}
}
