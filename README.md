# Expense Tracer Bot

A Telegram bot that logs personal expenses to Google Sheets. Send a receipt photo or a text command — the bot parses the details, asks for confirmation, then appends a row to your spreadsheet.

## Features

- **Photo scanning** — send a receipt image; a vision AI extracts date, category, description, and amount
- **Manual entry** — `/add` command for quick text-based logging
- **Daily summary** — `/today` lists every expense recorded today with a total
- **Monthly summary** — `/summary` breaks spending down by category for the current month
- **Budget tracking** — `/budget` shows cumulative spend vs. budget for the current day
- **Billed account** — every expense records where it was paid from (Jago Main by default); GoPay, BRI and cash spending is added to the pocket tracker automatically

## Bot Commands

| Command | Description |
|---|---|
| `/add <description> <nominal> <category> [DD/MM/YYYY]` | Log an expense manually; omit the date for today |
| `/today` | Show today's expenses and total |
| `/summary` | Show this month's category breakdown |
| `/budget` | Show cumulative budget progress for today |
| _(send a photo)_ | Extract expense from a receipt image |

**Example:**
```
/add makan siang 35000 Eating Out
/add makan siang 35000 Eating Out 17/10/2026
/add ojek 8000 Transportation @gopay
```

After a photo or `/add`, the bot shows the details with **Save**, **Cancel**, **Edit category** and **Edit account** buttons.

**Accounts:** `@jagomain` (default) · `@gopay` · `@bri` · `@cash` · `@leisure` · `@bills` · `@education` · `@orthodental`. The bot detects the account from the screenshot when it can.

**Pocket tracker:** daily-living spending paid with GoPay, BRI or cash is added to that account's "Used" cell (`+amount` appended to the cell's formula). An expense paid from an account other than the pocket that budgets it (e.g. a bill paid with GoPay, a date paid from Jago Main) is not added; the bot replies with the transfer to make instead.

## Supported Categories

`Groceries` · `Eating Out` · `Transportation` · `Shopping` · `Personal Care` · `Date` · `Hobbies & Leisure` · `Gifts & Donation` · `Others` · `Housing` · `Recurring Bills` · `Education` · `Orthodental` · `Investment`

## Vision Providers

Set `VISION_PROVIDER` to one of the following, or leave it unset for auto-detection (first available key wins; `claude-cli` is never auto-selected):

| Value | Provider | Key required |
|---|---|---|
| `claude` | Anthropic Claude | `ANTHROPIC_API_KEY` |
| `claude-cli` | Local `claude` CLI (Claude subscription) | `claude` on `PATH`, logged in |
| `openai` | OpenAI GPT-4o | `OPENAI_API_KEY` |
| `gcloud` | Google Cloud Vision | Service account key |

## Prerequisites

- Go 1.21+
- A Telegram bot token ([BotFather](https://t.me/BotFather))
- A Google Sheets spreadsheet
- A Google Cloud service account key (JSON)
- At least one vision provider key, or the `claude` CLI logged in to a subscription (`VISION_PROVIDER=claude-cli`)

## Setup

**1. Clone and install dependencies**
```bash
git clone https://github.com/dikyayodihamzah/expense-tracer
cd expense-tracer
go mod download
```

**2. Configure environment**
```bash
cp .env.example .env
# Edit .env with your values
```

**3. Create a Google service account**

1. In Google Cloud Console, enable the **Google Sheets API** (and the **Cloud Vision API** if you use `VISION_PROVIDER=gcloud`).
2. Under **IAM & Admin → Service Accounts**, create a service account, then **Keys → Add key → JSON**. Save the file next to the project and set `GOOGLE_CREDENTIALS_FILE` to its path.
3. Share the spreadsheet with the service account's email (`...@<project>.iam.gserviceaccount.com`) as **Editor**.

**4. Spreadsheet structure**

The bot expects two sheets:

*Transaction sheet* (default name: `Transaction 2026`) — columns A–F:

| A (Timestamp) | B (Date) | C (Month) | D (Category) | E (Description) | F (Nominal) |
|---|---|---|---|---|---|

*Budget sheet* — named `Monthly Expense`, columns A–D:

| A (Day) | B (Daily Expense) | C (Cumulative Expense) | D (Cumulative Budget) |
|---|---|---|---|

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `TELEGRAM_BOT_TOKEN` | yes | — | Bot token from BotFather |
| `GOOGLE_CREDENTIALS_FILE` | yes | — | Path to the service account key JSON |
| `SPREADSHEET_ID` | yes | — | Google Sheets spreadsheet ID |
| `VISION_PROVIDER` | no | auto | `claude`, `claude-cli`, `openai`, or `gcloud` |
| `ANTHROPIC_API_KEY` | if claude | — | Anthropic API key |
| `OPENAI_API_KEY` | if openai | — | OpenAI API key |
| `SHEET_NAME` | no | `Transaction 2026` | Name of the transaction sheet |
| `POCKET_CELL_BRI` | no | `Monthly Expense!J4` | Pocket tracker cell for BRI spending |
| `POCKET_CELL_CASH` | no | `Monthly Expense!J5` | Pocket tracker cell for cash spending |
| `POCKET_CELL_GOPAY` | no | `Monthly Expense!J8` | Pocket tracker cell for GoPay spending |

## Running

```bash
# Development
make run

# Build binary
make build
./bin/bot

# Tests
make test
```

## Docker

1. For `VISION_PROVIDER=claude-cli`, run `claude setup-token` on your machine and put the result in `.env` as `CLAUDE_CODE_OAUTH_TOKEN=...`.
2. Keep `GOOGLE_CREDENTIALS_FILE` in `.env` pointing at the host key file; compose mounts it read-only into the container.
3. Start it:
   ```bash
   docker compose up -d --build
   docker compose logs -f
   ```

Only run one instance per bot token (stop the local one first), or Telegram will reject the second poller.
