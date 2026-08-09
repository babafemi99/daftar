package mercury

import (
	"context"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	auditDocumentIndex = "audit_owner_document_occurred_id"
	auditOwnerIndex    = "audit_owner_occurred_id"
	DefaultAuditLimit  = int64(25)
	MaximumAuditLimit  = int64(100)
)

type Audits interface {
	Append(context.Context, model.AuditEvent) error
	ListDocument(context.Context, string, string, int64) ([]model.AuditEvent, error)
}

type AuditRepository struct{ collection *mongo.Collection }

var _ Audits = (*AuditRepository)(nil)

func NewAuditRepository(database *mongo.Database) *AuditRepository {
	return &AuditRepository{collection: database.Collection(model.AuditCollection)}
}

func (repository *AuditRepository) EnsureIndexes(ctx context.Context) error {
	_, err := repository.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "ownerId", Value: 1}, {Key: "documentId", Value: 1}, {Key: "occurredAt", Value: -1}, {Key: "_id", Value: -1}}, Options: options.Index().SetName(auditDocumentIndex)},
		{Keys: bson.D{{Key: "ownerId", Value: 1}, {Key: "occurredAt", Value: -1}, {Key: "_id", Value: -1}}, Options: options.Index().SetName(auditOwnerIndex)},
	})
	if err != nil {
		return buhari.Wrap(buhari.CodeInternalError, "Unable to initialize audit storage.", err)
	}
	return nil
}

func (repository *AuditRepository) Append(ctx context.Context, event model.AuditEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}
	if _, err := repository.collection.InsertOne(ctx, event); err != nil {
		return buhari.Wrap(buhari.CodeInternalError, "Unable to append audit event.", err)
	}
	return nil
}

func (repository *AuditRepository) ListDocument(ctx context.Context, ownerID, documentID string, limit int64) ([]model.AuditEvent, error) {
	if limit <= 0 {
		limit = DefaultAuditLimit
	}
	if limit > MaximumAuditLimit {
		limit = MaximumAuditLimit
	}
	cursor, err := repository.collection.Find(ctx, bson.D{{Key: "ownerId", Value: ownerID}, {Key: "documentId", Value: documentID}}, options.Find().SetSort(bson.D{{Key: "occurredAt", Value: -1}, {Key: "_id", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, buhari.Wrap(buhari.CodeInternalError, "Unable to list audit events.", err)
	}
	defer cursor.Close(ctx)
	events := make([]model.AuditEvent, 0)
	for cursor.Next(ctx) {
		var event model.AuditEvent
		if err := cursor.Decode(&event); err != nil {
			return nil, buhari.Wrap(buhari.CodeInternalError, "Unable to decode an audit event.", err)
		}
		if err := event.Validate(); err != nil {
			return nil, buhari.Wrap(buhari.CodeInternalError, "Stored audit event failed integrity validation.", err)
		}
		events = append(events, event)
	}
	if err := cursor.Err(); err != nil {
		return nil, buhari.Wrap(buhari.CodeInternalError, "Unable to list audit events.", err)
	}
	return events, nil
}
