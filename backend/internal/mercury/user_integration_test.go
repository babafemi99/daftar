//go:build integration

package mercury

import (
	"context"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
)

func TestUserRepositoryIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	repository := NewUserRepository(newMongoTestDatabase(t))
	if err := repository.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}

	user, err := model.NewUser(model.CreateUserRequest{
		Email: " Finance@Example.com ", Password: "password123",
		FirstName: "Finance", LastName: "User",
	}, "argon2id-test-hash", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(ctx, user); err != nil {
		t.Fatal(err)
	}

	byID, err := repository.FindByID(ctx, user.ID)
	if err != nil || byID.Email != "finance@example.com" {
		t.Fatalf("FindByID() email=%q error=%v", byID.Email, err)
	}
	byEmail, err := repository.FindByEmail(ctx, " FINANCE@EXAMPLE.COM ")
	if err != nil || byEmail.ID != user.ID {
		t.Fatalf("FindByEmail() id=%q error=%v", byEmail.ID, err)
	}

	duplicate := user
	duplicate.ID = lid.NewUser()
	if err := repository.Create(ctx, duplicate); !repositoryHasCode(err, buhari.CodeEmailAlreadyRegistered) {
		t.Fatalf("duplicate email error = %v", err)
	}
	if _, err := repository.FindByEmail(ctx, "missing@example.com"); !repositoryHasCode(err, buhari.CodeNotFound) {
		t.Fatalf("missing user error = %v", err)
	}
}
