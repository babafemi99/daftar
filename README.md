<p align="center">
  <img src="docs/brand-daftar/daftar-compact.svg" width="560" alt="Daftar compact ledger symbol and wordmark" />
</p>

<h1 align="center">Daftar — The modern financial register</h1>

<p align="center">
  <strong>From line-item arithmetic to an auditable, immutable business record.</strong>
</p>

<p align="center">
  <a href="https://github.com/babafemi99/daftar/actions/workflows/ci.yml"><img src="https://github.com/babafemi99/daftar/actions/workflows/ci.yml/badge.svg" alt="Daftar CI status" /></a>
</p>

<p align="center">
  <a href="https://daftar.benjys.me">Live demo</a> ·
  <a href="#five-minute-reviewer-tour">Reviewer tour</a> ·
  <a href="#complete-implemented-feature-inventory">Features</a> ·
  <a href="#step-by-step-setup">Setup</a> ·
  <a href="ENGINEERING_README.md">Engineering</a>
</p>

Daftar is a production-minded, full-stack multi-rate pricing calculator for
authenticated financial documents. It does more than produce the right total:
it protects ownership, preserves calculation integrity, prevents lost updates,
locks finalized records, records an immutable audit history and turns the
result into a polished document workflow.

The system combines a Go REST API, external MongoDB Atlas persistence and a
responsive Next.js interface. The application containers share one Docker
Compose stack while Atlas remains independent of the deployment host.
Financial arithmetic is deterministic and server-owned; the browser never
supplies calculated totals.

## The name: Daftar (دَفْتَر)

**Daftar** means a notebook, ledger or official register across
Arabic-influenced business cultures in the Middle East and South Asia. It is a
small word with exactly the right product promise: a trusted place where the
numbers are written down, kept in order and available when they must be
revisited.

That meaning maps naturally to the CrossVal multi-rate pricing assignment. The
brief is not only about reaching a total; it asks us to turn many line-level
decisions—quantity, unit price, fixed or percentage discount and tax rate—into
one reliable financial document. Daftar treats each document like an entry in
a modern register:

- A **draft page** can be revised while the commercial details are still being
  agreed.
- Server calculations keep every line and total internally consistent.
- **Finalization closes the entry** and makes it immutable.
- The audit trail records who changed the register, what kind of change
  occurred, when it happened and which document version resulted.
- Reporting reads finalized entries back as official business information,
  without mixing currencies into a misleading number.

The bilingual mark pairs **Daftar** with **دَفْتَر** as a supporting cultural
signature. The symbol itself is a D-shaped open ledger: horizontal strokes
suggest written entries, while the copper vertical stroke reads as the binding
that holds the record together. It compresses the whole engineering idea into
one mark—structured inputs on the page, preserved by a dependable spine.

The visual language follows the same story:

- **Palm green** communicates trust, continuity and the calm seriousness of a
  financial record.
- **Copper** adds warmth, trade and human craft to what could otherwise feel
  like cold accounting software.
- **Paper and ivory** surfaces keep the interface connected to the familiar
  experience of a physical ledger.
- Restrained typography, generous spacing and small mono references make the
  product feel official without becoming intimidating.

This is not decorative localization pasted onto a generic calculator. The name,
lifecycle, audit model, document editor and reporting experience all reinforce
the same idea: **Daftar is where a calculation becomes a record.** Arabic
localization and right-to-left layout remain deliberate future enhancements;
the Arabic mark represents the brand story rather than claiming that work is
already complete.

## Why this implementation stands out

| Area | What Daftar delivers |
| --- | --- |
| Financial correctness | Integer minor-unit money, scaled rates, explicit rounding, overflow protection and complete aggregate recalculation |
| Document workflow | Draft autosave, calculation preview, line CRUD/reordering, archive/restore, immutable finalization and safe duplication |
| Auditability | Transactional append-only events for creation, edits, archive, restore, finalization and duplication |
| Concurrency | Versioned aggregates, required `If-Match` and owner/status/archive/version-scoped conditional writes |
| Security | Owner isolation, short-lived access JWTs, rotating hashed refresh sessions, replay detection, strict JSON and stable errors |
| Retrieval | Weighted owner-scoped search, filters, stable pagination and inclusive issue-date reporting |
| Reviewer experience | One-click `421.50` sample, live calculations, polished states, responsive layout, audit timeline and printable A4 output |
| Engineering | Thin transports, tested domain/calculation/repository boundaries, Testcontainers, race checks and real-browser acceptance coverage |

