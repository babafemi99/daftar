package service

import (
	"context"
	"errors"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/mercury"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg"
	"github.com/babafemi99/daftar/backend/internal/pkg/argon"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
	"github.com/babafemi99/daftar/backend/internal/pkg/requestctx"
)

var ErrUserRepositoryRequired = errors.New("service: user repository is required")

type UserService struct {
	userRepo mercury.Users
}

func NewUserService(userRepo mercury.Users) (*UserService, error) {
	if userRepo == nil {
		return nil, ErrUserRepositoryRequired
	}

	return &UserService{
		userRepo: userRepo,
	}, nil
}

// Register creates a new user account.
func (s *UserService) Register(ctx context.Context, request *model.CreateUserRequest) (*model.User, error) {
	if request == nil {
		return nil, buhari.Validation(buhari.FieldError{
			Path:    "request",
			Code:    buhari.CodeValidationFailed,
			Message: "A registration request is required.",
		})
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}

	_, err := s.userRepo.FindByEmail(ctx, request.Email)
	if err == nil {
		return nil, emailAlreadyRegistered()
	}
	if !hasCode(err, buhari.CodeNotFound) {
		return nil, err
	}

	passwordHash, err := argon.Hash(request.Password)
	if err != nil {
		return nil, buhari.Wrap(buhari.CodeInternalError, "Unable to secure the password.", err)
	}

	user, err := model.NewUser(*request, passwordHash, pkg.UTCNow())
	if err != nil {
		return nil, err
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return &user, nil
}

// Login verifies login credentials. Missing users and incorrect
// passwords intentionally return the same public error.
func (s *UserService) Login(ctx context.Context, request *model.LoginRequest) (*model.User, error) {
	if request == nil {
		return nil, buhari.Validation(buhari.FieldError{
			Path:    "request",
			Code:    buhari.CodeValidationFailed,
			Message: "A login request is required.",
		})
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByEmail(ctx, request.Email)
	if err != nil {
		if hasCode(err, buhari.CodeNotFound) {
			return nil, invalidCredentials()
		}
		return nil, err
	}

	matched, err := argon.Verify(request.Password, user.PasswordHash)
	if err != nil {
		return nil, buhari.Wrap(buhari.CodeInternalError, "Unable to verify credentials.", err)
	}
	if !matched {
		return nil, invalidCredentials()
	}

	return &user, nil
}

func (s *UserService) GetByID(ctx context.Context, requestedUserID string) (*model.User, error) {
	executor, ok := requestctx.Executor(ctx)
	if !ok {
		return nil, buhari.New(buhari.CodeUnauthorized, "Authenticated user context is required.")
	}
	if err := lid.Validate(requestedUserID, lid.User); err != nil || string(executor.ID) != requestedUserID {
		return nil, userNotFound()
	}

	user, err := s.userRepo.FindByID(ctx, requestedUserID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func emailAlreadyRegistered() *buhari.Error {
	return buhari.New(buhari.CodeEmailAlreadyRegistered, "An account with this email already exists.")
}

func invalidCredentials() *buhari.Error {
	return buhari.New(buhari.CodeUnauthorized, "Invalid email or password.")
}

func userNotFound() *buhari.Error {
	return buhari.New(buhari.CodeNotFound, "User not found.")
}

func hasCode(err error, code buhari.Code) bool {
	var appErr *buhari.Error
	return errors.As(err, &appErr) && appErr.Code == code
}
