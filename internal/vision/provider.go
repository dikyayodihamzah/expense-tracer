package vision

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/oauth2"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

// Provider extracts expense fields from an image.
type Provider interface {
	ExtractExpense(ctx context.Context, imageData []byte) (*model.Expense, error)
}

type visionResponse struct {
	Date        string `json:"date"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Nominal     int64  `json:"nominal"`
}

// ParseJSON parses the JSON returned by any vision provider.
// Exported for testing without a live API call.
func ParseJSON(data []byte) (*model.Expense, error) {
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

	return &model.Expense{
		Date:        vr.Date,
		Category:    model.MatchCategory(vr.Category),
		Description: vr.Description,
		Nominal:     vr.Nominal,
	}, nil
}

// prompt is the shared prompt sent to every vision provider.
const prompt = `You are an expense extraction assistant.
Analyze this receipt or bank transaction image and extract the expense details.
Return ONLY a valid JSON object with these exact keys:
{
  "date": "DD/MM/YYYY",
  "category": "one of: Food, Personal Care, Transportation, Shopping, Entertainment and Leisure, Donation, Orthodental, Investment, Others",
  "description": "merchant name or short description",
  "nominal": <integer amount in IDR, no currency symbol>
}
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
	default:
		return nil, fmt.Errorf("unknown VISION_PROVIDER: %q (valid: claude, openai, gcloud)", provider)
	}
}
