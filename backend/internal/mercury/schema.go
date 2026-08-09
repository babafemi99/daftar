package mercury

import (
	"context"
	"errors"
	"fmt"

	"github.com/babafemi99/daftar/backend/internal/calculations"
	"github.com/babafemi99/daftar/backend/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	validationLevel  = "strict"
	validationAction = "error"
)

type collectionSchema struct {
	name      string
	validator bson.D
}

// EnsureSchemaValidators installs defense-in-depth constraints on every
// application-owned collection. Existing collections are updated with collMod;
// missing collections are created with the same validator. The operation is
// safe to repeat on every startup.
func (c *Core) EnsureSchemaValidators(ctx context.Context) error {
	if c == nil || c.database == nil {
		return ErrDatabaseRequired
	}
	for _, definition := range daftarCollectionSchemas() {
		if err := ensureCollectionSchema(ctx, c.database, definition); err != nil {
			return fmt.Errorf("mercury: ensure %s schema validator: %w", definition.name, err)
		}
	}
	return nil
}

func ensureCollectionSchema(ctx context.Context, database *mongo.Database, definition collectionSchema) error {
	names, err := database.ListCollectionNames(ctx, bson.D{{Key: "name", Value: definition.name}})
	if err != nil {
		return err
	}
	if len(names) == 0 {
		err = database.RunCommand(ctx, bson.D{
			{Key: "create", Value: definition.name},
			{Key: "validator", Value: definition.validator},
			{Key: "validationLevel", Value: validationLevel},
			{Key: "validationAction", Value: validationAction},
		}).Err()
		if err == nil {
			return nil
		}
		var commandError mongo.CommandError
		if !errors.As(err, &commandError) || commandError.Code != 48 { // NamespaceExists
			return err
		}
	}

	return database.RunCommand(ctx, bson.D{
		{Key: "collMod", Value: definition.name},
		{Key: "validator", Value: definition.validator},
		{Key: "validationLevel", Value: validationLevel},
		{Key: "validationAction", Value: validationAction},
	}).Err()
}

func daftarCollectionSchemas() []collectionSchema {
	return []collectionSchema{
		{name: model.UserCollection, validator: bson.D{{Key: "$jsonSchema", Value: userSchema()}}},
		{name: model.DocumentCollection, validator: documentValidator()},
		{name: model.DocumentCounterCollection, validator: bson.D{{Key: "$jsonSchema", Value: documentCounterSchema()}}},
		{name: model.AuditCollection, validator: bson.D{{Key: "$jsonSchema", Value: auditSchema()}}},
		{name: model.RefreshSessionCollection, validator: bson.D{{Key: "$jsonSchema", Value: refreshSessionSchema()}}},
	}
}

func objectSchema(required bson.A, properties bson.D) bson.D {
	schema := bson.D{
		{Key: "bsonType", Value: "object"},
		{Key: "additionalProperties", Value: false},
		{Key: "properties", Value: properties},
	}
	if len(required) > 0 {
		schema = append(schema, bson.E{Key: "required", Value: required})
	}
	return schema
}

func stringProperty(pattern string, minLength, maxLength int) bson.D {
	property := bson.D{{Key: "bsonType", Value: "string"}}
	if pattern != "" {
		property = append(property, bson.E{Key: "pattern", Value: pattern})
	}
	if minLength > 0 {
		property = append(property, bson.E{Key: "minLength", Value: minLength})
	}
	if maxLength > 0 {
		property = append(property, bson.E{Key: "maxLength", Value: maxLength})
	}
	return property
}

func integerProperty(minimum, maximum int64) bson.D {
	return bson.D{
		{Key: "bsonType", Value: bson.A{"int", "long"}},
		{Key: "minimum", Value: minimum},
		{Key: "maximum", Value: maximum},
	}
}

