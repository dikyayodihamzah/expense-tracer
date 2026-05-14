# Expense Tracer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go Telegram bot that accepts expense input (image or commands), parses it via a swappable AI vision provider, and writes records to a Google Spreadsheet.

**Architecture:** Single stateless Go binary using Telegram long-polling. Layered: controller (Telegram routing) → service (business logic) → repository (Sheets API + vision provider). No local database — all state lives in Google Sheets.

**Tech Stack:** Go 1.22+, `go-telegram-bot-api/telegram-bot-api/v5`, `google.golang.org/api/sheets/v4`, `github.com/joho/godotenv`, Anthropic/OpenAI/Google Cloud Vision APIs.

---

## File Map

| File | Responsibility |
|---|---|
| `cmd/bot/main.go` | Entry point: load config, wire dependencies, start polling loop |
| `internal/model/expense.go` | `Expense` struct, `Category` constants, category matcher |
| `internal/repository/vision.go` | `VisionProvider` interface + `NewVisionProvider` factory |
| `internal/repository/vision_claude.go` | Claude vision implementation |
| `internal/repository/vision_openai.go` | OpenAI vision implementation |
| `internal/repository/vision_gcloud.go` | Google Cloud Vision implementation |
| `internal/repository/sheets.go` | Google Sheets client: append, read, delete row |
| `internal/service/expense.go` | Parse commands, format summaries, validate fields |
| `internal/controller/telegram.go` | Route Telegram updates to service methods |
| `.env.example` | Template for all required env vars |
| `Makefile` | `make run`, `make build`, `make test` |

---

## Task 1: Project Scaffold & Dependencies

**Files:**
- Create: `go.mod`, `go.sum`
- Create: `.env.example`
- Create: `Makefile`
- Create: `cmd/bot/main.go` (skeleton)

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/synapsis/Documents/Projects/expense-tracer
go mod init github.com/dikyayodihamzah/expense-tracer
```

Expected: `go.mod` created.

- [ ] **Step 2: Install dependencies**

```bash
go get github.com/go-telegram-bot-api/telegram-bot-api/v5
go get google.golang.org/api/sheets/v4
go get google.golang.org/api/option
go get golang.org/x/oauth2/google
go get github.com/joho/godotenv
go get github.com/anthropics/anthropic-sdk-go
go get github.com/sashabaranov/go-openai
go get cloud.google.com/go/vision/apiv1
```

Expected: `go.sum` created, all packages downloaded.

- [ ] **Step 3: Create `.env.example`**

```env
TELEGRAM_BOT_TOKEN=

# claude | openai | gcloud
VISION_PROVIDER=claude

ANTHROPIC_API_KEY=
OPENAI_API_KEY=

# Path to Google service account JSON (used for Sheets + GCloud Vision)
GOOGLE_APPLICATION_CREDENTIALS=

SPREADSHEET_ID=
SHEET_NAME=Transaction 2026
```

- [ ] **Step 4: Create `Makefile`**

```makefile
.PHONY: run build test

build:
	go build -o bin/bot ./cmd/bot

run:
	go run ./cmd/bot

test:
	go test ./...
```

- [ ] **Step 5: Create skeleton `cmd/bot/main.go`**

```go
package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	log.Println("expense-tracer starting...")
}
```

- [ ] **Step 6: Verify build**

```bash
make build
```

Expected: `bin/bot` created with no errors.

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum .env.example Makefile cmd/bot/main.go
git commit -m "feat: project scaffold and dependencies"
```

---

## Task 2: Data Model

**Files:**
- Create: `internal/model/expense.go`
- Create: `internal/model/expense_test.go`

- [ ] **Step 1: Write failing test**

Create `internal/model/expense_test.go`:

```go
package model_test

import (
	"testing"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

func TestMatchCategory_KnownCategory(t *testing.T) {
	got := model.MatchCategory("Food")
	if got != model.CategoryFood {
		t.Errorf("expected %q, got %q", model.CategoryFood, got)
	}
}

func TestMatchCategory_UnknownFallsToOthers(t *testing.T) {
	got := model.MatchCategory("random gibberish")
	if got != model.CategoryOthers {
		t.Errorf("expected %q, got %q", model.CategoryOthers, got)
	}
}

func TestMatchCategory_CaseInsensitive(t *testing.T) {
	got := model.MatchCategory("food")
	if got != model.CategoryFood {
		t.Errorf("expected %q, got %q", model.CategoryFood, got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/model/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Implement `internal/model/expense.go`**

```go
package model

import "strings"

type Category string

