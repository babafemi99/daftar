# DAFTAR (دَفْتَر) - PRODUCT REQUIREMENTS DOCUMENT

**Version:** 1.0  
**Date:** 8 August 2026  
**Product:** Daftar  
**Assignment:** Multi-Rate Pricing Calculator

## Product identity

**Daftar (دَفْتَر)** is the sole product name used throughout the repository, interface, API metadata, PDFs, and deployment.

Daftar is a multi-rate financial document system for small businesses and finance teams. Users create quote- or invoice-like documents with custom line items, apply a fixed or percentage discount and a tax rate per line, review server-calculated totals, and finalize the document into an immutable financial record. Users can then download a PDF and report on finalized documents over a date range.

### Name and regional positioning

`Daftar` means notebook, ledger, or official register across Arabic-influenced business cultures in the Middle East and South Asia. The name connects the traditional account book with a modern, auditable financial workflow and gives the product a natural MENA identity.

The interface uses **Daftar** as the primary Latin-script product name and may display **دَفْتَر** as a supporting brand mark. Arabic-language localization and right-to-left layout are future enhancements, not implied by the name alone.

### Multi-rate versus multi-currency

The assignment's **multi-rate** requirement refers to different discount and tax rates on different line items. It does not require multiple currencies.

Daftar adds controlled multi-currency document support as a product enhancement:

- Each document has exactly one currency.
- Every line item inherits that currency.
- There is no currency conversion in version 1.
- Reports filter by currency or return separate totals per currency.
- Values in different currencies are never added into a misleading combined total.

The product is a focused implementation of the CrossVal take-home assignment. It demonstrates reliable financial calculations, document lifecycle management, auditability, reporting, MongoDB data modelling, and full-stack ownership.

## Goals

1. Calculate mixed line-level discounts and taxes correctly and reproducibly.
2. Make the Go backend the sole authority for all monetary calculations.
3. Provide a clear draft-to-finalized lifecycle with server-enforced immutability.
4. Give users useful document search, reporting, tax breakdown, print, and PDF workflows.
5. Demonstrate thoughtful MongoDB schema/index design behind a Next.js interface.
6. Ship a deployed, tested application with clear operational and engineering documentation.

## Non-goals

- Inventory, SKUs, warehouses, or stock movements.
- Payment collection, settlements, refunds, or accounts receivable.
- Double-entry bookkeeping or a general ledger.
- Tax filing or jurisdiction-specific tax advice.
- Currency conversion or live exchange rates.
- Bank/accounting-platform integrations.
- Sending documents by email.
- A full CRM or product catalogue.
- AI-generated calculations. Deterministic backend code is always authoritative.

## Primary user

A business owner or finance team member who needs to prepare and preserve clear financial documents without manually calculating discounts and taxes in spreadsheets.

The initial release has individual user accounts rather than multi-member organizations. Every resource belongs to exactly one authenticated user.

## Core concepts

### Financial document

A document is a business record stored in MongoDB and rendered in the web application. It is not an uploaded file. It contains metadata, custom line items, calculated totals, currency, lifecycle status, and timestamps.

### Custom line item

A user manually enters a product or service description, quantity, unit price, discount, and tax. Line items are not connected to inventory.

### Draft

A draft is editable and excluded from official financial reports. Draft PDFs and print views display a visible `DRAFT` marker.

### Finalized document

A finalized document is read-only, cannot be deleted, and is included in official reports. Finalization is a one-way transition. Reuse happens by duplicating the finalized document into a new draft.

## Functional requirements

### FR-1 Authentication and ownership

- Users can register with email and password.
- Users can log in, log out, and retrieve their current profile.
- Passwords are stored only as strong salted hashes.
- Every document, audit event, report, PDF, and idempotency record is scoped to the authenticated user.
- Cross-user access returns `404` to avoid disclosing that another user's resource exists.

### FR-2 Document creation

A user can create a draft with:

- Title
- Customer name
- Issue date
- ISO 4217 currency code from the supported set
- Zero or more custom line items while drafting; at least one is required to finalize

The backend generates a human-readable reference such as `DOC-2026-000001` and calculates every derived monetary field.

### FR-3 Line items

Each line contains:

- Description
- Quantity greater than or equal to 1
- Unit price greater than or equal to 0
- Optional discount: `fixed` or `percentage`, never both
- Optional tax percentage

The client submits inputs only. The server calculates and returns subtotal, discount amount, discounted amount, tax amount, and line total.

### FR-4 Calculations

For each line:

1. `subtotal = quantity x unit price`
2. Calculate and subtract the discount.
3. Calculate tax on the discounted amount.
4. `line total = discounted amount + tax`

At document level:

- Subtotal = sum of line subtotals
- Total discount = sum of line discount amounts
- Total tax = sum of line tax amounts
- Grand total = sum of line totals

The exact representation and rounding policy are defined in `CALCULATIONS.md`.

### FR-5 Draft editing

