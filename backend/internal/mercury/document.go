package mercury

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	documentReferenceIndex  = "documents_owner_reference_unique"
	documentStatusDateIndex = "documents_owner_status_issue_date_id"
	documentCurrencyIndex   = "documents_owner_currency_issue_date"
	documentArchiveIndex    = "documents_owner_archived_updated"
	documentSearchIndex     = "documents_owner_search"
	defaultDocumentLimit    = 20
	MaximumDocumentLimit    = 100
)

const DocumentSortIssueDateDesc = "issue_date_desc"

type DocumentListFilter struct {
	Status    *model.DocumentStatus
	Currency  *model.Currency
	IssueFrom *time.Time
	IssueTo   *time.Time
	Archived  *bool
	Limit     int64
	Page      int64
	Sort      string
	Search    string
}

type DocumentPage struct {
	Documents  []model.Document
	Page       int64
	PageSize   int64
	TotalItems int64
	TotalPages int64
}

type Documents interface {
	Create(ctx context.Context, document model.Document) error
	FindByID(ctx context.Context, ownerID, documentID string) (model.Document, error)
	FindByReference(ctx context.Context, ownerID, reference string) (model.Document, error)
	List(ctx context.Context, ownerID string, filter DocumentListFilter) (DocumentPage, error)
	ReplaceDraft(ctx context.Context, document model.Document, expectedVersion int64) error
	Finalize(ctx context.Context, document model.Document, expectedVersion int64) error
	Archive(ctx context.Context, document model.Document, expectedVersion int64) error
	Restore(ctx context.Context, document model.Document, expectedVersion int64) error
}

type DocumentRepository struct {
	collection *mongo.Collection
}

var _ Documents = (*DocumentRepository)(nil)

func NewDocumentRepository(database *mongo.Database) *DocumentRepository {
	return &DocumentRepository{collection: database.Collection(model.DocumentCollection)}
}

func (r *DocumentRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "ownerId", Value: 1}, {Key: "reference", Value: 1}},
			Options: options.Index().SetName(documentReferenceIndex).SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "ownerId", Value: 1}, {Key: "status", Value: 1}, {Key: "issueDate", Value: -1}, {Key: "_id", Value: -1}},
			Options: options.Index().SetName(documentStatusDateIndex),
		},
		{
			Keys:    bson.D{{Key: "ownerId", Value: 1}, {Key: "currency", Value: 1}, {Key: "issueDate", Value: -1}},
			Options: options.Index().SetName(documentCurrencyIndex),
		},
		{
			Keys:    bson.D{{Key: "ownerId", Value: 1}, {Key: "archivedAt", Value: 1}, {Key: "updatedAt", Value: -1}},
			Options: options.Index().SetName(documentArchiveIndex),
		},
		{Keys: bson.D{{Key: "ownerId", Value: 1}, {Key: "reference", Value: "text"}, {Key: "title", Value: "text"}, {Key: "customer", Value: "text"}}, Options: options.Index().SetName(documentSearchIndex).SetWeights(bson.D{{Key: "reference", Value: 10}, {Key: "title", Value: 5}, {Key: "customer", Value: 3}})},
	})
	if err != nil {
		return buhari.Wrap(buhari.CodeInternalError, "Unable to initialize document storage.", err)
	}
	return nil
}

func (r *DocumentRepository) Create(ctx context.Context, document model.Document) error {
	if err := document.Validate(); err != nil {
		return err
	}
	if _, err := r.collection.InsertOne(ctx, document); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return buhari.New(buhari.CodeConflict, "A document with this reference already exists.")
		}
		return buhari.Wrap(buhari.CodeInternalError, "Unable to save document.", err)
	}
	return nil
}

func (r *DocumentRepository) FindByID(ctx context.Context, ownerID, documentID string) (model.Document, error) {
	return r.findOne(ctx, bson.D{{Key: "_id", Value: documentID}, {Key: "ownerId", Value: ownerID}})
}