const (
	CategoryFood                 Category = "Food"
	CategoryPersonalCare         Category = "Personal Care"
	CategoryTransportation       Category = "Transportation"
	CategoryShopping             Category = "Shopping"
	CategoryEntertainmentLeisure Category = "Entertainment and Leisure"
	CategoryDonation             Category = "Donation"
	CategoryOrthodental          Category = "Orthodental"
	CategoryInvestment           Category = "Investment"
	CategoryOthers               Category = "Others"
)

var allCategories = []Category{
	CategoryFood,
	CategoryPersonalCare,
	CategoryTransportation,
	CategoryShopping,
	CategoryEntertainmentLeisure,
	CategoryDonation,
	CategoryOrthodental,
	CategoryInvestment,
	CategoryOthers,
}

type Expense struct {
	Date        string   // DD/MM/YYYY
	Month       string   // January, February, ...
	Category    Category
	Description string
	Nominal     int64
}

// MatchCategory matches a string to a known Category, case-insensitive.
// Returns CategoryOthers if no match found.
func MatchCategory(s string) Category {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, c := range allCategories {
		if strings.ToLower(string(c)) == lower {
			return c
		}
	}
	return CategoryOthers
}

// AllCategories returns a copy of the fixed category list.
func AllCategories() []Category {
	result := make([]Category, len(allCategories))
	copy(result, allCategories)
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/model/... -v
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/model/expense.go internal/model/expense_test.go
git commit -m "feat: expense data model and category matcher"
```

---

## Task 3: Google Sheets Repository

**Files:**
- Create: `internal/repository/sheets.go`
- Create: `internal/repository/sheets_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/repository/sheets_test.go`:

```go
package repository_test

import (
	"testing"
	"time"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
	"github.com/dikyayodihamzah/expense-tracer/internal/repository"
)

func TestBuildRow_FieldOrder(t *testing.T) {
	e := &model.Expense{
		Date:        "14/05/2026",
		Month:       "May",
		Category:    model.CategoryFood,
		Description: "lunch",
		Nominal:     50000,
	}
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, time.Local)
	row := repository.BuildRow(e, now)

	if len(row) != 6 {
		t.Fatalf("expected 6 columns, got %d", len(row))
	}
	if row[1] != "14/05/2026" {
		t.Errorf("Date column mismatch: %v", row[1])
	}
	if row[2] != "May" {
		t.Errorf("Month column mismatch: %v", row[2])
	}
	if row[3] != "Food" {
		t.Errorf("Category column mismatch: %v", row[3])
	}
	if row[4] != "lunch" {
		t.Errorf("Description column mismatch: %v", row[4])
	}
	if row[5] != int64(50000) {
		t.Errorf("Nominal column mismatch: %v", row[5])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/repository/... -v -run TestBuildRow
```

Expected: FAIL — package not found.

- [ ] **Step 3: Implement `internal/repository/sheets.go`**

```go
package repository

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

type SheetsClient struct {
	svc           *sheets.Service
	spreadsheetID string
	sheetName     string
}

func NewSheetsClient(ctx context.Context, credentialsPath, spreadsheetID, sheetName string) (*SheetsClient, error) {
	svc, err := sheets.NewService(ctx,
		option.WithCredentialsFile(credentialsPath),
		option.WithScopes(sheets.SpreadsheetsScope),
	)
	if err != nil {
		return nil, fmt.Errorf("sheets.NewService: %w", err)
	}
	return &SheetsClient{
		svc:           svc,
		spreadsheetID: spreadsheetID,
		sheetName:     sheetName,
	}, nil
}

// BuildRow converts an Expense into a Sheets row slice.
// Exported so it can be unit-tested without a live Sheets connection.
func BuildRow(e *model.Expense, now time.Time) []any {
	timestamp := now.Format("02/01/2006 15:04:05")
	return []any{
		timestamp,
		e.Date,
		e.Month,
		string(e.Category),
		e.Description,
		e.Nominal,
	}
}

// AppendExpense appends an expense row to the configured sheet.
func (c *SheetsClient) AppendExpense(ctx context.Context, e *model.Expense) error {
	row := BuildRow(e, time.Now())
	vr := &sheets.ValueRange{
		Values: [][]any{row},
	}
	_, err := c.svc.Spreadsheets.Values.
		Append(c.spreadsheetID, c.sheetName, vr).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("append row: %w", err)
	}
	return nil
}

// DeleteLastRow removes the last data row from the sheet.
func (c *SheetsClient) DeleteLastRow(ctx context.Context) error {
	readRange := c.sheetName + "!A:A"
	resp, err := c.svc.Spreadsheets.Values.
		Get(c.spreadsheetID, readRange).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("read rows: %w", err)
	}

	lastRow := len(resp.Values)
	if lastRow <= 1 {
		return fmt.Errorf("no data rows to delete")
	}

	sheetID, err := c.getSheetID(ctx)
	if err != nil {
		return err
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteDimension: &sheets.DeleteDimensionRequest{
					Range: &sheets.DimensionRange{
						SheetId:    sheetID,
						Dimension:  "ROWS",
						StartIndex: int64(lastRow - 1),
						EndIndex:   int64(lastRow),
					},
				},
			},
		},
	}
	_, err = c.svc.Spreadsheets.BatchUpdate(c.spreadsheetID, req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("delete row: %w", err)
	}
	return nil
}

