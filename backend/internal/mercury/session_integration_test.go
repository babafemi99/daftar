//go:build integration

package mercury

import (
	"context"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
)

func TestRefreshSessionRotationIsOneTimeIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repository := NewRefreshSessionRepository(newMongoTestDatabase(t))
	if err := repository.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	currentID := lid.NewSession()
	current := model.RefreshSession{ID: currentID, UserID: lid.NewUser(), TokenHash: "old-hash", FamilyID: currentID, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := repository.Create(ctx, current); err != nil {
		t.Fatal(err)
	}
	replacement := model.RefreshSession{ID: lid.NewSession(), TokenHash: "new-hash", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}
	rotated, err := repository.Rotate(ctx, current.TokenHash, replacement, now)
	if err != nil || rotated.ID != current.ID {
		t.Fatalf("rotated=%+v err=%v", rotated, err)
	}
	reused, err := repository.Rotate(ctx, current.TokenHash, model.RefreshSession{ID: lid.NewSession(), TokenHash: "replay-hash", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}, now)
	if err != nil || reused.RevokedAt == nil {
		t.Fatalf("replayed token session=%+v error=%v", reused, err)
	}
}
