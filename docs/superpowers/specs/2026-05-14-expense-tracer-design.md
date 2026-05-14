# Expense Tracer — Design Spec

**Date:** 2026-05-14  
**Status:** Approved

---

## Overview

A Go backend service that acts as a personal expense tracker. Users submit expenses via a Telegram bot — either by sending a photo of a receipt/bank transaction screenshot or via structured commands. The backend parses the input, confirms with the user, and appends the record directly to a Google Spreadsheet.

**Scope:** Single user, personal use. Stateless — no local database. All state lives in Google Sheets.

---

## Architecture

Single long-running Go binary. Two concurrent goroutines: Telegram polling loop and a Fiber HTTP server. Two outbound integrations: Google Sheets API and an AI vision provider.

```
Telegram (polling)         HTTP client (health check)
       │                          │
       ▼                          ▼
┌──────────────────────────────────────────┐
│              Go Backend                  │
│                                          │
│  [goroutine 1] Telegram polling loop     │
│  [goroutine 2] Fiber HTTP server         │
│                GET /health               │
│                                          │
│  controller/ → service/ → repository/   │
│                                          │
│  repository/vision.go  ──▶  Vision API  │
│  repository/sheets.go  ──▶  Google Sheets API v4
└──────────────────────────────────────────┘
```

**Layered architecture:**

```
expense-tracer/
├── cmd/
│   └── bot/
│       └── main.go              # entry point, dependency wiring, goroutine management
├── internal/
│   ├── controller/
│   │   ├── telegram.go          # routes Telegram updates, calls service
│   │   └── health.go            # GET /health handler (Fiber)
│   ├── service/
│   │   └── expense.go           # business logic: parse, validate, format
│   ├── repository/
│   │   ├── sheets.go            # read/write Google Sheets
│   │   └── vision.go            # vision provider (env-switched)
│   └── model/
│       └── expense.go           # ExpenseFields, Category constants
├── .env.example
└── Makefile                     # make run, make build
```

---

## Telegram Bot Interface

### Image Input Flow

1. User sends a photo (receipt or bank transaction screenshot)
2. Bot downloads the image and forwards it to the configured vision provider
3. Vision API extracts: Date, Category, Description, Nominal
4. Bot replies with a confirmation message showing parsed fields
5. User confirms via inline button (✓ Save / ✗ Cancel) or edits fields before saving
6. On confirm → append row to Transaction 2026

If parsing fails or fields are missing, the bot asks the user to fill in missing fields interactively before saving.

### Commands

| Command | Description |
|---|---|
| `/add <description> <nominal> <category>` | Manual quick-add; prompts for any missing fields |
| `/delete` | Removes the last appended row from Transaction 2026 |
| `/today` | Lists all expenses recorded today with running total |
| `/summary` | Monthly spending breakdown by category (current month) |
| `/budget` | Shows monthly budget progress from the Monthly Expense sheet |

---

## Data Model

### Expense Fields

Matches the existing Google Form exactly:

| Field | Type | Notes |
|---|---|---|
| Timestamp | string | Server-generated, format: `DD/MM/YYYY HH:MM:SS` |
| Date | string | Format: `DD/MM/YYYY` |
| Month | string | Full name: January, February, … December |
| Category | string | Fixed list (see below) |
| Description | string | Free text, merchant name or note |
| Nominal | int64 | Integer, Indonesian Rupiah (IDR) |

### Categories

`Food`, `Personal Care`, `Transportation`, `Shopping`, `Entertainment and Leisure`, `Donation`, `Orthodental`, `Investment`, `Others`

If vision extraction returns an unrecognized category, it defaults to `Others`.

---

## Google Sheets Integration

**Target spreadsheet:** `1RHkpTQjCED28_gRc4MbGDQOI09QRgc913l2xD7aXPHI`

### Write — Append Row (`/add`, image confirm)

- Target sheet: `Transaction 2026`
- Uses `spreadsheets.values.append` with `valueInputOption=USER_ENTERED`
- Row order: `Timestamp | Date | Month | Category | Description | Nominal`
- Does NOT go through Google Forms (Forms submission API is undocumented and fragile); writes directly to the sheet with identical results

### Read — `/today` and `/summary`

- Reads all rows from `Transaction 2026` via `spreadsheets.values.get`
- Filters client-side by date or month
- Avoids complex Sheets query syntax

### Read — `/budget`

- Reads `Monthly Expense` sheet: Day, Daily Expense, Cumulative Expense, Cumulative Budget columns
- Monthly Expense and Budgeting sheets are formula-driven; the backend never writes to them

### Delete — `/delete`

- Reads the full `Transaction 2026` column range to find the last row index
- Deletes it using `spreadsheets.batchUpdate` → `deleteDimension`
- Row 1 (header) is protected — cannot be deleted

### Auth

- Google Sheets API via **OAuth2 service account**
- Credentials JSON file path provided via `GOOGLE_APPLICATION_CREDENTIALS` env var
- Same credentials reused for Google Cloud Vision if that provider is selected

---

## Vision Provider

### Interface

```go
type VisionProvider interface {
    ExtractExpense(ctx context.Context, imageData []byte) (*model.ExpenseFields, error)
}
```

### Implementations

| Provider | Env value | Key env var |
|---|---|---|
| Anthropic Claude | `claude` (default) | `ANTHROPIC_API_KEY` |
| OpenAI GPT-4 Vision | `openai` | `OPENAI_API_KEY` |
| Google Cloud Vision | `gcloud` | `GOOGLE_APPLICATION_CREDENTIALS` |

### Prompt Strategy

Each provider receives the image plus a structured prompt instructing it to return JSON with the four expense fields. The extracted Category is matched against the fixed list; unrecognized values fall back to `Others`.

---

## Configuration

All configuration via environment variables:

```env
TELEGRAM_BOT_TOKEN=...

# Vision provider
VISION_PROVIDER=claude             # claude | openai | gcloud

# Provider API keys (only the selected one is required)
ANTHROPIC_API_KEY=...
OPENAI_API_KEY=...

# Google (service account — shared by Sheets and GCloud Vision)
GOOGLE_APPLICATION_CREDENTIALS=path/to/service-account.json

# Spreadsheet
SPREADSHEET_ID=1RHkpTQjCED28_gRc4MbGDQOI09QRgc913l2xD7aXPHI
SHEET_NAME=Transaction 2026

# HTTP server (Fiber)
PORT=8080
```

---

## Deployment

- Local machine, runs as a long-running process
- Telegram polling (no webhook, no public IP required)
- Fiber HTTP server runs on `PORT` (default `8080`) for health checks: `GET /health → 200 OK`
- Recommended: run via `systemd` user service or `screen`/`tmux` session
- Build: `make build` → single binary, no external runtime dependencies

---

## Out of Scope

- Multi-user support
- Local database / caching layer
- Budget editing via bot (Budgeting sheet is read-only from the bot's perspective)
- Investment record management (Recap 2026 sheet)
- Google Forms submission (bypassed in favor of direct Sheets API writes)
