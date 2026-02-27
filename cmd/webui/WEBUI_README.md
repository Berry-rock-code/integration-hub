# Web UI (Go + embedded static assets)

This folder contains the **static web UI** (HTML/CSS/JS) for the Automation Hub.

The UI is intentionally “simple but useful” for non-technical users:
- Drag & drop file upload (or file picker)
- A clear “jobs” list (what was processed recently)
- A details panel (what the system extracted / produced)

The goal is to provide a **boss-friendly interface** for workflows that would otherwise require a terminal.

---

## How it works (high level)

### 1) Browser UI (this folder)
- `index.html` renders the page layout
- `static/styles.css` provides the styling
- `static/app.js` handles:
  - file selection / drag-drop
  - POSTing the file to the Go server
  - fetching job history and job details from the API
  - updating the page with results

### 2) Go server (cmd/webui)
The server is the “controller”:
- Serves the static UI assets from this folder using Go's `embed`
- Exposes a minimal API used by `app.js`:
  - `POST /api/upload` — accepts a file (multipart form) and runs a processing step
  - `GET  /api/jobs` — returns a list of recent jobs
  - `GET  /api/job/{id}` — returns details for one job

### 3) Processing step (today: demo)
Right now, the processing step is a placeholder:
- reads the file
- computes size + checksum
- stores a small text preview

This is meant to be replaced with the real workflows (OCR, CSV parsing, reconciliations, etc.).

---

## Folder layout

Typical structure once embedded under `cmd/webui`:

```
cmd/webui/
  main.go
  web/
    index.html
    static/
      app.js
      styles.css
```

> Note: Go's `//go:embed` patterns are **relative to the package directory**, so we keep this `web/`
> folder alongside `cmd/webui/main.go` to make embedding straightforward.

---

## Running locally

From repo root:

### Git Bash / macOS / Linux
- `go run ./cmd/webui`

### PowerShell
- `go run .\cmd\webui`

Then open:
- http://localhost:8080

---

## “Linking it later” (how this becomes a real product)

The UI is already wired to call backend API endpoints. The plan is:

### Phase 1 — Single-process MVP (now)
Everything happens in one Go process:
- upload file → process → show results

Good for:
- demos
- early workflows
- proving UX and value quickly

### Phase 2 — Connect to shared integration clients
The Go server will call reusable packages under `internal/`:
- `internal/salesforce` — query/update Salesforce records
- `internal/sheets` / `internal/drive` — read/write Google Workspace data
- `internal/buildium` — pull/push Buildium data

This converts the UI into an “operations console”:
- “Upload file and reconcile”
- “Pull latest rent roll and export”
- “Update Salesforce and mirror to Sheets”

### Phase 3 — Background jobs + persistence (production-ready)
For workflows that take longer or need durability:
- upload file goes to Google Cloud Storage (GCS)
- job metadata goes to Firestore / Cloud SQL
- job processing runs asynchronously via Pub/Sub + worker

UI changes are minimal:
- it already polls job lists and job details

### Phase 4 — Authentication & auditing
Before exposing beyond a trusted internal network:
- add login (Google identity, IAP, or simple auth)
- store a permanent audit trail for writes:
  - what changed
  - when
  - who triggered it
  - success/failure

---

## API contract (current)

### POST /api/upload
- Request: `multipart/form-data` with field `file`
- Response: `{ "id": "<job-id>" }`

### GET /api/jobs
- Response: `[]Job`

### GET /api/job/{id}
- Response: `Job`

---

## Notes on future extensions (recommended direction)

### Keep the UI dumb; keep the server smart
- UI should only display and collect inputs
- Server should handle business logic, validation, and integration calls

### Keep the integration layer reusable
When linking to Salesforce/Sheets/Buildium:
- do NOT add vendor logic into `app.js`
- call server endpoints
- server uses `internal/<vendor>` packages

---

## Troubleshooting

### “pattern web/*: no matching files found”
This happens when the `web/` folder is not located where the Go embed expects it.

If `cmd/webui/main.go` contains:
```go
//go:embed web/*
var webFS embed.FS
```

Then the `web/` folder must be located at:
- `cmd/webui/web/`

---

## What to edit first

- `static/app.js` — UI behavior, fetching endpoints, rendering output
- `main.go` — add new API endpoints and connect to real processing logic
- `processFile()` — replace demo processing with real pipeline steps