## Complete implemented feature inventory

Everything in this section is implemented now—not a roadmap item.

### Accounts, sessions and isolation

- [x] Email/password registration, login, current-user lookup and logout
- [x] Argon2id password hashing
- [x] Short-lived access JWT in a separate `HttpOnly` cookie
- [x] Persistent rotating refresh sessions stored only as token hashes
- [x] Refresh-token replay detection, family revocation and TTL cleanup
- [x] Bearer-token support for API clients
- [x] Owner-scoped reads, writes, lists, search, reports and audit events
- [x] `RESOURCE_NOT_FOUND` for missing and cross-owner resources

### Documents and line items

- [x] Create a draft with zero or more lines
- [x] Retrieve and list owned documents
- [x] Replace editable metadata and complete line inputs
- [x] Server calculation preview with no persistence or reference allocation
- [x] Add, update, delete, duplicate and reorder embedded line items
- [x] Server-generated document IDs, line IDs and human-readable references
- [x] Draft archive and restore
- [x] Finalize a valid non-empty document
- [x] Duplicate a finalized document into a fresh draft with fresh IDs
- [x] Enforced maximum line count and complete field-level validation

### Financial calculation engine

- [x] Per-line fixed or percentage discounts
- [x] Independent per-line percentage tax rates
- [x] Tax applied after discount
- [x] Integer minor-unit money and scaled integer rates
- [x] Round-half-away-from-zero at the documented calculation boundaries
- [x] Per-line calculated values, document totals and grouped tax breakdown
- [x] Overflow protection and fixed-discount limit validation
- [x] Stored calculation-policy version and aggregate integrity verification
- [x] Currency-separated documents and reports with no cross-currency addition

### Lifecycle, concurrency and auditability

- [x] Immutable finalized documents
- [x] Required quoted `If-Match` versions for document mutations
- [x] Conditional MongoDB writes scoped by owner, ID, lifecycle and version
- [x] Stale-write rejection with `DOCUMENT_VERSION_CONFLICT`
- [x] Transactional audit events committed with document mutations
- [x] Audit actions for create, update, archive, restore, finalize and duplicate
- [x] Owner-scoped audit API and responsive visual timeline

### Search, filters and pagination

- [x] **Backend full-text search is implemented** across reference, title and customer
- [x] **Frontend search UI is implemented** on the Documents ledger
- [x] Weighted, owner-prefixed MongoDB text index
- [x] Search combined with status, currency, issue-date and archive filters
- [x] **Backend page-based pagination is implemented** with total counting
- [x] **Frontend Previous/Next pagination is implemented** with current-page metadata
- [x] Stable issue-date and ID ordering
- [x] Page resets to `1` whenever filters or search change

The current pagination is deliberately page-based and fully functional. Opaque
cursor pagination is listed later only as a possible high-scale replacement;
it is not a missing pagination feature.

### Reporting, output and frontend experience

- [x] Finalized-document summary reporting over inclusive issue dates
- [x] Per-currency document count, subtotal, discount, tax and grand total
- [x] Tax breakdown grouped by rate
- [x] Responsive dashboard, document ledger, editor, reports and detail views
- [x] Debounced server calculation preview and valid-draft autosave
- [x] Visible dirty, saving, saved and error states
- [x] Indexed inline validation with focus management
- [x] Phosphor icons and dismissible Sonner notifications
- [x] Keyboard navigation, skip links, focus-trapped dialogs and reduced motion
- [x] Branded loading, empty, error and confirmation states
- [x] Branded A4 document view with browser Print / Save PDF
- [x] One-click creation of the exact `421.50` assignment sample

### Delivery and verification

- [x] One Docker Compose application stack for Next.js and Go, backed by external MongoDB Atlas
- [x] Atlas replica-set transactions without a database container on the deployment host
- [x] Startup-managed strict MongoDB schema validators for every persisted collection
- [x] Multi-stage, non-root frontend and API images
- [x] Correlated structured operational logs with request, user and error-code context
- [x] GitHub Actions quality, integration and production-stack acceptance pipeline
- [x] Unit, race, HTTP, repository and Testcontainers integration coverage
- [x] Real authenticated Playwright browser workflow
- [x] API, calculation, data-model, architecture and brand documentation

