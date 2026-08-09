package mercury

import (
	"context"
	"errors"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const userEmailIndex = "users_email_unique"

type Users interface {
	Create(ctx context.Context, user model.User) error
	FindByID(ctx context.Context, id string) (model.User, error)
	FindByEmail(ctx context.Context, email string) (model.User, error)
}

type UserRepository struct {
	collection *mongo.Collection
}

var _ Users = (*UserRepository)(nil)

func NewUserRepository(database *mongo.Database) *UserRepository {
	return &UserRepository{
		collection: database.Collection(model.UserCollection),
	}
}

// EnsureIndexes creates the constraints required by the user repository. It
// is safe to call during application startup on every instance.
func (r *UserRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "email", Value: 1}},
		Options: options.Index().
			SetName(userEmailIndex).
			SetUnique(true),
	})
	if err != nil {
		return buhari.Wrap(buhari.CodeInternalError, "Unable to initialize user storage.", err)
	}

	return nil
}

func (r *UserRepository) Create(ctx context.Context, user model.User) error {
	if _, err := r.collection.InsertOne(ctx, user); err != nil {
		return mapUserWriteError(err)
	}

	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (model.User, error) {
	return r.findOne(ctx, bson.D{{Key: "_id", Value: id}})
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	return r.findOne(ctx, bson.D{{Key: "email", Value: model.NormalizeEmail(email)}})
}

func (r *UserRepository) findOne(ctx context.Context, filter any) (model.User, error) {
	var user model.User
	if err := r.collection.FindOne(ctx, filter).Decode(&user); err != nil {
		return model.User{}, mapUserReadError(err)
	}

	return user, nil
}

func mapUserReadError(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return buhari.New(buhari.CodeNotFound, "User not found.")
	}

	return buhari.Wrap(buhari.CodeInternalError, "Unable to retrieve user.", err)
}

func mapUserWriteError(err error) error {
	if mongo.IsDuplicateKeyError(err) {
		return buhari.New(buhari.CodeEmailAlreadyRegistered, "An account with this email already exists.")
	}

	return buhari.Wrap(buhari.CodeInternalError, "Unable to save user.", err)
}
