package model

import "strings"

type UserID string

type Executor struct {
	ID    UserID `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
	Role  string `json:"role,omitempty"`
}

type CreateUserRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// NormalizeEmail returns the canonical value used for user storage, unique
// indexes, login, and account lookup.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
