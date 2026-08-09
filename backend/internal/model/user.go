package model

import (
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
)

const (
	UserCollection    = "users"
	minimumPassword   = 8
	maximumPassword   = 128
	maximumNameLength = 100
)

var (
	ErrUserIDInvalid        = errors.New("user id is invalid")
	ErrEmailRequired        = errors.New("email is required")
	ErrEmailInvalid         = errors.New("email is invalid")
	ErrFirstNameRequired    = errors.New("first name is required")
	ErrLastNameRequired     = errors.New("last name is required")
	ErrPasswordHashRequired = errors.New("password hash is required")
)

// User is the persisted representation of a Daftar account.
type User struct {
	ID           string    `bson:"_id" json:"id"`
	Email        string    `bson:"email" json:"email"`
	PasswordHash string    `bson:"passwordHash" json:"-"`
	FirstName    string    `bson:"first_name" json:"first_name"`
	LastName     string    `bson:"last_name" json:"last_name"`
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time `bson:"updatedAt" json:"updatedAt"`
}

// Validate checks client-controlled registration input and returns every
// invalid field in one application error.
func (request CreateUserRequest) Validate() error {
	var fields []buhari.FieldError
	email := NormalizeEmail(request.Email)
	address, err := mail.ParseAddress(email)
	if email == "" || err != nil || address.Address != email {
		fields = append(fields, buhari.FieldError{
			Path:    "email",
			Code:    buhari.CodeInvalidEmail,
			Message: "Enter a valid email address.",
		})
	}

	passwordLength := len([]rune(request.Password))
	if passwordLength < minimumPassword || passwordLength > maximumPassword {
		fields = append(fields, buhari.FieldError{
			Path:    "password",
			Code:    buhari.CodeInvalidPassword,
			Message: "Password must be between 8 and 128 characters.",
		})
	}

	firstName := strings.TrimSpace(request.FirstName)
	if firstName == "" || len([]rune(firstName)) > maximumNameLength {
		fields = append(fields, buhari.FieldError{
			Path:    "first_name",
			Code:    buhari.CodeInvalidFirstName,
			Message: "First name must be between 1 and 100 characters.",
		})
	}

	lastName := strings.TrimSpace(request.LastName)
	if lastName == "" || len([]rune(lastName)) > maximumNameLength {
		fields = append(fields, buhari.FieldError{
			Path:    "last_name",
			Code:    buhari.CodeInvalidLastName,
			Message: "Last name must be between 1 and 100 characters.",
		})
	}

	if len(fields) > 0 {
		return buhari.Validation(fields...)
	}
	return nil
}

func (request LoginRequest) Validate() error {
	var fields []buhari.FieldError
	email := NormalizeEmail(request.Email)
	address, err := mail.ParseAddress(email)
	if email == "" || err != nil || address.Address != email {
		fields = append(fields, buhari.FieldError{
			Path:    "email",
			Code:    buhari.CodeInvalidEmail,
			Message: "Enter a valid email address.",
		})
	}
	if strings.TrimSpace(request.Password) == "" {
		fields = append(fields, buhari.FieldError{
			Path:    "password",
			Code:    buhari.CodeInvalidPassword,
			Message: "Password is required.",
		})
	}

	if len(fields) > 0 {
		return buhari.Validation(fields...)
	}
	return nil
}

// NewUser converts a validated create request and an already-computed Argon
// hash into the persisted user model. The plaintext password is never copied.
func NewUser(request CreateUserRequest, passwordHash string, now time.Time) (User, error) {
	if err := request.Validate(); err != nil {
		return User{}, err
	}
	if strings.TrimSpace(passwordHash) == "" {
		return User{}, ErrPasswordHashRequired
	}

	user := User{
		ID:           lid.NewUser(),
		Email:        NormalizeEmail(request.Email),
		PasswordHash: passwordHash,
		FirstName:    strings.TrimSpace(request.FirstName),
		LastName:     strings.TrimSpace(request.LastName),
		CreatedAt:    now.UTC(),
		UpdatedAt:    now.UTC(),
	}

	if err := user.Validate(); err != nil {
		return User{}, err
	}
	return user, nil
}

// Validate checks invariants on a constructed or database-loaded user.
func (u User) Validate() error {
	if err := lid.Validate(u.ID, lid.User); err != nil {
		return ErrUserIDInvalid
	}
	if u.Email == "" {
		return ErrEmailRequired
	}
	address, err := mail.ParseAddress(u.Email)
	if err != nil || address.Address != u.Email || NormalizeEmail(u.Email) != u.Email {
		return ErrEmailInvalid
	}
	if strings.TrimSpace(u.FirstName) == "" {
		return ErrFirstNameRequired
	}
	if strings.TrimSpace(u.LastName) == "" {
		return ErrLastNameRequired
	}
	if u.PasswordHash == "" {
		return ErrPasswordHashRequired
	}
	return nil
}