- Users can edit draft metadata and line items.
- Any relevant edit triggers a complete server-side recalculation.
- Users can delete drafts through a recoverable soft-delete/archive operation.
- Concurrent edits use optimistic version checks to prevent silent overwrites.

### FR-6 Finalization

- Users explicitly confirm finalization.
- The backend validates and recalculates the document before finalizing it.
- Finalization records `finalizedAt` and an audit event.
- Finalized metadata, line inputs, derived amounts, currency, and issue date are immutable.
- Finalization supports an idempotency key so retries do not create duplicate effects.
- Finalized documents cannot be archived or deleted.

### FR-7 Document duplication

- A user can duplicate a finalized document into a new draft.
- The new draft receives a new ID, reference, version, and timestamps.
- Finalization data and audit history are not copied.
- Inputs are copied and totals are recalculated using the current calculation engine.
- The original remains unchanged.

### FR-8 Document list and search

The document list supports:

- Search by title, customer, or reference
- Filter by `draft` or `finalized`
- Filter by issue-date range
- Filter by currency
- Sort by newest, oldest, or grand total
- Cursor or page-based pagination

The default view excludes archived drafts.

### FR-9 Document view

The detail view shows:

- Metadata and status
- Line-item calculations
- Document totals
- Tax breakdown by rate
- Calculation explanation per line
- Activity timeline
- Available actions based on status

### FR-10 PDF and print

- Users can print or download a server-generated PDF of their own document.
- Draft output displays a `DRAFT` marker.
- Finalized output displays the document reference and finalized date.
- The PDF is generated only from the stored, server-calculated record.
- PDF download is recorded in the audit trail without storing sensitive payloads.

### FR-11 Reporting dashboard

Users can select:

- Today
- Last 7 days
- This month
- Last month
- This quarter
- This year
- Custom issue-date range

Official reports include finalized documents only and return:

- Document count
- Subtotal
- Total discount
- Total tax
- Grand total
- Average document value
- Tax breakdown by rate
- Monthly grand-total timeline

Reports never add different currencies together. Results are grouped by currency or filtered to one currency.

### FR-12 Auditability

Audit events are created for material actions:

- Document created
- Document updated
- Document finalized
- Document duplicated
- Draft archived or restored
- PDF downloaded

Each event records actor, action, document, timestamp, and limited non-sensitive metadata. Audit records are append-only through the application.

## User flows

### Create and finalize

1. User registers or logs in.
2. User opens the dashboard and selects **New document**.
3. User enters metadata and custom line items.
4. Client sends inputs to the Go API.
5. Backend validates, calculates, saves, and returns the draft.
6. User edits and reviews the calculation explanation.
7. User confirms finalization.
8. Backend revalidates and recalculates, then atomically marks it finalized.
9. User views or downloads the immutable record.

### Duplicate

1. User opens a finalized document.
2. User selects **Duplicate**.
3. Backend creates a new draft with copied inputs and recalculated totals.
4. User edits and finalizes the new document independently.

### Reporting

1. User chooses a date preset or custom range.
2. User optionally selects a currency.
3. Backend aggregates only the user's finalized, non-archived documents.
4. Dashboard displays cards, tax breakdown, and monthly chart.

## Product decisions

- Supported initial currencies: `USD`, `AED`, `SAR`, `NGN`, `GBP`, and `EUR`.
- No FX conversion. Reports group by currency.
- Fixed discounts exceeding the line subtotal are rejected, not clamped.
- Tax and discounts cannot make a line total negative.
- Drafts are excluded from official reports.
- Finalization cannot be reversed.
- Finalized records cannot be deleted.
- Drafts use soft archive rather than destructive deletion.
- A document requires at least one valid line before finalization.
- Dates are stored as UTC instants where time is relevant; `issueDate` is treated as a calendar date.

## Success criteria

- The assignment sample returns subtotal `450.00`, discount `40.00`, tax `11.50`, and grand total `421.50`.
- No API accepts client-calculated totals as authoritative input.
- Attempts to mutate a finalized document fail with a specific error.
- A user cannot access another user's resources through any endpoint.
- Reports reconcile exactly to their included finalized documents.
- Mixed currencies are never silently combined.
- Core calculation and lifecycle tests pass in CI.
- A reviewer can register, create, finalize, download, duplicate, and report on a document in the deployed application.

## Delivery priority

### P0 - required

Authentication, ownership, document CRUD, calculations, lifecycle enforcement, REST API, summary report, tests, deployment, and README.

### P1 - differentiators

PDF/print, duplication, tax breakdown, audit trail, reference numbers, date presets, search/filter/sort, currency-safe reports, optimistic concurrency, and idempotent finalization.

### P2 - only after P0/P1 quality

A read-only AI explanation of already-calculated results. It must never calculate, edit, or approve financial data.

## Open implementation decisions

These may be settled during system design without changing product scope:

- Exact PDF library.
- Cookie-session versus short-lived access token implementation.
- Cursor versus page-number pagination.
- Hosting provider selection within the required public deployment.
