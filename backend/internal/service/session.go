package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/mercury"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
)

var ErrRefreshSessionRepositoryRequired = errors.New("service: refresh session repository is required")

type SessionService struct {
	sessions mercury.RefreshSessions
	users    mercury.Users
	tx       transactionRunner
	ttl      time.Duration
}

func NewSessionService(sessions mercury.RefreshSessions, users mercury.Users, tx transactionRunner, ttl time.Duration) (*SessionService, error) {
	if sessions == nil {
		return nil, ErrRefreshSessionRepositoryRequired
	}
	if users == nil {
		return nil, ErrUserRepositoryRequired
	}
	if tx == nil {
		return nil, ErrTransactionRunnerRequired
	}
	if ttl <= 0 {
		return nil, errors.New("service: refresh session TTL is required")
	}
	return &SessionService{sessions: sessions, users: users, tx: tx, ttl: ttl}, nil
}

func (service *SessionService) Issue(ctx context.Context, userID string) (string, time.Time, error) {
	raw, hash, err := newRefreshToken()
	if err != nil {
		return "", time.Time{}, buhari.Wrap(buhari.CodeInternalError, "Unable to create refresh token.", err)
	}
	now, id := time.Now().UTC(), lid.NewSession()
	session := model.RefreshSession{ID: id, UserID: userID, TokenHash: hash, FamilyID: id, CreatedAt: now, ExpiresAt: now.Add(service.ttl)}
	if err := service.sessions.Create(ctx, session); err != nil {
		return "", time.Time{}, err
	}
	return raw, session.ExpiresAt, nil
}

func (service *SessionService) Rotate(ctx context.Context, raw string) (*model.User, string, time.Time, error) {
	if raw == "" {
		return nil, "", time.Time{}, buhari.New(buhari.CodeUnauthorized, "A refresh session is required.")
	}
	newRaw, hash, err := newRefreshToken()
	if err != nil {
		return nil, "", time.Time{}, buhari.Wrap(buhari.CodeInternalError, "Unable to rotate refresh token.", err)
	}
	now := time.Now().UTC()
	replacement := model.RefreshSession{ID: lid.NewSession(), TokenHash: hash, CreatedAt: now, ExpiresAt: now.Add(service.ttl)}
	var current model.RefreshSession
	err = service.tx.RunInTx(ctx, func(txCtx context.Context) error {
		var rotateErr error
		current, rotateErr = service.sessions.Rotate(txCtx, tokenHash(raw), replacement, now)
		return rotateErr
	})
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if current.RevokedAt != nil {
		return nil, "", time.Time{}, buhari.New(buhari.CodeUnauthorized, "Refresh-token reuse was detected and the session was revoked.")
	}
	user, err := service.users.FindByID(ctx, current.UserID)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	return &user, newRaw, replacement.ExpiresAt, nil
}

func (service *SessionService) Revoke(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return service.sessions.Revoke(ctx, tokenHash(raw), time.Now().UTC())
}

func newRefreshToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	raw := base64.RawURLEncoding.EncodeToString(bytes)
	return raw, tokenHash(raw), nil
}

func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
