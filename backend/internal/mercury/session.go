package mercury

import (
	"context"
	"errors"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type RefreshSessions interface {
	Create(context.Context, model.RefreshSession) error
	Rotate(context.Context, string, model.RefreshSession, time.Time) (model.RefreshSession, error)
	Revoke(context.Context, string, time.Time) error
}

type RefreshSessionRepository struct{ collection *mongo.Collection }

func NewRefreshSessionRepository(database *mongo.Database) *RefreshSessionRepository {
	return &RefreshSessionRepository{collection: database.Collection(model.RefreshSessionCollection)}
}

func (repository *RefreshSessionRepository) EnsureIndexes(ctx context.Context) error {
	_, err := repository.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetName("refresh_token_hash_unique").SetUnique(true)},
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetName("refresh_expiry_ttl").SetExpireAfterSeconds(0)},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "familyId", Value: 1}}, Options: options.Index().SetName("refresh_user_family")},
	})
	if err != nil {
		return buhari.Wrap(buhari.CodeInternalError, "Unable to initialize session storage.", err)
	}
	return nil
}

func (repository *RefreshSessionRepository) Create(ctx context.Context, session model.RefreshSession) error {
	if _, err := repository.collection.InsertOne(ctx, session); err != nil {
		return buhari.Wrap(buhari.CodeInternalError, "Unable to create refresh session.", err)
	}
	return nil
}

func (repository *RefreshSessionRepository) Rotate(ctx context.Context, tokenHash string, replacement model.RefreshSession, now time.Time) (model.RefreshSession, error) {
	var current model.RefreshSession
	err := repository.collection.FindOneAndUpdate(ctx,
		bson.D{{Key: "tokenHash", Value: tokenHash}, {Key: "revokedAt", Value: nil}, {Key: "expiresAt", Value: bson.D{{Key: "$gt", Value: now}}}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "revokedAt", Value: now}, {Key: "replacedById", Value: replacement.ID}}}},
		options.FindOneAndUpdate().SetReturnDocument(options.Before)).Decode(&current)
	if errors.Is(err, mongo.ErrNoDocuments) {
		lookupErr := repository.collection.FindOne(ctx, bson.D{{Key: "tokenHash", Value: tokenHash}}).Decode(&current)
		if lookupErr == nil && current.RevokedAt != nil {
			if _, revokeErr := repository.collection.UpdateMany(ctx, bson.D{{Key: "userId", Value: current.UserID}, {Key: "familyId", Value: current.FamilyID}, {Key: "revokedAt", Value: nil}}, bson.D{{Key: "$set", Value: bson.D{{Key: "revokedAt", Value: now}}}}); revokeErr != nil {
				return current, buhari.Wrap(buhari.CodeInternalError, "Unable to revoke a reused session family.", revokeErr)
			}
			return current, nil
		}
		return current, buhari.New(buhari.CodeUnauthorized, "The refresh session is invalid or expired.")
	}
	if err != nil {
		return current, buhari.Wrap(buhari.CodeInternalError, "Unable to rotate refresh session.", err)
	}
	replacement.UserID, replacement.FamilyID = current.UserID, current.FamilyID
	if err := repository.Create(ctx, replacement); err != nil {
		return current, err
	}
	return current, nil
}

func (repository *RefreshSessionRepository) Revoke(ctx context.Context, tokenHash string, now time.Time) error {
	_, err := repository.collection.UpdateOne(ctx, bson.D{{Key: "tokenHash", Value: tokenHash}, {Key: "revokedAt", Value: nil}}, bson.D{{Key: "$set", Value: bson.D{{Key: "revokedAt", Value: now}}}})
	if err != nil {
		return buhari.Wrap(buhari.CodeInternalError, "Unable to revoke refresh session.", err)
	}
	return nil
}