// ReadTransactions reads all data rows from the transaction sheet.
func (c *SheetsClient) ReadTransactions(ctx context.Context) ([][]any, error) {
	readRange := c.sheetName + "!A2:F"
	resp, err := c.svc.Spreadsheets.Values.
		Get(c.spreadsheetID, readRange).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("read transactions: %w", err)
	}
	return resp.Values, nil
}

// ReadMonthlyExpense reads columns A–D from Monthly Expense sheet.
func (c *SheetsClient) ReadMonthlyExpense(ctx context.Context) ([][]any, error) {
	resp, err := c.svc.Spreadsheets.Values.
		Get(c.spreadsheetID, "Monthly Expense!A:D").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("read monthly expense: %w", err)
	}
	return resp.Values, nil
}

func (c *SheetsClient) getSheetID(ctx context.Context) (int64, error) {
	ss, err := c.svc.Spreadsheets.Get(c.spreadsheetID).Context(ctx).Do()
	if err != nil {
		return 0, fmt.Errorf("get spreadsheet: %w", err)
	}
	for _, s := range ss.Sheets {
		if s.Properties.Title == c.sheetName {
			return s.Properties.SheetId, nil
		}
	}
	return 0, fmt.Errorf("sheet %q not found", c.sheetName)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/repository/... -v -run TestBuildRow
```

Expected: PASS.

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add internal/repository/sheets.go internal/repository/sheets_test.go
git commit -m "feat: google sheets repository"
```

---

## Task 4: Vision Provider Interface & Claude Implementation

**Files:**
- Create: `internal/repository/vision.go`
- Create: `internal/repository/vision_claude.go`
- Create: `internal/repository/vision_test.go`

- [ ] **Step 1: Write failing test for category extraction**

Create `internal/repository/vision_test.go`:

```go
package repository_test

import (
	"encoding/json"
	"testing"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
	"github.com/dikyayodihamzah/expense-tracer/internal/repository"
)

func TestParseVisionJSON_ValidResponse(t *testing.T) {
	raw := `{"date":"14/05/2026","category":"Food","description":"lunch at warung","nominal":35000}`
	e, err := repository.ParseVisionJSON([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Date != "14/05/2026" {
		t.Errorf("date mismatch: %v", e.Date)
	}
	if e.Category != model.CategoryFood {
		t.Errorf("category mismatch: %v", e.Category)
	}
	if e.Nominal != 35000 {
		t.Errorf("nominal mismatch: %v", e.Nominal)
	}
}

func TestParseVisionJSON_UnknownCategoryFallsToOthers(t *testing.T) {
	raw := `{"date":"14/05/2026","category":"Groceries","description":"supermarket","nominal":120000}`
	e, err := repository.ParseVisionJSON([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Category != model.CategoryOthers {
		t.Errorf("expected Others, got %v", e.Category)
	}
}

func TestParseVisionJSON_InvalidJSON(t *testing.T) {
	_, err := repository.ParseVisionJSON([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/repository/... -v -run TestParseVisionJSON
```

Expected: FAIL — `ParseVisionJSON` not defined.

- [ ] **Step 3: Create `internal/repository/vision.go`**

```go
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

// VisionProvider extracts expense fields from an image.
type VisionProvider interface {
	ExtractExpense(ctx context.Context, imageData []byte) (*model.Expense, error)
}

type visionResponse struct {
	Date        string `json:"date"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Nominal     int64  `json:"nominal"`
}

