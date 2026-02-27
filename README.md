# Integration & Automation Hub (Go)

This repository is the foundation for a **company-wide automation platform** that connects and coordinates:

- **Salesforce** (sales + lifecycle events)
- **Google Workspace** (Drive / Sheets as operational data stores)
- **Buildium** (property management source-of-truth for leases, charges, etc.)
- **GCP** (deployment, scheduling, secrets, logging)

It is designed for a **small/medium real estate + property management company** with:
- limited engineering bandwidth (single SWE)
- low budget
- high operational reliance on SaaS systems

The core strategy is: **standardize integrations once, reuse everywhere**.

---

## Why this repo exists

Historically, “automation projects” tend to grow as isolated scripts/apps. That creates long-term problems:

- Integration code is duplicated (auth, retries, pagination, rate limits)
- Behavior becomes inconsistent across projects
- Fixes must be applied in multiple places
- Credentials end up scattered
- Adding new automations gets slower over time

This repo solves that by establishing a **single Go monorepo** with:

1. A shared **integration library layer** (Salesforce / Google / Buildium clients)
2. Multiple small automation apps (jobs/services) that **import** the shared layer
3. A consistent “production-ready” baseline: config, tests, logging patterns, deployment shape

---

## Goals

### Primary goals
- **One integration layer** for each vendor system (Salesforce, Google, Buildium)
- **Reusable business primitives** (e.g., `Property`, `Lease`, `Opportunity`) where beneficial
- Make it easy to add new automations as **small apps** without rewriting plumbing
- Encourage reliability:
  - consistent retries/backoff (where appropriate)
  - consistent error handling
  - consistent logging/auditing hooks
  - safe batching for APIs (Sheets, Salesforce)
- Keep cost and ops overhead low (Cloud Run + Scheduler + Pub/Sub when needed)

### Secondary goals
- Provide a clean path for OCR/AI integration:
  - Go orchestrates and calls managed services (Vision / Document AI), or
  - Go calls a separate model service endpoint (if we later need custom ML)

---

## Non-goals (by design)

- This is **not** intended to be a single “mega-service” that does everything.
- This is **not** intended to be a framework.
- We avoid premature abstraction—integrations should remain **thin** and practical.

---

## Architectural approach (plain English)

Think of this repo as:

- **Toolbox (internal/...)**: “How to talk to systems”
- **Jobs/Services (cmd/...)**: “What we want to do with that data”

When Buildium changes an API detail, or we add better retry behavior, we update the toolbox once and every job benefits.

---

## Repository layout

```
brh-automation/
  go.mod
  cmd/                      # runnable apps (one per automation)
    demo/                   # example app proving the wiring
  internal/                 # shared packages (integration clients, helpers)
    buildium/               # Buildium client wrapper (thin)
    httpx/                  # shared HTTP client defaults
```

As the repo grows, expect:

```
internal/
  salesforce/
  sheets/
  drive/
  ocr/
  config/
  logging/
cmd/
  sf_to_sheets_sync/
  nightly_reconciliation/
  buildium_export_to_drive/
  ...
```

---

## Development workflow

### Prerequisites
- Go installed and working:
  - `go version`

### Common commands
- Format everything:
  - `gofmt -w .`
- Run tests:
  - `go test ./...`
- Run a specific app:
  - Git Bash / macOS / Linux: `go run ./cmd/demo`
  - PowerShell: `go run .\cmd\demo`

### Current “smoke test”
- `internal/buildium` includes a small `Ping()` method and a unit test using `httptest`.
- `cmd/demo` imports `internal/buildium` and calls it successfully.

This proves the core concept:
> A new “project” can be added under `cmd/` and reuse shared integration code under `internal/`.

---

## Configuration & secrets

### Local development (simple)
Use environment variables.

Example pattern (will expand as integrations are added):

- `BUILDIUM_BASE_URL`
- `BUILDIUM_CLIENT_ID`
- `BUILDIUM_CLIENT_SECRET`

### Production (GCP)
Use **Secret Manager** and inject secrets as environment variables at deploy time.

Principle:
- **no credentials in code**
- **no credentials in git**
- secrets live in **one place**

---

## How to add a new automation (“project”)

1. Create a new folder under `cmd/`:
   - `cmd/<automation_name>/main.go`

2. Import the shared clients you need:
   - `internal/buildium`
   - `internal/salesforce`
   - `internal/sheets`

3. Keep business logic in the app.
   - The app decides *what to do*.
   - The shared clients decide *how to talk to vendors*.

**Rule of thumb**
- `internal/<vendor>`: auth, request building, pagination, retry primitives, typed responses
- `cmd/<job>`: orchestration, decisions, workflow steps, domain-specific mapping, audit logging

---

## How to add a new integration client

Add a new package under `internal/`:

- `internal/salesforce`
- `internal/sheets`
- `internal/drive`
- `internal/ocr`

Keep it thin:
- a `Client` type
- a constructor
- a few methods for the endpoints we use
- tests using `httptest` wherever possible

---

## Testing philosophy

We aim for:
- fast unit tests
- minimal reliance on real external systems
- “offline” tests using `httptest` for HTTP integrations

Why:
- reliable CI
- no surprise API costs
- no flakiness due to vendor downtime
- no accidental production writes

---

## Deployment philosophy (GCP)

This repo is designed to map cleanly to low-overhead GCP services:

### Preferred baseline
- **Cloud Run** for HTTP services and workers
- **Cloud Scheduler** for recurring jobs
- **Pub/Sub** for decoupling and buffering (when needed)
- **Cloud Logging + Error Reporting** for observability
- **Secret Manager** for credentials

This avoids GKE and heavy ops overhead unless we truly need it.

---

## OCR / AI positioning

Go can orchestrate OCR/AI work in three ways:

1. **Managed OCR** (recommended for MVP):
   - Call Google Vision / Document AI from Go
2. **Local OCR**:
   - Use something like Tesseract via a Go wrapper (best for printed docs)
3. **Custom ML models**:
   - Train elsewhere, export model (e.g., ONNX), and run inference in Go, or
   - Host model on Vertex AI / a model server and call it from Go

Key point:
> Go remains the “integration backbone.” AI is a component we plug in when needed.

---

## Roadmap (suggested)

### Phase 1: Foundation (now)
- shared HTTP defaults (`internal/httpx`)
- Buildium client skeleton + tests
- a demo app proving reuse (`cmd/demo`)

### Phase 2: Core integrations
- Salesforce client wrapper
- Google Sheets wrapper
- Common config + structured logging

### Phase 3: First production automations
- “Salesforce stage change → update Google Sheet”
- “Nightly reconciliation job”
- “Buildium → Google Drive/Sheets export”

### Phase 4: Quality / ops
- CI checks (format, tests)
- deployment scripts / Cloud Run pipeline
- audit trail for important writes (who/what/when)

---

## Conventions

- Prefer small, readable packages over clever abstraction.
- Prefer “boring” reliability patterns: retries, timeouts, idempotency.
- `gofmt` is non-negotiable (format is standardized).
- Keep credentials out of git—always.

---

## Quick start (today)

From repo root:

1. Run tests:
   - `go test ./...`

2. Run the demo:
   - Git Bash: `go run ./cmd/demo`

You should see:

> `demo OK (cmd/demo is calling internal/buildium successfully)`

---

## Notes for non-engineering stakeholders

This repo is an investment in **operational reliability** and **lower long-term cost**.

Instead of building many disconnected scripts, we build one consistent integration layer.
That makes it faster to add automations, safer to change them, and easier to audit outcomes.

