package vision_test

import (
	"testing"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
	"github.com/dikyayodihamzah/expense-tracer/internal/vision"
)

func TestParseJSON_ValidResponse(t *testing.T) {
	raw := `{"date":"14/05/2026","category":"Food","description":"lunch at warung","nominal":35000}`
	e, err := vision.ParseJSON([]byte(raw))
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

func TestParseJSON_UnknownCategoryFallsToOthers(t *testing.T) {
	raw := `{"date":"14/05/2026","category":"Groceries","description":"supermarket","nominal":120000}`
	e, err := vision.ParseJSON([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Category != model.CategoryOthers {
		t.Errorf("expected Others, got %v", e.Category)
	}
}

func TestParseJSON_InvalidJSON(t *testing.T) {
	_, err := vision.ParseJSON([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseJSON_WithPreamble(t *testing.T) {
	raw := `Here is the result: {"date":"14/05/2026","category":"Food","description":"lunch","nominal":35000}`
	e, err := vision.ParseJSON([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Nominal != 35000 {
		t.Errorf("nominal mismatch: %v", e.Nominal)
	}
}

func TestNew_NoKeys(t *testing.T) {
	_, err := vision.New("", "", "", nil)
	if err == nil {
		t.Error("expected error when no provider keys are set")
	}
}

func TestNew_UnknownProvider(t *testing.T) {
	_, err := vision.New("unknown", "key", "", nil)
	if err == nil {
		t.Error("expected error for unknown provider name")
	}
}

func TestNew_ClaudeEmptyKey(t *testing.T) {
	_, err := vision.New("claude", "", "", nil)
	if err == nil {
		t.Error("expected error for claude with empty API key")
	}
}

func TestNew_ClaudeValidKey(t *testing.T) {
	p, err := vision.New("claude", "sk-test-key", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Error("expected non-nil provider")
	}
}

func TestNew_OpenAIEmptyKey(t *testing.T) {
	_, err := vision.New("openai", "", "", nil)
	if err == nil {
		t.Error("expected error for openai with empty API key")
	}
}

func TestNew_OpenAIValidKey(t *testing.T) {
	p, err := vision.New("openai", "", "sk-test-key", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Error("expected non-nil provider")
	}
}

func TestNew_GCloudNilTokenSource(t *testing.T) {
	_, err := vision.New("gcloud", "", "", nil)
	if err == nil {
		t.Error("expected error for gcloud with nil token source")
	}
}

func TestNew_AutoDetectClaude(t *testing.T) {
	p, err := vision.New("", "sk-test-key", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Error("expected non-nil provider")
	}
}

func TestNew_AutoDetectOpenAI(t *testing.T) {
	p, err := vision.New("", "", "sk-test-key", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Error("expected non-nil provider")
	}
}