// ParseVisionJSON parses the JSON returned by any vision provider.
// Exported for testing without a live API call.
func ParseVisionJSON(data []byte) (*model.Expense, error) {
	// Strip markdown code fences if present (some models wrap JSON in ```json ... ```)
	s := strings.TrimSpace(string(data))
	if idx := strings.Index(s, "{"); idx > 0 {
		s = s[idx:]
	}
	if idx := strings.LastIndex(s, "}"); idx >= 0 && idx < len(s)-1 {
		s = s[:idx+1]
	}

	var vr visionResponse
	if err := json.Unmarshal([]byte(s), &vr); err != nil {
		return nil, fmt.Errorf("parse vision JSON: %w", err)
	}

	return &model.Expense{
		Date:        vr.Date,
		Category:    model.MatchCategory(vr.Category),
		Description: vr.Description,
		Nominal:     vr.Nominal,
	}, nil
}

// visionPrompt is the shared prompt sent to every vision provider.
const visionPrompt = `You are an expense extraction assistant.
Analyze this receipt or bank transaction image and extract the expense details.
Return ONLY a valid JSON object with these exact keys:
{
  "date": "DD/MM/YYYY",
  "category": "one of: Food, Personal Care, Transportation, Shopping, Entertainment and Leisure, Donation, Orthodental, Investment, Others",
  "description": "merchant name or short description",
  "nominal": <integer amount in IDR, no currency symbol>
}
Do not include any text outside the JSON object.`

// NewVisionProvider creates a VisionProvider based on the VISION_PROVIDER env value.
func NewVisionProvider(provider, anthropicKey, openaiKey, googleCredentials string) (VisionProvider, error) {
	switch strings.ToLower(provider) {
	case "claude", "":
		return newClaudeVision(anthropicKey)
	case "openai":
		return newOpenAIVision(openaiKey)
	case "gcloud":
		return newGCloudVision(googleCredentials)
	default:
		return nil, fmt.Errorf("unknown VISION_PROVIDER: %q (valid: claude, openai, gcloud)", provider)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/repository/... -v -run TestParseVisionJSON
```

Expected: all 3 PASS.

- [ ] **Step 5: Create `internal/repository/vision_claude.go`**

```go
package repository

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

type claudeVision struct {
	client *anthropic.Client
}

func newClaudeVision(apiKey string) (*claudeVision, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is required for claude vision provider")
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &claudeVision{client: client}, nil
}

func (c *claudeVision) ExtractExpense(ctx context.Context, imageData []byte) (*model.Expense, error) {
	encoded := base64.StdEncoding.EncodeToString(imageData)

	msg, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.ModelClaude3_5SonnetLatest),
		MaxTokens: anthropic.F(int64(512)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewImageBlockBase64("image/jpeg", encoded),
				anthropic.NewTextBlock(visionPrompt),
			),
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("claude vision: %w", err)
	}

	if len(msg.Content) == 0 {
		return nil, fmt.Errorf("claude vision: empty response")
	}
	text := msg.Content[0].Text

	return ParseVisionJSON([]byte(text))
}
```

- [ ] **Step 6: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add internal/repository/vision.go internal/repository/vision_claude.go internal/repository/vision_test.go
git commit -m "feat: vision provider interface and Claude implementation"
```

---

## Task 5: OpenAI & Google Cloud Vision Implementations

**Files:**
- Create: `internal/repository/vision_openai.go`
- Create: `internal/repository/vision_gcloud.go`

- [ ] **Step 1: Create `internal/repository/vision_openai.go`**

```go
package repository

import (
	"context"
	"encoding/base64"
	"fmt"

	openai "github.com/sashabaranov/go-openai"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

type openaiVision struct {
	client *openai.Client
}

func newOpenAIVision(apiKey string) (*openaiVision, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required for openai vision provider")
	}
	return &openaiVision{client: openai.NewClient(apiKey)}, nil
}

func (o *openaiVision) ExtractExpense(ctx context.Context, imageData []byte) (*model.Expense, error) {
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:image/jpeg;base64," + encoded

	resp, err := o.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL: dataURL,
						},
					},
					{
						Type: openai.ChatMessagePartTypeText,
						Text: visionPrompt,
					},
				},
			},
		},
		MaxTokens: 512,
	})
	if err != nil {
		return nil, fmt.Errorf("openai vision: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai vision: empty response")
	}

	return ParseVisionJSON([]byte(resp.Choices[0].Message.Content))
}
```

- [ ] **Step 2: Create `internal/repository/vision_gcloud.go`**