func (r *DocumentRepository) FindByReference(ctx context.Context, ownerID, reference string) (model.Document, error) {
	return r.findOne(ctx, bson.D{{Key: "ownerId", Value: ownerID}, {Key: "reference", Value: reference}})
}

func (r *DocumentRepository) List(ctx context.Context, ownerID string, filter DocumentListFilter) (DocumentPage, error) {
	query := bson.D{{Key: "ownerId", Value: ownerID}}
	if filter.Search != "" {
		query = append(query, bson.E{Key: "$text", Value: bson.D{{Key: "$search", Value: filter.Search}}})
	}
	if filter.Status != nil {
		query = append(query, bson.E{Key: "status", Value: *filter.Status})
	}
	if filter.Currency != nil {
		query = append(query, bson.E{Key: "currency", Value: *filter.Currency})
	}
	if filter.IssueFrom != nil || filter.IssueTo != nil {
		rangeFilter := bson.D{}
		if filter.IssueFrom != nil {
			rangeFilter = append(rangeFilter, bson.E{Key: "$gte", Value: filter.IssueFrom.UTC()})
		}
		if filter.IssueTo != nil {
			rangeFilter = append(rangeFilter, bson.E{Key: "$lte", Value: filter.IssueTo.UTC()})
		}
		query = append(query, bson.E{Key: "issueDate", Value: rangeFilter})
	}
	if filter.Archived == nil || !*filter.Archived {
		query = append(query, bson.E{Key: "archivedAt", Value: nil})
	} else {
		query = append(query, bson.E{Key: "archivedAt", Value: bson.D{{Key: "$ne", Value: nil}}})
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = defaultDocumentLimit
	}
	if limit > MaximumDocumentLimit {
		limit = MaximumDocumentLimit
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return DocumentPage{}, buhari.Wrap(buhari.CodeInternalError, "Unable to count documents.", err)
	}
	cursor, err := r.collection.Find(ctx, query, options.Find().
		SetSort(bson.D{{Key: "issueDate", Value: -1}, {Key: "_id", Value: -1}}).
		SetSkip((page-1)*limit).
		SetLimit(limit))
	if err != nil {
		return DocumentPage{}, buhari.Wrap(buhari.CodeInternalError, "Unable to list documents.", err)
	}
	defer cursor.Close(ctx)

	documents := make([]model.Document, 0)
	for cursor.Next(ctx) {
		var document model.Document
		if err := cursor.Decode(&document); err != nil {
			return DocumentPage{}, buhari.Wrap(buhari.CodeInternalError, "Unable to decode a stored document.", err)
		}
		if err := document.Validate(); err != nil {
			return DocumentPage{}, buhari.Wrap(buhari.CodeInternalError, "Stored document failed integrity validation.", err)
		}
		documents = append(documents, document)
	}
	if err := cursor.Err(); err != nil {
		return DocumentPage{}, buhari.Wrap(buhari.CodeInternalError, "Unable to list documents.", err)
	}
	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}
	return DocumentPage{Documents: documents, Page: page, PageSize: limit, TotalItems: total, TotalPages: totalPages}, nil
}

func (r *DocumentRepository) ReplaceDraft(ctx context.Context, document model.Document, expectedVersion int64) error {
	return r.replace(ctx, document, expectedVersion, mutationReplace)
}

func (r *DocumentRepository) Finalize(ctx context.Context, document model.Document, expectedVersion int64) error {
	return r.replace(ctx, document, expectedVersion, mutationFinalize)
}

func (r *DocumentRepository) Archive(ctx context.Context, document model.Document, expectedVersion int64) error {
	return r.replace(ctx, document, expectedVersion, mutationArchive)
}

func (r *DocumentRepository) Restore(ctx context.Context, document model.Document, expectedVersion int64) error {
	return r.replace(ctx, document, expectedVersion, mutationRestore)
}

type documentMutation uint8

const (
	mutationReplace documentMutation = iota
	mutationFinalize
	mutationArchive
	mutationRestore
)

