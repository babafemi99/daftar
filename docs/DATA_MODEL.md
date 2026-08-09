# DAFTAR (دَفْتَر) - MONGODB DATA MODEL

**Version:** 1.0  
**Date:** 8 August 2026  
**Product:** Daftar  
**Database:** MongoDB

This document defines the persistent data model for Daftar. MongoDB stores only server-validated inputs and server-calculated financial results; client-supplied totals are never authoritative.

## Modelling principles

1. Model around observed query and update patterns.
2. Keep a financial document and its bounded line items as one aggregate.
3. Use application validation plus MongoDB validators for critical invariants.
4. Store monetary amounts in integer minor units.
5. Store rates as scaled integers, not binary floating point.
6. Separate unbounded audit and session history from documents.
7. Scope every business query and unique constraint by owner where appropriate.

## Collections

### `users`

```json
{
  "_id": "ObjectId",
  "email": "finance@example.com",
  "emailNormalized": "finance@example.com",
  "passwordHash": "<argon2id hash>",
  "displayName": "Finance User",
  "reportingTimezone": "UTC",
  "createdAt": "ISODate",
  "updatedAt": "ISODate"
}
```

Indexes:

```javascript
{ emailNormalized: 1 } // unique
```

Rules:

- Normalize email before lookup and storage.
- Never return `passwordHash` from repositories used by general profile handlers.
- Account deletion is outside the assignment scope.

### `documents`

```json
{
  "_id": "ObjectId",
  "ownerId": "ObjectId",
  "reference": "DOC-2026-000001",
  "title": "August Software Services",
  "customer": "Acme Limited",
  "issueDate": "ISODate(2026-08-08T00:00:00.000Z)",
  "currency": "USD",
  "status": "draft",
  "lineItems": [
    {
      "id": "01K...",
      "description": "Widget A",
      "quantity": 2,
      "unitPriceMinor": 10000,
      "discount": {
        "type": "percentage",
        "value": 100000
      },
      "taxRate": 50000,
      "calculated": {
        "subtotalMinor": 20000,
        "discountAmountMinor": 2000,
        "discountedAmountMinor": 18000,
        "taxAmountMinor": 900,
        "lineTotalMinor": 18900
      }
    }
  ],
  "totals": {
    "subtotalMinor": 20000,
    "discountMinor": 2000,
    "taxMinor": 900,
    "grandTotalMinor": 18900
  },
  "taxBreakdown": [
    {
      "rate": 50000,
      "taxableAmountMinor": 18000,
      "taxAmountMinor": 900
    }
  ],
  "calculationPolicyVersion": "2026-08-v1",
  "version": 1,
  "finalizedAt": null,
  "archivedAt": null,
  "createdAt": "ISODate",
  "updatedAt": "ISODate"
}
```

Rate scale in the example is parts per million of the base amount:

```text
100% = 1,000,000
10%  =   100,000
5%   =    50,000
```

This supports up to four displayed decimal places in a percentage without floats. The API may accept decimal strings such as `"5.25"` and convert them to the scaled integer.

#### Why line items are embedded

- They have no independent lifecycle outside the document.
- A document is normally read with all its lines.
- Draft recalculation and finalization update the aggregate atomically.
- The assignment's line count is bounded and far below MongoDB's document-size limit.

The API enforces a practical maximum of 200 line items and description-length limits.

#### Stored calculated values

Both raw inputs and calculated values are stored because finalized documents should reproduce historical output efficiently. Calculated values are never accepted from the client and can always be verified from the raw inputs plus `calculationPolicyVersion`.

Indexes:

```javascript
{ ownerId: 1, reference: 1 }                         // unique
{ ownerId: 1, status: 1, issueDate: -1, _id: -1 }  // lists/reports
{ ownerId: 1, currency: 1, issueDate: -1 }          // currency reports
{ ownerId: 1, archivedAt: 1, updatedAt: -1 }        // active/archived drafts
{ ownerId: 1, status: 1, archivedAt: 1, issueDate: 1, currency: 1 } // summary reports
```

Optional search index for hosted MongoDB search:

```text
reference, title, customer scoped by ownerId
```

For the take-home, anchored case-insensitive search or normalized search fields may be used. Avoid an unindexed global regex scan.

Validation highlights:

- `ownerId`, reference, title, customer, issueDate, currency, status, version, and timestamps required.
- `status` in `draft | finalized`.
- `currency` in supported ISO code set.
- `lineItems` maximum 200.
- Quantity integer from 1 to 1,000,000.
- Minor-unit fields nonnegative signed 64-bit-safe integers.
- Discount type in `fixed | percentage` or absent.
- Tax and percentage discount rates between 0% and 100% inclusive.
- Finalized documents require `finalizedAt`.

