package mercury

import (
	"context"
	"fmt"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const documentCounterOwnerYearIndex = "document_counters_owner_year_unique"

type DocumentReferences interface {
	Allocate(ctx context.Context, ownerID string, year int, now time.Time) (string, error)
}

type DocumentReferenceRepository struct {
	collection *mongo.Collection
}

type documentCounter struct {
	OwnerID   string    `bson:"ownerId"`
	Year      int       `bson:"year"`
	Sequence  int64     `bson:"sequence"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

var _ DocumentReferences = (*DocumentReferenceRepository)(nil)

func NewDocumentReferenceRepository(database *mongo.Database) *DocumentReferenceRepository {
	return &DocumentReferenceRepository{collection: database.Collection(model.DocumentCounterCollection)}
}

func (r *DocumentReferenceRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "ownerId", Value: 1}, {Key: "year", Value: 1}},
		Options: options.Index().SetName(documentCounterOwnerYearIndex).SetUnique(true),
	})
	if err != nil {
		return buhari.Wrap(buhari.CodeInternalError, "Unable to initialize document references.", err)
	}
	return nil
}

// Allocate atomically reserves a reference. Gaps are allowed and references
// are never reused, even when a later document insert fails.
func (r *DocumentReferenceRepository) Allocate(ctx context.Context, ownerID string, year int, now time.Time) (string, error) {
	if year < 1 || year > 9999 {
		return "", buhari.New(buhari.CodeValidationFailed, "Document reference year is invalid.")
	}
	filter := bson.D{{Key: "ownerId", Value: ownerID}, {Key: "year", Value: year}}
	update := bson.D{
		{Key: "$inc", Value: bson.D{{Key: "sequence", Value: 1}}},
		{Key: "$set", Value: bson.D{{Key: "updatedAt", Value: now.UTC()}}},
		{Key: "$setOnInsert", Value: bson.D{{Key: "ownerId", Value: ownerID}, {Key: "year", Value: year}}},
	}

	var counter documentCounter
	err := r.collection.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&counter)
	if err != nil {
		return "", buhari.Wrap(buhari.CodeInternalError, "Unable to allocate document reference.", err)
	}
	if counter.Sequence < 1 || counter.Sequence > 999_999 {
		return "", buhari.New(buhari.CodeConflict, "The annual document reference sequence is exhausted.")
	}
	return fmt.Sprintf("DOC-%04d-%06d", year, counter.Sequence), nil
}