## Deployed URL

> **Live application:** [https://daftar.benjys.me](https://daftar.benjys.me)

Include the same URL in the submission email.

## Prerequisites

The recommended setup only requires:

- Docker Desktop or Docker Engine
- Docker Compose v2 (`docker compose`)
- Git

For development outside Docker, install Go matching `backend/go.mod`, Node.js
24 or newer and npm. Every runtime uses the external MongoDB Atlas database
configured in `.env`.

## Step-by-step setup

1. Clone the repository and enter it:

   ```sh
   git clone https://github.com/babafemi99/daftar.git
   cd daftar
   ```

2. Create the local environment file:

   ```sh
   cp .env.example .env
   ```

   `.env` is the configuration source of truth for the Compose stack. The API
   receives it directly; only explicitly safe frontend settings are exposed to
   the Next.js container.

3. Build and start the complete stack:

   ```sh
   docker compose up --build -d
   ```

4. Confirm that all services are running:

   ```sh
   docker compose ps
   curl http://localhost:8080/api/v1/health/live
   ```

5. Optionally create the idempotent reviewer account and demonstration ledger:

   ```sh
   make seed
   ```

   Then sign in at <http://localhost:3000> with
   `reviewer@daftar.local` / `DaftarDemo2026!`, or register your own account.

6. From the dashboard, choose **Create the 421.50 sample** for a quick
   verification of mixed discounts and tax rates.

### Local endpoints

| Service | URL |
| --- | --- |
| Next.js application | <http://localhost:3000> |
| Go REST API | <http://localhost:8080> |
| Liveness endpoint | <http://localhost:8080/api/v1/health/live> |
| Database | External MongoDB Atlas cluster configured in `.env` |

### Optional reviewer account

`make seed` runs the API image as a one-shot command, creates the reviewer from
the root `.env`, and exits without starting another server. It also creates an
empty draft, a mixed-tax/discount draft, a finalized invoice and an archived
draft through the real application services. Re-running it creates no duplicate
users or documents and repairs a partially completed finalize/archive step.
The published credentials are local demonstration values only; replace them
before seeding any public environment.

### Run without Docker

Copy `.env.example`, configure the Atlas URI, then run the backend from the
repository root. The Makefile loads the same root `.env` used by Compose:

```sh
make dev
```

This starts the Go API and Next.js frontend together. Use `make run` or
`make frontend` when only one process is needed.

For separate terminals:

```sh
cd frontend
npm install
cd ..
make frontend
```

The frontend proxies `/api/v1/*` to `DAFTAR_API_INTERNAL_URL_NATIVE`, which
defaults to `http://localhost:8080` during native local development.

### API capability map

All application routes live under `/api/v1` and return the standard Daftar
success or error envelope.

| Capability | Representative endpoint |
| --- | --- |
| Register and authenticate | `POST /auth/register`, `POST /auth/login` |
| Rotate or end a session | `POST /auth/refresh`, `POST /auth/logout` |
| Create and list documents | `POST /documents`, `GET /documents` |
| Search, filter and paginate | `GET /documents?search=...&page=1&limit=10` |
| Retrieve or replace a document | `GET /documents/{id}`, `PATCH /documents/{id}` |
| Preview server calculations | `POST /documents/preview-calculation` |
| Manage embedded lines | `/documents/{id}/line-items` and line-specific routes |
| Reorder lines | `POST /documents/{id}/line-items/reorder` |
| Archive and restore | `DELETE /documents/{id}`, `POST /documents/{id}/restore` |
| Finalize and duplicate | `POST /documents/{id}/finalize`, `POST /documents/{id}/duplicate` |
| Read the audit timeline | `GET /documents/{id}/audit-events` |
| Run a summary report | `GET /reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` |

Mutating existing documents require a quoted version in `If-Match`. Complete
request, response and error contracts are documented in `docs/API_SPEC.md`.

## Five-minute reviewer tour

1. Sign in and inspect the dashboard's live document counts and
   currency-separated finalized totals.
2. Select **Create the 421.50 sample** to generate the assignment example with
   mixed fixed/percentage discounts and multiple tax rates.
3. Edit a line and pause: the preview recalculates on the server, then the valid
   draft autosaves with a visible save state.
4. Add, duplicate, reorder and remove lines. IDs remain server-owned and every
   financial mutation recalculates the complete aggregate.
5. Finalize the document. The editor becomes a read-only financial record and
   later mutations are rejected.
6. Scroll to **Document activity** to see the versioned audit timeline for
   creation, edits and finalization.
7. Duplicate the finalized record and confirm the new draft receives a fresh
   reference, document ID and line IDs.
8. Visit Documents to combine text search, lifecycle/currency/date filters and
   pagination.
9. Visit Reports to aggregate finalized records by inclusive issue date,
   separated by currency and tax rate.
10. Use **Print / Save PDF** to inspect the branded A4 layout.

Useful commands:

```sh
docker compose logs -f
docker compose down
```

MongoDB Atlas owns database persistence independently of the Compose lifecycle;
stopping or rebuilding the containers does not remove application data.

### Run the verification suites

```sh
cd backend
go test ./...
go test -race ./...
go vet ./...
go test -count=1 -tags=integration ./internal/mercury ./internal/transport/web
```

```sh
cd frontend
npm install
npm run lint
npm run typecheck
npm run build
npm run test:e2e
```

The automated coverage includes the worked calculation, rounding boundaries,
discount and tax invariants, aggregate integrity, optimistic concurrency,
conditional MongoDB writes, audit isolation, refresh-token rotation/replay,
search ownership, HTTP envelopes and an authenticated browser workflow.

## Architecture at a glance

```text
Next.js UI
   │  secure cookies / JSON / If-Match
   ▼
Go HTTP transport
   │  authenticate, parse, convert, encode
   ▼
Application services
   │  owner-scoped orchestration
   ▼
Document + calculation domain
   │  invariants and deterministic arithmetic
   ▼
MongoDB repositories
      conditional aggregate writes, transactions, indexes and reporting
```

Line items are embedded inside their owning document. This makes a complete
document replacement atomic and ensures stored lines, totals, tax breakdown and
version always describe the same aggregate state.

## Transactional audit trail

Auditability is a first-class backend capability, not a UI-only activity feed.
Every successful material document mutation writes its audit event in the
**same MongoDB transaction** as the document change. If either write fails,
neither is committed.

Recorded actions are:

- `document.created`
- `document.updated` — including metadata and embedded-line mutations
- `document.archived`
- `document.restored`
- `document.finalized`
- `document.duplicated`

Each append-only event records:

- The owner and authenticated actor
- Document ID, reference and resulting version
- Action and UTC occurrence time
- Request ID for operational correlation
- A small allowlisted set of changed fields
- Source document ID for duplication, where applicable
- Calculation-policy version at finalization

The event model deliberately excludes raw sensitive payloads and client totals.
MongoDB indexes support owner/document/time retrieval, and the public endpoint
first verifies access to the owned document:

```http
GET /api/v1/documents/{documentId}/audit-events?limit=25
```

Cross-owner requests return `RESOURCE_NOT_FOUND`, matching the rest of the
document API. The frontend renders these events newest-first as a responsive,
accessible timeline with distinct creation, update, lifecycle and finalization
states. Existing history cannot be edited or deleted through the application.

## Authentication and owner isolation

- Access JWTs are short-lived and stored in an `HttpOnly` cookie.
- Refresh tokens are opaque random values; only SHA-256 hashes are persisted.
- Refresh sessions rotate once, expire through a TTL index and are revocable.
- Reusing a consumed token is detected and revokes the active session family.
- The frontend coordinates refresh and retries the original request once.
- Every document, report, search and audit query is scoped by the authenticated
  owner. Inaccessible IDs intentionally look missing.

## Calculation and rounding policy

Money is represented as signed 64-bit integer minor units. Rates are scaled
integers with a scale of `1,000,000`. Financial calculations never use binary
floating-point arithmetic.

Each line is evaluated in this order:

1. `subtotal = quantity × unit price`
2. Calculate either a fixed or percentage discount.
3. `discounted amount = subtotal − discount amount`
4. Calculate tax from the discounted amount.
5. `line total = discounted amount + tax amount`

Percentage-derived money is rounded to the nearest minor unit using
round-half-away-from-zero. Rounding occurs per line, and document totals sum
the displayed, already-rounded line results. A fixed discount greater than the
line subtotal is rejected rather than silently clamped.

### Worked example

| Line | Qty | Unit price | Discount | Tax | Subtotal | Discount | Tax amount | Line total |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Widget A | 2 | 100.00 | 10% | 5% | 200.00 | 20.00 | 9.00 | 189.00 |
| Widget B | 1 | 50.00 | — | 5% | 50.00 | 0.00 | 2.50 | 52.50 |
| Service fee | 1 | 200.00 | 20.00 fixed | 0% | 200.00 | 20.00 | 0.00 | 180.00 |

```text
Subtotal:       450.00
Total discount:  40.00
Total tax:       11.50
Grand total:    421.50
```

Tax is applied after discount: Widget A is taxed at 5% of `180.00`, not
`200.00`. The complete policy is in `docs/CALCULATIONS.md`.

## Finalization and immutability

- Documents begin as editable drafts and may temporarily contain no lines.
- Finalization requires at least one valid line.
- The server revalidates and completely recalculates the document immediately
  before finalization.
- Finalization is a one-way transition. Finalized metadata, line inputs,
  currency and calculated values cannot be changed.
- All mutations use optimistic concurrency through a quoted `If-Match`
  document version.
- Finalized documents may be duplicated into new drafts. The duplicate receives
  a fresh document ID, reference and fresh line IDs.
- Drafts may be archived and restored; finalized documents cannot be restored
  into editable drafts.
- Material lifecycle changes and edits are recorded in the append-only audit
  timeline in the same MongoDB transaction as the document mutation.

## Assumptions and trade-offs

- Quantity is a positive whole number; fractional quantities are outside the
  assignment scope.
- Supported currencies use two decimal minor units. Currency conversion and
  exchange rates are excluded.
- Summary reports contain finalized, non-archived records because drafts are
  not official financial records.
- Issue dates are UTC calendar dates and report boundaries are inclusive.
- Line items are embedded in their MongoDB document so aggregate replacement,
  recalculation and version checks remain atomic.
- Page-based pagination is fully implemented in the repository, API and UI.
  Opaque cursors would only replace it if dataset scale justified the added
  complexity.
- Full-text search is fully implemented using MongoDB's owner-prefixed weighted
  text index across reference, title and customer.
- Printable output uses the browser's native print/PDF workflow; the API image
  therefore does not require Chromium or font packages.

## Improvements already made toward production

Beyond the core calculator workflow, this implementation includes:

- Short-lived access JWTs and rotating, hashed, server-revocable refresh
  sessions with replay detection and TTL cleanup.
- Strict owner isolation that returns `RESOURCE_NOT_FOUND` for cross-owner
  access.
- Conditional writes and optimistic concurrency to prevent lost updates.
- Transactional, append-only, owner-scoped audit events with actor, request,
  version and limited metadata—plus a polished document timeline.
- Owner-scoped text search, filters and stable page-based pagination.
- Server-generated ULIDs for documents and line items.
- Strict JSON decoding, stable error envelopes, request limits, rate limiting,
  origin/CORS checks, trusted-proxy handling and security headers.
- Redaction-safe structured operational logging with request IDs, route patterns,
  authenticated user correlation, stable error codes, latency and response size.
- Integer-safe calculations, overflow protection and stored aggregate
  integrity validation.
- Multi-currency reports that never combine unrelated currencies.
- Responsive and accessible UI states, keyboard support, Phosphor icons,
  Sonner notifications and branded loading experiences.
- Multi-stage non-root application containers backed by an external Atlas
  replica set for transactional writes and host-independent persistence.
- Idempotent MongoDB `$jsonSchema` validators that reject malformed direct
  writes across users, documents, counters, audits and refresh sessions.
- Unit, race, repository integration, production-router integration and real
  browser workflow coverage.

### UX and product improvements beyond the basic calculator

- A reviewer shortcut that builds the exact assignment sample in one click.
- Debounced server calculation preview and draft autosave with clear save
  feedback.
- Reusable document editor, line editor, totals, filters, pagination, report,
  dialog and state components.
- Indexed nested validation messages such as
  `lineItems[2].discount.value`, with focus moved to the failing field.
- Search across reference, title and customer, combined with every current
  filter and pagination.
- Dashboard summaries and currency-separated reports based only on finalized
  records.
- Branded loading, empty and failure states; Sonner toasts with dismiss buttons;
  Phosphor icons; keyboard shortcuts; reduced-motion support; and responsive
  mobile navigation.
- A finalized document presentation designed for clean A4 printing.

## What I would improve before production

- Deploy behind managed TLS using rotated secrets, secure cookies and exact
  production CORS origins.
- Add automated backups, tested restoration procedures, metrics, tracing,
  centralized aggregation of the existing structured logs and alerting.
- Add a documented versioned migration process for future schema evolution.
- Store refresh-session device and security metadata and provide a user-facing
  session-management screen.
- Add account recovery, email verification and stronger abuse controls.
- Replace offset pagination with opaque cursors for very large datasets.
- Add server-rendered, versioned PDFs where exact regulated output is required.
- Pin dependency ranges, automate security scanning and establish an upgrade
  policy.
- Expand browser coverage across engines and complete an external security and
  accessibility review.

## Product direction: BOQs for the people who do the work

Daftar can already represent a **simple BOQ-style priced schedule**: each line
has a description, quantity, unit rate, discount, tax and calculated total;
lines can be reordered, the whole schedule recalculates, and the result can be
finalized, audited, reported and printed.

It is not yet a complete construction-grade BOQ system. A professional BOQ
usually also needs:

- Units of measurement such as `m`, `m²`, `m³`, `kg`, `nr` and `lot`
- Decimal quantities rather than whole-number quantities only
- Item codes, bill numbers, sections, headings and subtotals
- Specifications, dimensions, locations and trade/work classifications
- Rate build-ups for labour, material, plant and wastage
- Provisional sums, contingencies, retention and variation management
- Tender comparison, revisions and approval workflows
- BOQ-specific spreadsheet/PDF import and export

The current document aggregate is a strong foundation for that extension, but
those concepts should be introduced as an explicit `boq` document type rather
than overloading ordinary pricing documents with construction-specific rules.

Daftar is being positioned to support **blue-collar professionals and small
field businesses**, not only office-based finance teams. Electricians,
plumbers, builders, fabricators, painters, mechanics, installers, technicians,
artisans and independent contractors all need to turn labour and materials into
clear prices—and then preserve what was agreed.

Many of these businesses currently depend on handwritten notebooks, chat
messages or fragile spreadsheets. The name **Daftar** respects that familiar
ledger habit while giving it the reliability of server calculations,
searchable records, immutable finalization and an audit trail.

The current product already provides a useful foundation for that audience:

- Fast custom line entry without requiring a product catalogue
- Quantities, rates, discounts, taxes and automatic totals
- Draft autosave while an estimate is still being prepared
- Mobile-responsive document creation and retrieval
- Searchable customer, title and reference history
- Professional printable documents for customers or project records
- Duplication of previous jobs into fresh drafts
- Finalized records and audit history when the agreed price must be revisited
- Currency-safe reporting for the business owner

The next field-focused evolution would prioritize practical tools rather than
enterprise accounting complexity:

- Trade-friendly units and decimal measurements
- Labour, material, equipment and transport cost categories
- BOQ sections and subtotals for construction work
- Reusable job templates and frequently used line items
- Customer contact and job-site details
- Business-profile customization: uploaded logo, legal or trading name,
  supporting name/subtitle and brand accent
- Reusable business details such as address, phone, email, registration or tax
  identifiers and payment instructions
- Consistent custom branding across estimates, invoices, BOQs, printable views
  and exported PDFs, including configurable headers, footers, notes and terms
- Photo and attachment support for site evidence
- Deposits, staged payments, variations and work-completion status
- Shareable estimate links and lightweight customer approval
- Offline-first drafting for unreliable connectivity
- Localized language, terminology, currency and tax presets
- Full Arabic localization and right-to-left layouts where appropriate

The positioning is intentional: **Daftar should help skilled people present
their work professionally, price it confidently and retain a trustworthy record
of the agreement—without asking them to become accountants first.**

## Further documentation

- `README.md` — canonical assignment submission and product narrative
- `ENGINEERING_README.md` — complete engineering and operational reference
- `docs/API_SPEC.md` — REST contracts and error codes
- `docs/CALCULATIONS.md` — authoritative financial policy
- `docs/DATA_MODEL.md` — MongoDB models and indexes
- `docs/SYSTEM_DESIGN.md` — architecture and operational decisions
- `docs/brand-daftar/daftar-brand-guidelines.pdf` — bilingual identity,
  logo usage, palette and visual system
