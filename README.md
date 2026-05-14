# Expense Tracer Bot

A Telegram bot that logs personal expenses to Google Sheets. Send a receipt photo or a text command — the bot parses the details, asks for confirmation, then appends a row to your spreadsheet.

## Features

- **Photo scanning** — send a receipt image; a vision AI extracts date, category, description, and amount
- **Manual entry** — `/add` command for quick text-based logging
- **Daily summary** — `/today` lists every expense recorded today with a total
- **Monthly summary** — `/summary` breaks spending down by category for the current month
- **Budget tracking** — `/budget` shows cumulative spend vs. budget for the current day
- **Undo** — `/delete` removes the last appended row

## Bot Commands

| Command | Description |
|---|---|
| `/add <description> <nominal> <category>` | Log an expense manually |
| `/today` | Show today's expenses and total |
| `/summary` | Show this month's category breakdown |
| `/budget` | Show cumulative budget progress for today |
| `/delete` | Delete the last saved expense |
| _(send a photo)_ | Extract expense from a receipt image |

**Example:**
```
/add makan siang 35000 Food
```

## Supported Categories

`Food` · `Personal Care` · `Transportation` · `Shopping` · `Entertainment and Leisure` · `Donation` · `Orthodental` · `Investment` · `Others`

## Vision Providers

Set `VISION_PROVIDER` to one of the following, or leave it unset for auto-detection (first available key wins):

| Value | Provider | Key required |
|---|---|---|
| `claude` | Anthropic Claude | `ANTHROPIC_API_KEY` |
| `openai` | OpenAI GPT-4o | `OPENAI_API_KEY` |
| `gcloud` | Google Cloud Vision | OAuth2 credentials |

## Prerequisites

- Go 1.21+
- A Telegram bot token ([BotFather](https://t.me/BotFather))
- A Google Sheets spreadsheet
- A Google Cloud OAuth2 Desktop App credentials file
- At least one vision provider key

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

**3. Authenticate Google OAuth2**

On first run the bot will print a URL. Open it in a browser, grant access, and paste the code back into the terminal. The token is saved to `token.json` and reused on subsequent runs.

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
| `GOOGLE_CREDENTIALS_FILE` | yes | — | Path to OAuth2 credentials JSON |
| `SPREADSHEET_ID` | yes | — | Google Sheets spreadsheet ID |
| `VISION_PROVIDER` | no | auto | `claude`, `openai`, or `gcloud` |
| `ANTHROPIC_API_KEY` | if claude | — | Anthropic API key |
| `OPENAI_API_KEY` | if openai | — | OpenAI API key |
| `GOOGLE_TOKEN_FILE` | no | `token.json` | Path to save/load OAuth2 token |
| `SHEET_NAME` | no | `Transaction 2026` | Name of the transaction sheet |

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
