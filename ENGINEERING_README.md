# Daftar

> **Reviewer:** Start with the visual, assignment-focused
> [Daftar Submission Guide](README.md). This file remains the concise
> repository and operational reference.

**Canonical repository:** <https://github.com/babafemi99/daftar>

[![Daftar CI](https://github.com/babafemi99/daftar/actions/workflows/ci.yml/badge.svg)](https://github.com/babafemi99/daftar/actions/workflows/ci.yml)

Daftar is a full-stack multi-rate pricing calculator for creating financial
documents with per-line discounts and taxes. It provides authenticated,
owner-scoped document management, deterministic server-side calculations, an
immutable finalized state, and issue-date summary reporting.

The repository is organized as a small monorepo:

```text
daftar/
├── backend/       Go REST API
├── frontend/      Next.js application
├── docs/          Product, API, calculation, and brand specifications
├── compose.yaml   Frontend, API, and MongoDB orchestration
└── Makefile       Repository-level development commands
```

## Engineering guide

This README documents how Daftar is built, why its boundaries exist, and how to
operate and verify it. The [submission guide](README.md) contains the visual
product story, assignment mapping and future positioning.

### Technology stack

| Layer | Technology | Responsibility |
| --- | --- | --- |
| Web application | Next.js 16, React, TypeScript | Responsive authenticated UI, editor, reports and print layout |
| API transport | Go 1.26, Chi | Routing, middleware, strict decoding, conversion and response envelopes |
| Domain | Plain Go packages | Document invariants, lifecycle and financial calculation policy |
| Persistence | MongoDB 8 | Owner-scoped aggregates, transactions, indexes, search and reports |
| Authentication | Argon2id, JWT, opaque refresh tokens | Password protection, short access sessions and rotating renewal |
| UI system | CSS, Phosphor Icons, Sonner | Branded components, interaction states and notifications |
| Verification | Go test/race/vet, Testcontainers, Playwright | Unit, repository, HTTP and browser acceptance coverage |
| Runtime | Docker, Compose, scratch/Alpine images | Reproducible local and deployment topology |

### Source layout

```text
backend/
├── cmd/api/                     composition root, startup and healthcheck CLI
└── internal/
    ├── buhari/                  stable application errors and field details
    ├── calculations/            integer-safe shared calculation engine
    ├── cfg/                     typed environment configuration
    ├── mercury/                 MongoDB repositories, indexes and transactions
    ├── model/                   persisted aggregates and domain input models
    ├── pkg/                     Argon2, JWT, ULID and request-context helpers
    ├── service/                 thin owner-scoped application orchestration
    ├── sys/                     dependency lifecycle and bootstrap wiring
    └── transport/web/           Chi routes, middleware, DTO conversion and JSON

frontend/
├── public/                      production brand assets
├── src/app/                     Next.js routes
├── src/components/              reusable shell, editor, report and state UI
├── src/lib/api.ts               typed HTTP client and transparent token refresh
└── tests/e2e/                   real authenticated browser workflow

docs/
├── API_SPEC.md                  HTTP contract and public error codes
├── CALCULATIONS.md              authoritative arithmetic and rounding policy
├── DATA_MODEL.md                MongoDB document and index design
├── SYSTEM_DESIGN.md             architecture and operational decisions
└── brand-daftar/                identity, tokens, guidelines and screen concepts
```

## Architecture

Daftar uses a layered modular-monolith design. Each boundary has one job:

```text
Browser / API client
        │
        ▼
Chi middleware and HTTP handlers
  authenticate · parse · convert · encode
        │
        ▼
Application services
  owner scope · command orchestration · transaction boundary
        │
        ▼
Document and calculation domain
  invariants · lifecycle · deterministic arithmetic
        │
        ▼
Mercury MongoDB repositories
  conditional writes · indexes · aggregation · append-only events
```

HTTP handlers do not contain calculation or lifecycle rules. Services do not
interpret JSON. Repositories do not decide business policy. This keeps the
highest-risk financial logic independently testable and prevents transport
changes from altering calculation behavior.

### Document aggregate

A financial document is the consistency boundary. Metadata, raw line inputs,
calculated line values, totals, tax breakdown, lifecycle state and version are
stored together. Embedded lines are a deliberate MongoDB choice:

- A complete document read requires one owner-scoped query.
- Financially relevant changes recalculate the whole aggregate.
- Conditional replacement persists lines and totals atomically.
- Reordering changes presentation order without altering financial totals.
- Finalization locks one internally consistent version.

Line and document IDs are server-generated prefixed ULIDs. Human references use
an owner/year counter such as `DOC-2026-000001` with a unique compound index.

### Mutation and concurrency flow

```text
Authenticate owner
  → load by owner ID + document ID
  → verify expected version
  → require active draft
  → mutate raw inputs
  → recalculate the complete aggregate
  → conditionally persist owner + ID + status + archive + version
  → append audit event in the same transaction
  → return the new version and ETag
```

Daftar never silently reloads and retries a user-authored stale mutation. A
failed conditional write returns `DOCUMENT_VERSION_CONFLICT`, allowing the user
to review the newer state instead of applying a command to data they never saw.

### Transactional audit trail

Material document writes and their audit events share one MongoDB transaction.
Events are append-only through the application and record actor, owner,
document/reference, resulting version, action, request ID, timestamp and a
small allowlisted metadata object. Raw financial payloads and sensitive values
are intentionally excluded.

Recorded actions include creation, draft updates, archive, restore,
finalization and duplication. Audit reads first verify access to the owned
document, preserving `RESOURCE_NOT_FOUND` for cross-owner access.

### Authentication lifecycle

1. Registration validates input and stores only an Argon2id password hash.
2. Login issues a 15-minute signed access JWT in an `HttpOnly` cookie.
3. A separate opaque refresh token is generated from cryptographic randomness;
   MongoDB stores only its SHA-256 hash.
4. Refresh consumes the current session once, rotates both cookies and retries
   the original frontend request.
5. Reuse of a consumed refresh token revokes the active session family.
6. Logout revokes the persisted refresh session and expires both cookies.
7. A TTL index removes expired refresh records automatically.

Bearer access tokens remain supported for non-browser clients. Refresh tokens
are never accepted by protected API middleware.

### Search, pagination and reporting

- Search uses a compound owner/text index weighted by reference, title and
  customer.
- Search combines with status, currency, inclusive issue-date and archive
  filters before pagination.
- The current ledger uses stable page-based pagination ordered by issue date
  and ID, returning total items, total pages and `hasMore`.
- Summary reporting uses stored server-calculated totals from finalized,
  non-archived documents and filters on `issueDate`.
- MongoDB aggregation groups money by currency and tax rate. Different
  currencies are never added together.

## Live application

The public deployment URL will be added here before submission.

## Product capabilities

### Dashboard and reviewer experience

- Personalized financial overview after sign-in
- Live counts for all documents, drafts, and finalized records
- Current-year finalized activity and currency-separated financial pulse
- Recent-document table with direct navigation
- Quick actions for document creation and reporting
- One-click CrossVal assignment sample using the exact mixed-discount and
  multi-tax inputs that produce the server-calculated `USD 421.50` total
- Responsive branded interface, loading screens, empty states, confirmation
  dialogs, Phosphor icons, and dismissible Sonner notifications
- Keyboard skip navigation, semantic current-page states, accessible tables,
  live save/calculation announcements, reduced motion, and focus-trapped dialogs

### Authentication and isolation

- Email and password registration and login
- Short-lived access JWTs plus rotating, hashed, server-revocable refresh
  sessions in separate `HttpOnly` cookies; Bearer-token support remains for API clients
- Owner-scoped document reads, writes, lists, and reports
- Cross-owner resources consistently returned as `RESOURCE_NOT_FOUND`
- Strict request decoding, request-size limits, rate limiting, trusted-proxy
  handling, CORS policy, origin checks, and security headers
- Optional idempotent bootstrap user for local review environments

### Document workspace

- Create drafts with zero or more embedded line items
- Edit document title, customer, issue date, currency, and complete line inputs
- Live server-backed calculation preview without allocating a reference or
  persisting anything
- Debounced autosave for valid existing drafts with visible unsaved, saving,
  saved, and failure states
- Explicit Save action remains available alongside autosave
- Add, update, duplicate, remove, and reorder line items
- Move-up and move-down controls plus `Cmd/Ctrl + Enter` line creation
- Automatic focus when adding or duplicating a line
- Indexed inline API validation that focuses and describes the exact failing
  field, including nested line discount and tax inputs
- Server-generated document and line ULIDs; clients cannot choose new IDs
- Archive and restore support instead of destructive document deletion

### Financial calculations

- Fixed or percentage discount per line, never both
- Independent percentage tax rates per line
- Tax calculated after discount
- Server-calculated line subtotals, discount amounts, taxable amounts, taxes,
  line totals, document totals, and tax breakdowns
- Integer minor-unit money and scaled-integer rates with no floating-point
  financial arithmetic
- Deterministic round-half-away-from-zero policy applied per line
- Overflow protection, maximum line count, and precise field-level validation
- Stored calculation-policy version and aggregate integrity verification

### Lifecycle and concurrency

- Drafts are fully editable and may temporarily contain no lines
- Finalization requires at least one valid line and performs a final server
  recalculation
- Finalized documents are permanently immutable
- Optimistic concurrency through document versions and quoted `If-Match`
  headers
- Conditional MongoDB writes scoped by owner, document, status, archive state,
  and expected version
- Finalized documents can be duplicated into independent drafts with fresh
  document and line IDs

### Ledger, reporting, and output

- Owner-scoped full-text search across reference, title, and customer, combined
  with status, currency, issue-date, and archive filters
- Page-based pagination with stable issue-date/ID ordering, total counts, and
  responsive Previous/Next controls
- Inclusive issue-date summary reporting over finalized documents
- Document count, subtotal, discount, tax, grand total, and grouped tax-rate
  breakdown for every currency
- Different currencies are always returned separately and never summed
  together
- Branded A4 printable finalized-document view with customer details, line
  items, taxes, totals, reference, finalization timestamp, and calculation
  policy
- Browser-native **Print / Save PDF** workflow without adding a rendering
  engine to the API image

### API and engineering quality

- Versioned REST API under `/api/v1`
- Strict JSON decoding that rejects unknown fields and multiple values
- Stable application error codes and a consistent Daftar error envelope
- Thin HTTP handlers over independently tested domain, calculation, service,
  and MongoDB repository layers
- Focused HTTP tests plus real authenticated production-router and MongoDB
  Testcontainers workflows
- Multi-stage, non-root containers and one Compose stack for Next.js, Go, and
  MongoDB

## Prerequisites

The recommended setup requires:

- Docker with Docker Compose v2

For development without containers, install:

- Go matching the version declared in `backend/go.mod`
- Node.js 24 or newer with npm
- MongoDB 8

## Run the complete application with Docker

Copy the example configuration and start the stack:

```sh
cp .env.example .env
docker compose up --build -d
```

Open:

- Frontend: <http://localhost:3000>
- API: <http://localhost:8080>
- API liveness check: <http://localhost:8080/api/v1/health/live>

Check the running services:

```sh
docker compose ps
curl http://localhost:8080/api/v1/health/live
```

Follow logs or stop the stack:

```sh
docker compose logs -f
docker compose down
```

MongoDB Atlas owns persistence independently of the Compose lifecycle.

Create the idempotent local reviewer dataset with:

```sh
make seed
```

This executes `/daftar-api -seed` as a one-shot Compose command. It obtains the
reviewer identity from the root `.env`, uses the normal user and audited
document services, creates active/finalized/archived examples, and exits. Use
`make seed-native` when the API and MongoDB dependencies run outside Docker.
The example credentials are intentionally local; override them before seeding
a public environment.

## Configuration reference

The root `.env` file is the runtime configuration source of truth. Compose
loads it directly into the API through `env_file` and uses it for safe
orchestration interpolation such as ports and the frontend's internal API URL.
Secrets are not injected into the frontend container. Configuration is loaded
into typed Go structs and validated before any server listener starts.
Production mode additionally requires secure cookies and rejects wildcard CORS
origins.

The Makefile uses that same file for native and Docker commands. Both execution
modes connect to the external Atlas URI; only the frontend's internal API
hostname differs between Compose and native execution. Override the file
consistently with `make ENV_FILE=.env.production <target>`.

| Variable | Default/example | Purpose |
| --- | --- | --- |
| `DAFTAR_ENVIRONMENT` | `development` | Enables environment-specific safety validation |
| `DAFTAR_LOG_LEVEL` | `info` | Minimum operational log level: `debug`, `info`, `warn`, or `error` |
| `DAFTAR_LOG_FORMAT` | `pretty` | Readable local output; production defaults to structured `json` when unset |
| `DAFTAR_LOG_SLOW_REQUEST_THRESHOLD` | `750ms` | Requests at or above this duration are logged at warning level |
| `DAFTAR_HTTP_ADDRESS` | `:8080` | API listen address |
| `DAFTAR_CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | Exact browser origins allowed to send credentialed requests |
| `DAFTAR_API_INTERNAL_URL` | `http://api:8080` | Server-side Next.js proxy destination inside Compose |
| `DAFTAR_API_INTERNAL_URL_NATIVE` | `http://localhost:8080` | Server-side Next.js proxy destination for `make frontend` |
| `DAFTAR_MONGODB_URI` | Atlas `mongodb+srv://...` URI | External database used by the container runtime |
| `DAFTAR_MONGODB_URI_NATIVE` | Atlas `mongodb+srv://...` URI | External database selected by `make run` |
| `DAFTAR_MONGODB_DATABASE` | `daftar` | Application database |
| `DAFTAR_JWT_SECRET` | no safe production default | HMAC key; must contain at least 32 characters |
| `DAFTAR_JWT_ISSUER` | `daftar-api` | Required JWT issuer |
| `DAFTAR_JWT_AUDIENCE` | `daftar-web` | Required JWT audience |
| `DAFTAR_JWT_ACCESS_TTL` | `15m` | Short-lived access-token lifetime |
| `DAFTAR_JWT_REFRESH_TTL` | `720h` | Persistent refresh-session lifetime |
| `DAFTAR_COOKIE_SECURE` | `false` locally | Must be `true` in production |
| `DAFTAR_COOKIE_SAME_SITE` | `lax` | `lax`, `strict`, or secure `none` |
| `DAFTAR_TRUSTED_PROXIES` | empty | Explicit IP/CIDR list allowed to supply forwarded addresses |
| `DAFTAR_BOOTSTRAP_USER_ENABLED` | `false` | Enables idempotent local reviewer-account creation |

HTTP read/write/idle timeouts, header limits and rate limits are also
environment-configurable; see `backend/internal/cfg/cfg.go` for the complete
typed configuration surface.

Operational logs are written to stdout for Docker or the deployment platform
to collect. Each HTTP completion record includes its request ID, route pattern,
status, duration and response size, plus the authenticated user ID and stable
application error code when available. Daftar does not log request bodies,
credentials, authorization headers, cookies, tokens, secrets or MongoDB URIs.
Use `make logs` locally; production should set `DAFTAR_LOG_FORMAT=json`.

## MongoDB collections and indexes

Strict MongoDB `$jsonSchema` validators and indexes are installed idempotently
during application startup. Missing collections are created with validators;
existing collections are upgraded through `collMod`. Startup fails if either
validators or indexes cannot be ensured, preventing the API from running
against an incorrect persistence shape.

The validators reject missing required fields, incorrect BSON types, unknown
stored properties, invalid resource-ID prefixes, unsupported currencies and
lifecycle states, floating-point money, excessive embedded lines, malformed
audit actions and unhashed refresh tokens. Cross-field financial reconciliation
remains in the domain because MongoDB validation is intentionally defense in
depth rather than a second calculation engine.

| Collection | Important indexes and purpose |
| --- | --- |
| `users` | Unique normalized email |
| `documents` | Unique owner/reference; owner/status/date; owner/currency/date; owner/archive/update; weighted owner/text search |
| `document_counters` | Unique owner/year sequence allocation |
| `audit_events` | Owner/document/time and owner/time timeline retrieval |
| `refresh_sessions` | Unique token hash, user/family lookup and expiry TTL |

The reporting aggregation is supported by an owner/status/archive/issue-date/
currency index. Integration tests exercise the real MongoDB query and index
behavior through Testcontainers.

## HTTP conventions

### Success envelope

```json
{
  "data": {},
  "requestId": "req-..."
}
```

Collections additionally include page metadata:

```json
{
  "data": [],
  "page": {
    "number": 1,
    "size": 10,
    "totalItems": 24,
    "totalPages": 3,
    "hasMore": true,
    "nextCursor": null
  },
  "requestId": "req-..."
}
```

### Error envelope

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "One or more fields are invalid.",
    "details": {
      "fields": [
        {
          "path": "lineItems[2].discount.value",
          "code": "INVALID_DISCOUNT_RATE",
          "message": "Rate must be a decimal percentage between 0 and 100."
        }
      ]
    },
    "requestId": "req-..."
  }
}
```

Request bodies are size-limited and decoded with unknown-field rejection. A
second or trailing JSON value is rejected. Money and rates cross the API as
decimal strings before conversion into integer domain representations.

### Middleware order and security

The HTTP stack applies request IDs, trusted-proxy handling, security headers,
panic recovery, timeouts, origin/CORS policy and IP rate limiting. Protected
routes then validate the access JWT and add a small typed executor to context.
State-changing browser requests must originate from an explicitly allowed
origin.

## Container and runtime design

The Compose topology contains two services and connects to Atlas externally:

```text
frontend :3000  →  api :8080  →  MongoDB Atlas
```

- The Go build uses cached module/build mounts, disables CGO, strips symbol
  tables and embeds timezone data.
- The API runs from `scratch` as numeric user `65532`, with only the binary and
  CA certificate bundle.
- The Next.js standalone output runs on Alpine as the unprivileged `nextjs`
  user.
- Atlas supplies the replica-set transactions used by document/audit and
  refresh-rotation flows.
- Compose waits for API health before starting the frontend.
- Database availability and persistence are independent of the VPS containers.

## Two-minute reviewer tour

1. Register or sign in at <http://localhost:3000>.
2. Open the dashboard and select **Create the 421.50 sample**.
3. Confirm the server-calculated subtotal `450.00`, discount `40.00`, tax
   `11.50`, and grand total `421.50`.
4. Edit a field and pause briefly to see draft autosave complete.
5. Duplicate or reorder a line, then observe the complete recalculation.
6. Finalize the document and confirm that the editor becomes read-only.
7. Select **Print / Save PDF** to inspect the branded A4 output.
8. Open Reports and choose a range containing the document's issue date.
9. Return to Documents to try filtering and pagination.

## Run without Docker

Copy the example environment file and configure its Atlas URI:

```sh
cp .env.example .env
```

Run both processes from the repository root. The Makefile loads `.env` and
selects its native MongoDB and API endpoints automatically:

```sh
make dev
```

Use `make run` and `make frontend` in separate terminals when independent
process control is preferred.

```sh
cd frontend
npm install
cd ..
make frontend
```

The Next.js application proxies `/api/v1/*` to
`DAFTAR_API_INTERNAL_URL_NATIVE`, which defaults to
`http://localhost:8080`.

## Calculation and rounding policy

Money is represented as signed 64-bit integer minor units. Rates use an integer
scale of `1,000,000`, so no binary floating-point arithmetic is used for
financial calculations.

Each line is calculated in this order:

1. `subtotal = quantity × unit price`
2. Calculate a fixed discount or a percentage discount, never both.
3. `discounted amount = subtotal − discount amount`
4. Calculate tax from the discounted amount.
5. `line total = discounted amount + tax amount`

Percentage-derived money is rounded to the nearest minor unit using round-half
away from zero. Valid inputs are nonnegative, so exact halves round upward.
Document totals are the sums of the already-rounded line results. This ensures
that displayed lines always reconcile with the document totals.

A fixed discount greater than its line subtotal is rejected rather than
clamped. The complete policy and boundary cases are documented in
`docs/CALCULATIONS.md`.

### Worked assignment example

| Line | Quantity | Unit price | Discount | Tax | Subtotal | Discount amount | Tax amount | Line total |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Widget A | 2 | 100.00 | 10% | 5% | 200.00 | 20.00 | 9.00 | 189.00 |
| Widget B | 1 | 50.00 | — | 5% | 50.00 | 0.00 | 2.50 | 52.50 |
| Service fee | 1 | 200.00 | 20.00 fixed | — | 200.00 | 20.00 | 0.00 | 180.00 |

The resulting document totals are:

```text
Subtotal:       450.00
Total discount:  40.00
Total tax:       11.50
Grand total:    421.50
```

Tax is applied after discount. For example, Widget A's tax is 5% of `180.00`,
not 5% of `200.00`.

## Document lifecycle

A new document starts as a `draft`. Draft metadata and lines are editable, and
a draft may temporarily contain no lines. Financially relevant changes cause
the server to recalculate the complete document.

Finalization is a one-way transition:

- A document must contain at least one valid line before finalization.
- The server revalidates and recalculates it immediately before finalizing.
- A finalized document is immutable: metadata, lines, amounts, and currency
  cannot be changed.
- Attempts to mutate a finalized document receive a specific application error.
- A finalized document can optionally be duplicated into a new draft with new
  server-generated document and line IDs.

Drafts can be archived and restored. Finalized documents cannot be archived or
restored as drafts. Official summary reports include only finalized,
non-archived documents.

## API overview

The REST API is served under `/api/v1`. It provides authentication, document
and embedded-line operations, calculation preview, finalization, duplication,
owner-scoped audit timelines, and owner-scoped summary reporting.

All calculated values are produced by the backend. Clients cannot select
document IDs, select new line IDs, or submit calculated totals. Mutating an
existing document requires its quoted version in the `If-Match` header.

See `docs/API_SPEC.md` for request bodies, responses, routes, and stable error
codes.

## Summary reporting

`GET /api/v1/reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` aggregates stored
server-calculated values for the authenticated owner's finalized documents.
Both boundaries are inclusive and filter by document issue date.

The response contains document count, subtotal, total discount, total tax,
grand total, and tax breakdowns grouped by rate. Different currencies are
returned as separate summaries and are never added together.

## Audit trail

Every successful material document mutation appends an immutable, owner-scoped
event in the same MongoDB transaction as the document write. Creation, draft
updates, archive, restore, finalization, and duplication record the actor,
document version, request ID, timestamp, and limited non-sensitive metadata.

`GET /api/v1/documents/{documentId}/audit-events` returns the authenticated
owner's timeline newest first. Cross-owner requests preserve the API's
`RESOURCE_NOT_FOUND` behavior. Document detail screens render the same history
as a responsive activity timeline with distinct lifecycle states.

## Tests and quality checks

The [Daftar CI workflow](.github/workflows/ci.yml) runs automatically for
pushes and pull requests targeting `main`, and can also be dispatched manually.
Independent backend, frontend and Testcontainers jobs run first; only after all
three pass does CI build the production Compose stack and execute the complete
authenticated Playwright workflow. Failed browser runs retain screenshots,
traces and Docker logs for seven days, while concurrency control cancels stale
runs from the same branch.

### Test matrix

| Layer | What it proves |
| --- | --- |
| Calculation unit tests | Worked sample, discounts, tax-after-discount, rounding, reconciliation, validation and overflow |
| Model/domain tests | Lifecycle invariants, deep-copy ownership, line IDs, totals and tax-breakdown integrity |
| Service-focused tests | Filter validation, line orchestration, preview non-persistence and owner scope |
| HTTP tests | Authentication, strict decoding, `If-Match`, DTO conversion, envelopes and application status mapping |
| MongoDB integration | Indexes, owner isolation, conditional writes, search, aggregation, audit order and refresh rotation |
| Production-router integration | Real auth, service, calculation and repository workflow through registered routes |
| Race detector | Concurrent Go access across all non-integration packages |
| Playwright acceptance | Real registration, sample creation, autosave, lifecycle, print, reporting and cross-owner rejection |

Integration tests are build-tagged so the fast unit suite does not require
Docker. The integration fixture starts MongoDB 8 with a replica set through
Testcontainers and uses a fresh database per test.

Run the backend unit suite:

```sh
cd backend
go test ./...
go test -race ./...
go vet ./...
```

With Docker running, execute the uncached MongoDB and production-router
integration suites:

```sh
cd backend
go test -count=1 -tags=integration ./internal/mercury ./internal/transport/web
```

Run frontend validation:

```sh
cd frontend
npm run lint
npm run typecheck
npm run build
npm run test:e2e
```

The calculation suite includes the assignment sample, fixed and percentage
discounts, tax-after-discount behavior, half-minor-unit rounding, line-total
reconciliation, validation boundaries, and overflow protection.

The Playwright acceptance workflow uses real Chromium, the production Next.js
router, authenticated Go API, and MongoDB. It registers two owners, creates the
`421.50` assignment sample, verifies autosave, finalizes it, checks printable
output, audit history, search, pagination, refresh and reporting, and proves
cross-owner access returns `RESOURCE_NOT_FOUND`.

## Operational checks and troubleshooting

Inspect the stack:

```sh
docker compose ps
docker compose logs api
curl --fail http://localhost:8080/api/v1/health/live
```

Common local issues:

- **API cannot connect to MongoDB:** verify the Atlas URI, database-user role
  and Atlas Network Access allowlist for the VPS outbound address.
- **Old login stops working after auth changes:** sign in once to obtain the
  current access and rotating refresh cookies.
- **Port collision:** override `DAFTAR_FRONTEND_PORT` or `DAFTAR_API_PORT` in
  `.env`.
- **Origin rejected:** ensure the browser URL exactly matches an entry in
  `DAFTAR_CORS_ALLOWED_ORIGINS`.
- **Need a clean database:** remove the intended records in Atlas explicitly;
  `docker compose down` never deletes database data.

### Safe shutdown

The application composition root owns HTTP and MongoDB lifecycles. Shutdown
stops accepting requests, drains the HTTP server within its configured timeout
and disconnects the MongoDB client. Container restart policies are
`unless-stopped` for local resilience.

## Assumptions and trade-offs

- Quantity is a positive whole number. Fractional quantities are outside the
  assignment scope.
- Supported currencies use two decimal minor units in this implementation.
- Reports include finalized documents only because drafts are not official
  financial records.
- Dates are stored as UTC calendar dates and report boundaries are inclusive.
- Fixed discounts larger than the line subtotal are rejected.
- MongoDB embeds line items in their owning document so complete-document
  recalculation and conditional replacement remain atomic.
- A finite supported-currency set provides strict validation. Currency
  conversion and exchange rates are intentionally excluded.
- Archive is used instead of destructive document deletion.
- Duplicate-finalized-document support is included as an optional assignment
  enhancement.
- Printable output uses the browser's native print/PDF support. Dedicated
  server-rendered PDF storage and delivery remain production enhancements.
- The current ledger uses page-based pagination for a clear reviewer-facing
  experience. Opaque cursor pagination remains preferable at very large scale.

## Before production

The following would be addressed before operating Daftar as a production
financial system:

- Deploy behind managed TLS with rotated secrets and secure cookies.
- Add backups, restoration drills, monitoring, tracing, and alerting.
- Add frontend component and browser end-to-end test suites.
- Add a versioned migration process for future validator and model evolution.
- Add server-rendered, versioned PDF storage where regulated delivery or exact
  cross-browser reproduction is required.
- Replace page offsets with opaque cursor pagination for very large document
  collections.
- Pin frontend dependency ranges and automate dependency/security updates.
- Perform an external security and accessibility review.

## Further documentation

- `docs/PRD.md` — product scope and decisions
- `docs/API_SPEC.md` — HTTP contract and errors
- `docs/CALCULATIONS.md` — authoritative financial policy
- `docs/DATA_MODEL.md` — MongoDB aggregate design
- `docs/SYSTEM_DESIGN.md` — architecture and operational decisions
