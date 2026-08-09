//go:build integration

package mercury

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/calculations"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const documentValidationFailureCode = 121

func TestSchemaValidatorsAcceptModelsAndRejectMalformedDirectWrites(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database := newMongoTestDatabase(t)
	core, err := NewCore(database)
	if err != nil {
		t.Fatal(err)
	}
	if err := core.EnsureSchemaValidators(ctx); err != nil {
		t.Fatalf("EnsureSchemaValidators() first call: %v", err)
	}
	if err := core.EnsureSchemaValidators(ctx); err != nil {
		t.Fatalf("EnsureSchemaValidators() idempotent call: %v", err)
	}
	if err := core.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	user, err := model.NewUser(model.CreateUserRequest{
		Email: "schema@example.com", Password: "password123",
		FirstName: "Schema", LastName: "Reviewer",
	}, "argon2id-schema-test-hash", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := core.Users.Create(ctx, user); err != nil {
		t.Fatalf("valid user rejected: %v", err)
	}

	document := repositoryTestDocument(t)
	document.OwnerID = user.ID
	if err := core.Documents.Create(ctx, document); err != nil {
		t.Fatalf("valid document rejected: %v", err)
	}
	if _, err := core.DocumentReferences.Allocate(ctx, user.ID, now.Year(), now); err != nil {
		t.Fatalf("valid document counter rejected: %v", err)
	}

	audit, err := model.NewAuditEvent(user.ID, user.ID, lid.NewRequest(), document, model.AuditDocumentCreated, model.AuditMetadata{ChangedFields: []string{"title", "lineItems", "totals"}, CalculationPolicyVersion: document.CalculationPolicyVersion}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := core.Audits.Append(ctx, audit); err != nil {
		t.Fatalf("valid audit event rejected: %v", err)
	}

	sessionID := lid.NewSession()
	session := model.RefreshSession{
		ID: sessionID, UserID: user.ID, TokenHash: strings.Repeat("a", 64), FamilyID: sessionID,
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	if err := core.RefreshSessions.Create(ctx, session); err != nil {
		t.Fatalf("valid refresh session rejected: %v", err)
	}

	tests := []struct {
		name       string
		collection string
		document   any
	}{
		{name: "user missing required fields", collection: model.UserCollection, document: bson.D{{Key: "_id", Value: lid.NewUser()}, {Key: "email", Value: "malformed@example.com"}}},
		{name: "document rejects floating point money", collection: model.DocumentCollection, document: malformedDocument(document, "totals.grandTotalMinor", 12.5)},
		{name: "document rejects unknown stored fields", collection: model.DocumentCollection, document: documentWithUnknownField(document)},
		{name: "finalized document requires lines", collection: model.DocumentCollection, document: malformedFinalizedDocument(document, now)},
		{name: "counter rejects invalid sequence", collection: model.DocumentCounterCollection, document: bson.D{{Key: "ownerId", Value: user.ID}, {Key: "year", Value: 2026}, {Key: "sequence", Value: 0}, {Key: "updatedAt", Value: now}}},
		{name: "audit rejects unknown action", collection: model.AuditCollection, document: malformedAudit(audit)},
		{name: "session rejects unhashed token", collection: model.RefreshSessionCollection, document: bson.D{{Key: "_id", Value: lid.NewSession()}, {Key: "userId", Value: user.ID}, {Key: "tokenHash", Value: "plaintext"}, {Key: "familyId", Value: sessionID}, {Key: "createdAt", Value: now}, {Key: "expiresAt", Value: now.Add(time.Hour)}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := database.Collection(test.collection).InsertOne(ctx, test.document)
			if !hasMongoErrorCode(err, documentValidationFailureCode) {
				t.Fatalf("InsertOne() error = %v, want MongoDB validation code %d", err, documentValidationFailureCode)
			}
		})
	}
}

func hasMongoErrorCode(err error, code int) bool {
	if err == nil {
		return false
	}
	var serverError mongo.ServerError
	return errors.As(err, &serverError) && serverError.HasErrorCode(code)
}

func malformedDocument(document model.Document, path string, value any) bson.D {
	raw, _ := bson.Marshal(document)
	var result bson.D
	_ = bson.Unmarshal(raw, &result)
	if path == "totals.grandTotalMinor" {
		for index := range result {
			if result[index].Key != "totals" {
				continue
			}
			totals := result[index].Value.(bson.D)
			for totalIndex := range totals {
				if totals[totalIndex].Key == "grandTotalMinor" {
					totals[totalIndex].Value = value
				}
			}
			result[index].Value = totals
		}
	}
	for index := range result {
		if result[index].Key == "_id" {
			result[index].Value = lid.NewDocument()
		}
	}
	return result
}

func documentWithUnknownField(document model.Document) bson.D {
	document.ID = lid.NewDocument()
	raw, _ := bson.Marshal(document)
	var result bson.D
	_ = bson.Unmarshal(raw, &result)
	return append(result, bson.E{Key: "clientCalculatedTotal", Value: int64(1)})
}

func malformedFinalizedDocument(document model.Document, now time.Time) bson.D {
	document.ID = lid.NewDocument()
	document.Status = model.DocumentFinalized
	document.FinalizedAt = &now
	document.LineItems = make([]calculations.LineResult, 0)
	raw, _ := bson.Marshal(document)
	var result bson.D
	_ = bson.Unmarshal(raw, &result)
	return result
}

func malformedAudit(event model.AuditEvent) bson.D {
	event.ID = lid.NewAudit()
	event.Action = model.AuditAction("document.deleted")
	raw, _ := bson.Marshal(event)
	var result bson.D
	_ = bson.Unmarshal(raw, &result)
	return result
}