func (r *DocumentRepository) replace(ctx context.Context, document model.Document, expectedVersion int64, mutation documentMutation) error {
	if expectedVersion < 1 || document.Version != expectedVersion+1 {
		return buhari.New(buhari.CodeDocumentVersionConflict, "Document version does not match the expected version.")
	}
	if err := document.Validate(); err != nil {
		return err
	}
	if err := validateMutationTarget(document, mutation); err != nil {
		return err
	}

	filter := bson.D{
		{Key: "_id", Value: document.ID},
		{Key: "ownerId", Value: document.OwnerID},
		{Key: "status", Value: model.DocumentDraft},
		{Key: "version", Value: expectedVersion},
	}
	if mutation == mutationRestore {
		filter = append(filter, bson.E{Key: "archivedAt", Value: bson.D{{Key: "$ne", Value: nil}}})
	} else {
		filter = append(filter, bson.E{Key: "archivedAt", Value: nil})
	}

	result, err := r.collection.ReplaceOne(ctx, filter, document)
	if err != nil {
		return buhari.Wrap(buhari.CodeInternalError, "Unable to update document.", err)
	}
	if result.MatchedCount == 1 {
		return nil
	}
	return r.diagnoseMutation(ctx, document.OwnerID, document.ID, expectedVersion, mutation)
}

func (r *DocumentRepository) diagnoseMutation(ctx context.Context, ownerID, documentID string, expectedVersion int64, mutation documentMutation) error {
	current, err := r.FindByID(ctx, ownerID, documentID)
	if err != nil {
		return err
	}
	return classifyMutationConflict(current, expectedVersion, mutation)
}

func validateMutationTarget(document model.Document, mutation documentMutation) error {
	valid := false
	switch mutation {
	case mutationReplace:
		valid = document.Status == model.DocumentDraft && document.ArchivedAt == nil && document.FinalizedAt == nil
	case mutationFinalize:
		valid = document.Status == model.DocumentFinalized && document.FinalizedAt != nil && document.ArchivedAt == nil
	case mutationArchive:
		valid = document.Status == model.DocumentDraft && document.ArchivedAt != nil && document.FinalizedAt == nil
	case mutationRestore:
		valid = document.Status == model.DocumentDraft && document.ArchivedAt == nil && document.FinalizedAt == nil
	}
	if !valid {
		return buhari.New(buhari.CodeConflict, "Document does not contain the state required by this operation.")
	}
	return nil
}

func classifyMutationConflict(current model.Document, expectedVersion int64, mutation documentMutation) error {
	if current.Status == model.DocumentFinalized {
		if mutation == mutationFinalize {
			return buhari.New(buhari.CodeDocumentAlreadyFinalized, "Document is already finalized.")
		}
		return buhari.New(buhari.CodeDocumentFinalized, "Finalized documents are immutable.")
	}
	if mutation != mutationRestore && current.ArchivedAt != nil {
		return buhari.New(buhari.CodeDocumentArchived, "Archived documents must be restored before editing.")
	}
	if mutation == mutationRestore && current.ArchivedAt == nil {
		return buhari.New(buhari.CodeDocumentNotArchived, "Document is not archived.")
	}
	if current.Version != expectedVersion {
		return buhari.New(buhari.CodeDocumentVersionConflict, "Document was changed by another request.")
	}
	return buhari.New(buhari.CodeConflict, "Document state does not allow this operation.")
}

func (r *DocumentRepository) findOne(ctx context.Context, filter any) (model.Document, error) {
	var document model.Document
	if err := r.collection.FindOne(ctx, filter).Decode(&document); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Document{}, buhari.New(buhari.CodeNotFound, "Document not found.")
		}
		return model.Document{}, buhari.Wrap(buhari.CodeInternalError, "Unable to retrieve document.", err)
	}
	if err := document.Validate(); err != nil {
		return model.Document{}, buhari.Wrap(buhari.CodeInternalError, "Stored document failed integrity validation.", fmt.Errorf("document %s: %w", document.ID, err))
	}
	return document, nil
}
