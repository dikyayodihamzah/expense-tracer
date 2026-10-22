package vision

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/oauth2"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

// Provider extracts one expense per category from an image.
type Provider interface {
	ExtractExpenses(ctx context.Context, imageData []byte) ([]*model.Expense, error)
}

type visionItem struct {
	Category    string `json:"category"`
	Description string `json:"description"`
	Nominal     int64  `json:"nominal"`
}

type visionResponse struct {
	Date     string       `json:"date"`
	Expenses []visionItem `json:"expenses"`
}

// ParseJSON parses the JSON returned by any vision provider into one Expense per
// category group, all sharing the image's date. Entries without a positive nominal
// are dropped; an error is returned when none remain.
// Exported for testing without a live API call.
func ParseJSON(data []byte) ([]*model.Expense, error) {
	s := strings.TrimSpace(string(data))
	if idx := strings.Index(s, "{"); idx > 0 {
		s = s[idx:]
	}
	if idx := strings.LastIndex(s, "}"); idx >= 0 {
		s = s[:idx+1]
	}

	var vr visionResponse
	if err := json.Unmarshal([]byte(s), &vr); err != nil {
		return nil, fmt.Errorf("parse vision JSON: %w", err)
	}

	var expenses []*model.Expense
	for _, it := range vr.Expenses {
		if it.Nominal <= 0 {
			continue
		}
		expenses = append(expenses, &model.Expense{
			Date:        vr.Date,
			Category:    model.MatchCategory(it.Category),
			Description: it.Description,
			Nominal:     it.Nominal,
		})
	}
	if len(expenses) == 0 {
		return nil, fmt.Errorf("no expense with a positive amount found in vision response")
	}
	return expenses, nil
}

// prompt is the shared prompt sent to every vision provider.
const prompt = `You are an expense extraction assistant.
Analyze this receipt or bank transaction image and extract the expense details.
Return ONLY a valid JSON object with these exact keys:
{
  "date": "DD/MM/YYYY, the transaction date printed on the image, never today's date",
  "expenses": [
    {
      "category": "one of: Food, Personal Care, Transportation, Shopping, Entertainment and Leisure, Donation, Orthodental, Investment, Others",
      "description": "merchant name plus the items in this group, short",
      "nominal": <integer amount in IDR, no currency symbol>
    }
  ]
}
Grouping rules: if the items on the image belong to different categories, return one entry per category
with that category's items summed. Example: a supermarket receipt with peanuts, milk and detergent becomes
Food (peanuts and milk) and Shopping (detergent). If everything belongs to one category, return a single entry.
Date rules: convert whatever format is printed (e.g. "17 Okt 2026", "17-10-26", "2026-10-17 18:42") to DD/MM/YYYY.
The day always comes before the month. Always output a 4-digit year; a 2-digit year means 20YY.
If no transaction date is visible, use an empty string for "date".
Nominal rules: record only the money actually charged, and make the entries' nominal values add up to it.
Spread tax, service charge, shipping and discounts across the entries in proportion to their items.
On a GoPay invoice, ignore the part paid with GoPay Coins / GoPay points ("poin") and use only the amount
charged to the GoPay balance ("saldo GoPay") or to another real payment source.
Do not include any text outside the JSON object.`

// New creates a Provider based on the VISION_PROVIDER env value.
// If VISION_PROVIDER is unset, the provider is auto-detected from whichever
// key is present. Returns an error if no provider key is available.
func New(provider, anthropicKey, openaiKey string, googleTS oauth2.TokenSource) (Provider, error) {
	p := strings.ToLower(provider)

	if p == "" {
		switch {
		case anthropicKey != "":
			p = "claude"
		case openaiKey != "":
			p = "openai"
		case googleTS != nil:
			p = "gcloud"
		default:
			return nil, fmt.Errorf("at least one vision provider key must be set: ANTHROPIC_API_KEY, OPENAI_API_KEY, or GOOGLE_CREDENTIALS_FILE")
		}
	}

	switch p {
	case "claude":
		return newClaude(anthropicKey)
	case "openai":
		return newOpenAI(openaiKey)
	case "gcloud":
		return newGCloud(googleTS)
	case "claude-cli":
		return newClaudeCLI()
	default:
		return nil, fmt.Errorf("unknown VISION_PROVIDER: %q (valid: claude, claude-cli, openai, gcloud)", provider)
	}
}