MongoDB validators are installed idempotently at API startup for all
application-owned collections. New collections are created with strict/error
validation and existing collections are updated through `collMod`. They are
defense in depth: more expressive financial reconciliation and cross-field
rules remain in the domain service.

### `audit_events`

```json
{
  "_id": "ObjectId",
  "ownerId": "ObjectId",
  "actorId": "ObjectId",
  "documentId": "ObjectId",
  "documentReference": "DOC-2026-000001",
  "action": "document.finalized",
  "documentVersion": 4,
  "metadata": {
    "changedFields": ["lineItems", "totals"],
    "sourceDocumentId": null
  },
  "requestId": "req_...",
  "occurredAt": "ISODate"
}
```

Indexes:

```javascript
{ ownerId: 1, documentId: 1, occurredAt: -1 }
{ ownerId: 1, occurredAt: -1 }
```

Rules:

- Created only by backend services.
- No update/delete route.
- Metadata has a strict allowlist and size limit.
- Secrets, passwords, tokens, cookies, and raw request bodies are prohibited.

### `idempotency_records`

```json
{
  "_id": "ObjectId",
  "ownerId": "ObjectId",
  "scope": "POST:/api/v1/documents/{id}/finalize",
  "keyHash": "sha256(...)",
  "requestHash": "sha256(canonical request)",
  "state": "completed",
  "resourceId": "ObjectId",
  "responseStatus": 200,
  "responseBody": {},
  "createdAt": "ISODate",
  "expiresAt": "ISODate"
}
```

Indexes:

```javascript
{ ownerId: 1, scope: 1, keyHash: 1 } // unique
{ expiresAt: 1 }                     // TTL
```

The raw idempotency key is not stored. Response bodies must not contain secrets.

### `document_counters`

```json
{
  "_id": "ObjectId",
  "ownerId": "ObjectId",
  "year": 2026,
  "sequence": 14,
  "updatedAt": "ISODate"
}
```

Index:

```javascript
{ ownerId: 1, year: 1 } // unique
```

Counter increments are atomic. Sequence gaps are allowed if document creation later fails.

### `refresh_sessions`

```json
{
  "_id": "ObjectId",
  "userId": "ObjectId",
  "tokenHash": "sha256(...)",
  "familyId": "opaque identifier",
  "userAgentHash": "sha256(...)",
  "expiresAt": "ISODate",
  "revokedAt": null,
  "createdAt": "ISODate"
}
```

Indexes:

```javascript
{ tokenHash: 1 }  // unique
{ userId: 1, createdAt: -1 }
{ expiresAt: 1 } // TTL
```

## Data access patterns

| Operation | Query pattern | Supporting index |
|---|---|---|
| Get document | owner + document ID | `_id` plus owner filter |
| List by status/date | owner + status + date cursor | owner/status/issueDate/_id |
| Report by date | owner + finalized + issueDate | owner/status/issueDate/_id |
| Report by currency | owner + currency + date | owner/currency/issueDate |
| Audit timeline | owner + document + time | owner/document/occurredAt |
| Reference lookup | owner + reference | unique owner/reference |

## Mutation patterns

### Draft update

Use one conditional replacement/update on the aggregate:

```javascript
{
  _id: documentId,
  ownerId,
  status: "draft",
  archivedAt: null,
  version: expectedVersion
}
```

The server recalculates and sets the complete line/totals projection, increments `version`, and updates `updatedAt`.

### Finalization

Use a short transaction for:

1. Conditional document mutation.
2. Audit insert.
3. Idempotency completion.

The pricing calculation occurs before the transactional write using the loaded draft, followed by an expected-version condition to prevent stale finalization.

### Duplicate

Read the owned finalized source, allocate a new reference, recalculate inputs, then insert a new draft and audit event. The source document is never mutated.

## Data integrity invariants

- Every document belongs to exactly one user.
- Reference is unique within an owner's namespace.
- Grand total equals subtotal minus discount plus tax.
- Sum of line amounts equals stored document totals.
- Sum of tax breakdown equals total tax.
- A finalized document has `finalizedAt` and cannot be mutated through application commands.
- An archived document is always a draft.
- Rates and monetary values never use BSON double.
- Mixed currencies are aggregated into separate groups.

## Migration and evolution

- Each document stores `calculationPolicyVersion`.
- Schema migrations run as explicit, resumable scripts rather than silently rewriting records at application startup.
- New optional fields require backward-compatible readers before backfill.
- Finalized historical calculations are not automatically rewritten when calculation logic changes.
- Index creation is versioned and documented.