func userSchema() bson.D {
	return objectSchema(
		bson.A{"_id", "email", "passwordHash", "first_name", "last_name", "createdAt", "updatedAt"},
		bson.D{
			{Key: "_id", Value: stringProperty(`^user-[0-9A-HJKMNP-TV-Z]{26}$`, 0, 0)},
			{Key: "email", Value: stringProperty(`^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$`, 3, 320)},
			{Key: "passwordHash", Value: stringProperty("", 1, 512)},
			{Key: "first_name", Value: stringProperty("", 1, 100)},
			{Key: "last_name", Value: stringProperty("", 1, 100)},
			{Key: "createdAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
			{Key: "updatedAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
		},
	)
}

func documentValidator() bson.D {
	schema := objectSchema(
		bson.A{"_id", "ownerId", "reference", "title", "customer", "issueDate", "currency", "status", "lineItems", "totals", "taxBreakdown", "calculationPolicyVersion", "version", "finalizedAt", "archivedAt", "createdAt", "updatedAt"},
		bson.D{
			{Key: "_id", Value: stringProperty(`^doc-[0-9A-HJKMNP-TV-Z]{26}$`, 0, 0)},
			{Key: "ownerId", Value: stringProperty(`^user-[0-9A-HJKMNP-TV-Z]{26}$`, 0, 0)},
			{Key: "reference", Value: stringProperty(`^DOC-[0-9]{4}-[0-9]{6}$`, 0, 0)},
			{Key: "title", Value: stringProperty("", 1, 160)},
			{Key: "customer", Value: stringProperty("", 1, 200)},
			{Key: "issueDate", Value: bson.D{{Key: "bsonType", Value: "date"}}},
			{Key: "currency", Value: bson.D{{Key: "enum", Value: bson.A{"USD", "AED", "SAR", "NGN", "GBP", "EUR"}}}},
			{Key: "status", Value: bson.D{{Key: "enum", Value: bson.A{"draft", "finalized"}}}},
			{Key: "lineItems", Value: bson.D{{Key: "bsonType", Value: "array"}, {Key: "maxItems", Value: calculations.MaxLineItems}, {Key: "items", Value: lineItemSchema()}}},
			{Key: "totals", Value: totalsSchema()},
			{Key: "taxBreakdown", Value: bson.D{{Key: "bsonType", Value: "array"}, {Key: "maxItems", Value: calculations.MaxLineItems}, {Key: "items", Value: taxBreakdownSchema()}}},
			{Key: "calculationPolicyVersion", Value: bson.D{{Key: "enum", Value: bson.A{calculations.PolicyVersion}}}},
			{Key: "version", Value: integerProperty(1, int64(^uint64(0)>>1))},
			{Key: "finalizedAt", Value: bson.D{{Key: "bsonType", Value: bson.A{"date", "null"}}}},
			{Key: "archivedAt", Value: bson.D{{Key: "bsonType", Value: bson.A{"date", "null"}}}},
			{Key: "createdAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
			{Key: "updatedAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
		},
	)
	lifecycle := bson.D{{Key: "$or", Value: bson.A{
		bson.D{{Key: "status", Value: "draft"}, {Key: "finalizedAt", Value: nil}},
		bson.D{{Key: "status", Value: "finalized"}, {Key: "finalizedAt", Value: bson.D{{Key: "$type", Value: "date"}}}, {Key: "archivedAt", Value: nil}, {Key: "lineItems.0", Value: bson.D{{Key: "$exists", Value: true}}}},
	}}}
	return bson.D{{Key: "$and", Value: bson.A{bson.D{{Key: "$jsonSchema", Value: schema}}, lifecycle}}}
}

func lineItemSchema() bson.D {
	return objectSchema(
		bson.A{"id", "description", "quantity", "unitPriceMinor", "taxRate", "calculated"},
		bson.D{
			{Key: "id", Value: stringProperty(`^line-[0-9A-HJKMNP-TV-Z]{26}$`, 0, 0)},
			{Key: "description", Value: stringProperty("", 1, calculations.MaxDescription)},
			{Key: "quantity", Value: integerProperty(1, calculations.MaxQuantity)},
			{Key: "unitPriceMinor", Value: integerProperty(0, int64(^uint64(0)>>1))},
			{Key: "discount", Value: discountSchema()},
			{Key: "taxRate", Value: integerProperty(0, int64(calculations.RateScale))},
			{Key: "calculated", Value: calculatedLineSchema()},
		},
	)
}

func discountSchema() bson.D {
	return objectSchema(bson.A{"type", "value"}, bson.D{
		{Key: "type", Value: bson.D{{Key: "enum", Value: bson.A{"fixed", "percentage"}}}},
		{Key: "value", Value: integerProperty(0, int64(^uint64(0)>>1))},
	})
}

func calculatedLineSchema() bson.D {
	return monetaryObjectSchema(bson.A{"subtotalMinor", "discountAmountMinor", "discountedAmountMinor", "taxAmountMinor", "lineTotalMinor"})
}

func totalsSchema() bson.D {
	return monetaryObjectSchema(bson.A{"subtotalMinor", "discountMinor", "taxMinor", "grandTotalMinor"})
}

func monetaryObjectSchema(fields bson.A) bson.D {
	properties := make(bson.D, 0, len(fields))
	for _, field := range fields {
		properties = append(properties, bson.E{Key: field.(string), Value: integerProperty(0, int64(^uint64(0)>>1))})
	}
	return objectSchema(fields, properties)
}

func taxBreakdownSchema() bson.D {
	return objectSchema(bson.A{"rate", "taxableAmountMinor", "taxAmountMinor"}, bson.D{
		{Key: "rate", Value: integerProperty(0, int64(calculations.RateScale))},
		{Key: "taxableAmountMinor", Value: integerProperty(0, int64(^uint64(0)>>1))},
		{Key: "taxAmountMinor", Value: integerProperty(0, int64(^uint64(0)>>1))},
	})
}

func documentCounterSchema() bson.D {
	return objectSchema(bson.A{"ownerId", "year", "sequence", "updatedAt"}, bson.D{
		{Key: "_id", Value: bson.D{{Key: "bsonType", Value: "objectId"}}},
		{Key: "ownerId", Value: stringProperty(`^user-[0-9A-HJKMNP-TV-Z]{26}$`, 0, 0)},
		{Key: "year", Value: integerProperty(1, 9999)},
		{Key: "sequence", Value: integerProperty(1, 999999)},
		{Key: "updatedAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
	})
}

func auditSchema() bson.D {
	return objectSchema(bson.A{"_id", "ownerId", "actorId", "documentId", "documentReference", "action", "documentVersion", "metadata", "requestId", "occurredAt"}, bson.D{
		{Key: "_id", Value: stringProperty(`^audit-[0-9A-HJKMNP-TV-Z]{26}$`, 0, 0)},
		{Key: "ownerId", Value: stringProperty(`^user-[0-9A-HJKMNP-TV-Z]{26}$`, 0, 0)},
		{Key: "actorId", Value: stringProperty(`^user-[0-9A-HJKMNP-TV-Z]{26}$`, 0, 0)},
		{Key: "documentId", Value: stringProperty(`^doc-[0-9A-HJKMNP-TV-Z]{26}$`, 0, 0)},
		{Key: "documentReference", Value: stringProperty(`^DOC-[0-9]{4}-[0-9]{6}$`, 0, 0)},
		{Key: "action", Value: bson.D{{Key: "enum", Value: bson.A{"document.created", "document.updated", "document.finalized", "document.duplicated", "document.archived", "document.restored"}}}},
		{Key: "documentVersion", Value: integerProperty(1, int64(^uint64(0)>>1))},
		{Key: "metadata", Value: auditMetadataSchema()},
		{Key: "requestId", Value: stringProperty(`^[A-Za-z0-9._:-]{1,128}$`, 1, 128)},
		{Key: "occurredAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
	})
}

func auditMetadataSchema() bson.D {
	return objectSchema(bson.A{}, bson.D{
		{Key: "changedFields", Value: bson.D{{Key: "bsonType", Value: "array"}, {Key: "maxItems", Value: 12}, {Key: "uniqueItems", Value: true}, {Key: "items", Value: bson.D{{Key: "enum", Value: bson.A{"title", "customer", "issueDate", "currency", "lineItems", "totals", "taxBreakdown", "status", "archivedAt"}}}}}},
		{Key: "sourceDocumentId", Value: stringProperty(`^doc-[0-9A-HJKMNP-TV-Z]{26}$`, 0, 0)},
		{Key: "calculationPolicyVersion", Value: bson.D{{Key: "enum", Value: bson.A{calculations.PolicyVersion}}}},
	})
}

func refreshSessionSchema() bson.D {
	return objectSchema(bson.A{"_id", "userId", "tokenHash", "familyId", "createdAt", "expiresAt"}, bson.D{
		{Key: "_id", Value: stringProperty(`^session-[0-9A-HJKMNP-TV-Z]{26}$`, 0, 0)},
		{Key: "userId", Value: stringProperty(`^user-[0-9A-HJKMNP-TV-Z]{26}$`, 0, 0)},
		{Key: "tokenHash", Value: stringProperty(`^[0-9a-f]{64}$`, 64, 64)},
		{Key: "familyId", Value: stringProperty(`^session-[0-9A-HJKMNP-TV-Z]{26}$`, 0, 0)},
		{Key: "createdAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
		{Key: "expiresAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
		{Key: "revokedAt", Value: bson.D{{Key: "bsonType", Value: "date"}}},
		{Key: "replacedById", Value: stringProperty(`^session-[0-9A-HJKMNP-TV-Z]{26}$`, 0, 0)},
	})
}