```go
package repository

import (
	"context"
	"fmt"

	vision "cloud.google.com/go/vision/apiv1"
	"cloud.google.com/go/vision/apiv1/visionpb"
	"google.golang.org/api/option"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

type gcloudVision struct {
	credentialsPath string
}

func newGCloudVision(credentialsPath string) (*gcloudVision, error) {
	if credentialsPath == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS is required for gcloud vision provider")
	}
	return &gcloudVision{credentialsPath: credentialsPath}, nil
}

func (g *gcloudVision) ExtractExpense(ctx context.Context, imageData []byte) (*model.Expense, error) {
	client, err := vision.NewImageAnnotatorClient(ctx, option.WithCredentialsFile(g.credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("gcloud vision client: %w", err)
	}
	defer client.Close()

	image := &visionpb.Image{Content: imageData}
	resp, err := client.DetectDocumentText(ctx, image, nil)
	if err != nil {
		return nil, fmt.Errorf("gcloud detect text: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("gcloud vision: no text detected")
	}

	// GCloud Vision returns raw OCR text; we need a second LLM pass to extract
	// structured fields. Return a partially filled expense prompting the user
	// to confirm/edit fields manually.
	return &model.Expense{
		Description: resp.Text,
		Category:    model.CategoryOthers,
	}, nil
}
```

> **Note:** Google Cloud Vision returns raw OCR text, not structured JSON. The `gcloud` provider extracts the full text and returns it as the description so the user can confirm/correct fields via the Telegram confirmation flow. For best results use `claude` or `openai`.

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/repository/vision_openai.go internal/repository/vision_gcloud.go
git commit -m "feat: openai and google cloud vision implementations"
```

---

## Task 6: Expense Service

**Files:**
- Create: `internal/service/expense.go`
- Create: `internal/service/expense_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/service/expense_test.go`:

```go
package service_test

import (
	"testing"
	"time"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
	"github.com/dikyayodihamzah/expense-tracer/internal/service"
)

func TestParseAddCommand_AllArgs(t *testing.T) {
	e, err := service.ParseAddCommand("makan siang 35000 Food")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Description != "makan siang" {
		t.Errorf("description: %v", e.Description)
	}
	if e.Nominal != 35000 {
		t.Errorf("nominal: %v", e.Nominal)
	}
	if e.Category != model.CategoryFood {
		t.Errorf("category: %v", e.Category)
	}
}

func TestParseAddCommand_MissingArgs(t *testing.T) {
	_, err := service.ParseAddCommand("makan siang")
	if err == nil {
		t.Error("expected error for missing args")
	}
}

func TestFormatSummary_GroupsByCategory(t *testing.T) {
	rows := [][]any{
		{"ts", "14/05/2026", "May", "Food", "lunch", "35000"},
		{"ts", "14/05/2026", "May", "Food", "dinner", "50000"},
		{"ts", "14/05/2026", "May", "Transportation", "grab", "25000"},
	}
	month := "May"
	out := service.FormatSummary(rows, month)
	if out == "" {
		t.Error("expected non-empty summary")
	}
}

func TestMonthFromDate_ValidDate(t *testing.T) {
	got, err := service.MonthFromDate("14/05/2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "May" {
		t.Errorf("expected May, got %v", got)
	}
}

func TestFillExpenseMeta_SetsMonthFromDate(t *testing.T) {
	e := &model.Expense{Date: "14/05/2026"}
	err := service.FillExpenseMeta(e, time.Date(2026, 5, 14, 0, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Month != "May" {
		t.Errorf("expected May, got %v", e.Month)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/service/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Implement `internal/service/expense.go`**

```go
package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

// ParseAddCommand parses "/add <description> <nominal> <category>" args.
// args is the text after "/add ".
func ParseAddCommand(args string) (*model.Expense, error) {
	parts := strings.Fields(args)
	if len(parts) < 3 {
		return nil, fmt.Errorf("usage: /add <description> <nominal> <category>")
	}

	// Last part is category, second-to-last is nominal, rest is description.
	categoryStr := parts[len(parts)-1]
	nominalStr := parts[len(parts)-2]
	description := strings.Join(parts[:len(parts)-2], " ")

	nominal, err := strconv.ParseInt(nominalStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("nominal must be a number, got %q", nominalStr)
	}

	return &model.Expense{
		Description: description,
		Nominal:     nominal,
		Category:    model.MatchCategory(categoryStr),
	}, nil
}

// FillExpenseMeta sets Date (if empty) and Month from the given time.
func FillExpenseMeta(e *model.Expense, now time.Time) error {
	if e.Date == "" {
		e.Date = now.Format("02/01/2006")
	}
	month, err := MonthFromDate(e.Date)
	if err != nil {
		return err
	}
	e.Month = month
	return nil
}

// MonthFromDate returns the full month name from a DD/MM/YYYY date string.
func MonthFromDate(date string) (string, error) {
	t, err := time.Parse("02/01/2006", date)
	if err != nil {
		return "", fmt.Errorf("invalid date %q (expected DD/MM/YYYY): %w", date, err)
	}
	return t.Format("January"), nil
}

// FormatConfirmation formats a parsed expense for user confirmation.
func FormatConfirmation(e *model.Expense) string {
	return fmt.Sprintf(
		"📋 *Expense Details*\n\n📅 Date: %s\n🗂 Category: %s\n📝 Description: %s\n💰 Nominal: Rp %s\n\nSave this expense?",
		e.Date,
		string(e.Category),
		e.Description,
		formatNominal(e.Nominal),
	)
}

// FormatToday formats today's expenses as a message.
func FormatToday(rows [][]any, today string) string {
	var lines []string
	var total int64

	for _, row := range rows {
		if len(row) < 6 {
			continue
		}
		if row[1] != today {
			continue
		}
		desc, _ := row[4].(string)
		nomStr, _ := row[5].(string)
		nom, _ := strconv.ParseInt(nomStr, 10, 64)
		total += nom
		lines = append(lines, fmt.Sprintf("• %s — Rp %s", desc, formatNominal(nom)))
	}

	if len(lines) == 0 {
		return "No expenses recorded today."
	}
	return fmt.Sprintf("*Today (%s)*\n\n%s\n\n*Total: Rp %s*", today, strings.Join(lines, "\n"), formatNominal(total))
}

// FormatSummary formats a monthly category breakdown.
func FormatSummary(rows [][]any, month string) string {
	totals := make(map[string]int64)
	var grandTotal int64

	for _, row := range rows {
		if len(row) < 6 {
			continue
		}
		if row[2] != month {
			continue
		}
		cat, _ := row[3].(string)
		nomStr, _ := row[5].(string)
		nom, _ := strconv.ParseInt(nomStr, 10, 64)
		totals[cat] += nom
		grandTotal += nom
	}

	if len(totals) == 0 {
		return fmt.Sprintf("No expenses found for %s.", month)
	}

	var lines []string
	for _, c := range model.AllCategories() {
		if v, ok := totals[string(c)]; ok {
			lines = append(lines, fmt.Sprintf("• %s: Rp %s", c, formatNominal(v)))
		}
	}
	return fmt.Sprintf("*%s Summary*\n\n%s\n\n*Total: Rp %s*", month, strings.Join(lines, "\n"), formatNominal(grandTotal))
}

// FormatBudget formats the monthly budget progress.
func FormatBudget(rows [][]any) string {
	if len(rows) < 2 {
		return "No budget data available."
	}

	var lastDay, dailyExp, cumExp, cumBudget string
	for _, row := range rows[1:] { // skip header
		if len(row) < 4 {
			continue
		}
		d, _ := row[0].(string)
		if d != "" {
			lastDay = d
			dailyExp, _ = row[1].(string)
			cumExp, _ = row[2].(string)
			cumBudget, _ = row[3].(string)
		}
	}

	return fmt.Sprintf(
		"*Budget Progress (Day %s)*\n\n📆 Today's Expense: %s\n📊 Cumulative Expense: %s\n🎯 Cumulative Budget: %s",
		lastDay, dailyExp, cumExp, cumBudget,
	)
}

func formatNominal(n int64) string {
	s := strconv.FormatInt(n, 10)
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/service/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/expense.go internal/service/expense_test.go
git commit -m "feat: expense service with command parsing and formatting"
```

---

## Task 7: Telegram Controller

**Files:**
- Create: `internal/controller/telegram.go`

- [ ] **Step 1: Create `internal/controller/telegram.go`**

```go
package controller

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
	"github.com/dikyayodihamzah/expense-tracer/internal/repository"
	"github.com/dikyayodihamzah/expense-tracer/internal/service"
)

type TelegramController struct {
	bot    *tgbotapi.BotAPI
	sheets *repository.SheetsClient
	vision repository.VisionProvider

	// pendingExpense holds a parsed expense awaiting user confirmation.
	// Key: chatID, Value: *model.Expense
	pendingExpense map[int64]*model.Expense
}

func NewTelegramController(bot *tgbotapi.BotAPI, sheets *repository.SheetsClient, vision repository.VisionProvider) *TelegramController {
	return &TelegramController{
		bot:            bot,
		sheets:         sheets,
		vision:         vision,
		pendingExpense: make(map[int64]*model.Expense),
	}
}

func (tc *TelegramController) HandleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		tc.handleCallback(ctx, update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}

	msg := update.Message
	chatID := msg.Chat.ID

	if msg.Photo != nil {
		tc.handlePhoto(ctx, msg)
		return
	}

	if !msg.IsCommand() {
		return
	}

	switch msg.Command() {
	case "add":
		tc.handleAdd(ctx, chatID, msg.CommandArguments())
	case "delete":
		tc.handleDelete(ctx, chatID)
	case "today":
		tc.handleToday(ctx, chatID)
	case "summary":
		tc.handleSummary(ctx, chatID)
	case "budget":
		tc.handleBudget(ctx, chatID)
	}
}

func (tc *TelegramController) handlePhoto(ctx context.Context, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	tc.reply(chatID, "⏳ Analyzing image...")

	// Get the largest photo size.
	photos := msg.Photo
	largest := photos[len(photos)-1]

	fileURL, err := tc.bot.GetFileDirectURL(largest.FileID)
	if err != nil {
		tc.reply(chatID, "❌ Failed to get image: "+err.Error())
		return
	}

	imageData, err := downloadFile(fileURL)
	if err != nil {
		tc.reply(chatID, "❌ Failed to download image: "+err.Error())
		return
	}

	expense, err := tc.vision.ExtractExpense(ctx, imageData)
	if err != nil {
		tc.reply(chatID, "❌ Failed to analyze image: "+err.Error())
		return
	}

	if err := service.FillExpenseMeta(expense, time.Now()); err != nil {
		tc.reply(chatID, "❌ Invalid date in parsed expense: "+err.Error())
		return
	}

	tc.pendingExpense[chatID] = expense
	tc.sendConfirmation(chatID, expense)
}

func (tc *TelegramController) handleAdd(ctx context.Context, chatID int64, args string) {
	if strings.TrimSpace(args) == "" {
		tc.reply(chatID, "Usage: /add <description> <nominal> <category>\nExample: /add makan siang 35000 Food")
		return
	}

	expense, err := service.ParseAddCommand(args)
	if err != nil {
		tc.reply(chatID, "❌ "+err.Error())
		return
	}

	if err := service.FillExpenseMeta(expense, time.Now()); err != nil {
		tc.reply(chatID, "❌ "+err.Error())
		return
	}

	tc.pendingExpense[chatID] = expense
	tc.sendConfirmation(chatID, expense)
}

func (tc *TelegramController) handleDelete(ctx context.Context, chatID int64) {
	if err := tc.sheets.DeleteLastRow(ctx); err != nil {
		tc.reply(chatID, "❌ Failed to delete: "+err.Error())
		return
	}
	tc.reply(chatID, "✅ Last expense deleted.")
}

func (tc *TelegramController) handleToday(ctx context.Context, chatID int64) {
	rows, err := tc.sheets.ReadTransactions(ctx)
	if err != nil {
		tc.reply(chatID, "❌ Failed to read transactions: "+err.Error())
		return
	}
	today := time.Now().Format("02/01/2006")
	tc.replyMarkdown(chatID, service.FormatToday(rows, today))
}

func (tc *TelegramController) handleSummary(ctx context.Context, chatID int64) {
	rows, err := tc.sheets.ReadTransactions(ctx)
	if err != nil {
		tc.reply(chatID, "❌ Failed to read transactions: "+err.Error())
		return
	}
	month := time.Now().Format("January")
	tc.replyMarkdown(chatID, service.FormatSummary(rows, month))
}

func (tc *TelegramController) handleBudget(ctx context.Context, chatID int64) {
	rows, err := tc.sheets.ReadMonthlyExpense(ctx)
	if err != nil {
		tc.reply(chatID, "❌ Failed to read budget: "+err.Error())
		return
	}
	tc.replyMarkdown(chatID, service.FormatBudget(rows))
}

func (tc *TelegramController) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	chatID := cb.Message.Chat.ID
	tc.bot.Request(tgbotapi.NewCallback(cb.ID, ""))

	expense, ok := tc.pendingExpense[chatID]
	if !ok {
		tc.reply(chatID, "No pending expense found. Please submit again.")
		return
	}

	switch cb.Data {
	case "confirm":
		if err := tc.sheets.AppendExpense(ctx, expense); err != nil {
			tc.reply(chatID, "❌ Failed to save: "+err.Error())
			return
		}
		delete(tc.pendingExpense, chatID)
		tc.reply(chatID, fmt.Sprintf("✅ Saved: %s — Rp %d", expense.Description, expense.Nominal))

	case "cancel":
		delete(tc.pendingExpense, chatID)
		tc.reply(chatID, "❌ Cancelled.")
	}
}

func (tc *TelegramController) sendConfirmation(chatID int64, e *model.Expense) {
	text := service.FormatConfirmation(e)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✓ Save", "confirm"),
			tgbotapi.NewInlineKeyboardButtonData("✗ Cancel", "cancel"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = keyboard
	tc.bot.Send(msg)
}

func (tc *TelegramController) reply(chatID int64, text string) {
	tc.bot.Send(tgbotapi.NewMessage(chatID, text))
}

func (tc *TelegramController) replyMarkdown(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	tc.bot.Send(msg)
}

func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// Run starts the Telegram long-polling loop. Blocks until ctx is cancelled.
func (tc *TelegramController) Run(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := tc.bot.GetUpdatesChan(u)

	log.Println("bot is running, waiting for updates...")
	for {
		select {
		case <-ctx.Done():
			tc.bot.StopReceivingUpdates()
			return
		case update := <-updates:
			go tc.HandleUpdate(ctx, update)
		}
	}
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/controller/telegram.go
git commit -m "feat: telegram controller with all command handlers"
```

---

## Task 8: Wire Everything in main.go

**Files:**
- Modify: `cmd/bot/main.go`

- [ ] **Step 1: Replace skeleton `cmd/bot/main.go` with wired implementation**

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"github.com/dikyayodihamzah/expense-tracer/internal/controller"
	"github.com/dikyayodihamzah/expense-tracer/internal/repository"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	token := mustEnv("TELEGRAM_BOT_TOKEN")
	visionProvider := os.Getenv("VISION_PROVIDER")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")
	googleCreds := mustEnv("GOOGLE_APPLICATION_CREDENTIALS")
	spreadsheetID := mustEnv("SPREADSHEET_ID")
	sheetName := os.Getenv("SHEET_NAME")
	if sheetName == "" {
		sheetName = "Transaction 2026"
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}
	log.Printf("authorized as @%s", bot.Self.UserName)

	sheetsClient, err := repository.NewSheetsClient(ctx, googleCreds, spreadsheetID, sheetName)
	if err != nil {
		log.Fatalf("failed to create sheets client: %v", err)
	}

	vision, err := repository.NewVisionProvider(visionProvider, anthropicKey, openaiKey, googleCreds)
	if err != nil {
		log.Fatalf("failed to create vision provider: %v", err)
	}

	ctrl := controller.NewTelegramController(bot, sheetsClient, vision)
	ctrl.Run(ctx)

	log.Println("bot stopped")
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s environment variable is required", key)
	}
	return v
}
```

- [ ] **Step 2: Verify build**

```bash
make build
```

Expected: `bin/bot` built successfully, no errors.

- [ ] **Step 3: Verify tests still pass**

```bash
make test
```

Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/bot/main.go
git commit -m "feat: wire all components in main.go"
```

---

## Task 9: Smoke Test & Final Config

**Files:**
- Create: `.env` (from `.env.example`, not committed)
- Modify: `.gitignore`

- [ ] **Step 1: Create `.gitignore`**

```gitignore
.env
bin/
*.json
```

- [ ] **Step 2: Copy `.env.example` to `.env` and fill in real values**

```bash
cp .env.example .env
```

Edit `.env` with your actual values:
- `TELEGRAM_BOT_TOKEN` — from @BotFather on Telegram
- `ANTHROPIC_API_KEY` — from console.anthropic.com
- `GOOGLE_APPLICATION_CREDENTIALS` — path to downloaded service account JSON
- `SPREADSHEET_ID=1RHkpTQjCED28_gRc4MbGDQOI09QRgc913l2xD7aXPHI`

- [ ] **Step 3: Enable Google Sheets API for the service account**

In Google Cloud Console:
1. Enable the **Google Sheets API** for your project
2. Share the spreadsheet (`1RHkpTQjCED28_gRc4MbGDQOI09QRgc913l2xD7aXPHI`) with the service account email (Editor permission)

- [ ] **Step 4: Run the bot**

```bash
make run
```

Expected output:
```
no .env file found, reading from environment  (or skipped if .env exists)
authorized as @<your_bot_name>
bot is running, waiting for updates...
```

- [ ] **Step 5: Test each command in Telegram**

Send to your bot:
- `/add makan siang 35000 Food` → confirmation message with ✓ Save / ✗ Cancel buttons → tap Save → check spreadsheet row added
- Send a receipt photo → confirmation message → Save
- `/today` → today's expense list
- `/summary` → monthly breakdown
- `/budget` → budget progress
- `/delete` → last row removed from spreadsheet

- [ ] **Step 6: Commit `.gitignore`**

```bash
git add .gitignore
git commit -m "chore: add gitignore"
```
