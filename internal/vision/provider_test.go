package vision_test

import (
	"testing"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
	"github.com/dikyayodihamzah/expense-tracer/internal/vision"
)

func TestParseJSON_SingleExpense(t *testing.T) {
	raw := `{"date":"14/05/2026","expenses":[{"category":"Food","description":"lunch at warung","nominal":35000}]}`
	es, err := vision.ParseJSON([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(es) != 1 {
		t.Fatalf("expected 1 expense, got %d", len(es))
	}
	e := es[0]
	if e.Date != "14/05/2026" || e.Category != model.CategoryFood || e.Nominal != 35000 || e.Description != "lunch at warung" {
		t.Errorf("unexpected expense: %+v", e)
	}
}

func TestParseJSON_MultipleCategoriesShareDate(t *testing.T) {
	raw := `{"date":"14/05/2026","expenses":[
		{"category":"Food","description":"Indomaret: peanut, milk","nominal":25000},
		{"category":"Shopping","description":"Indomaret: detergent","nominal":20000}]}`
	es, err := vision.ParseJSON([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(es) != 2 {
		t.Fatalf("expected 2 expenses, got %d", len(es))
	}
	if es[0].Category != model.CategoryFood || es[1].Category != model.CategoryShopping {
		t.Errorf("categories: %v, %v", es[0].Category, es[1].Category)
	}
	if es[0].Date != "14/05/2026" || es[1].Date != "14/05/2026" {
		t.Errorf("dates not shared: %v, %v", es[0].Date, es[1].Date)
	}
}

func TestParseJSON_DropsNonPositiveNominal(t *testing.T) {
	raw := `{"date":"14/05/2026","expenses":[{"category":"Food","description":"free sample","nominal":0},{"category":"Shopping","description":"soap","nominal":9000}]}`
	es, err := vision.ParseJSON([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(es) != 1 || es[0].Category != model.CategoryShopping {
		t.Errorf("expected only the Shopping entry, got %+v", es)
	}
}

func TestParseJSON_NoUsableExpense(t *testing.T) {
	for name, raw := range map[string]string{
		"empty list":   `{"date":"14/05/2026","expenses":[]}`,
		"zero nominal": `{"date":"14/05/2026","expenses":[{"category":"Food","description":"ERROR: empty image","nominal":0}]}`,
		"old shape":    `{"date":"14/05/2026","category":"Food","description":"lunch","nominal":35000}`,
	} {
		if _, err := vision.ParseJSON([]byte(raw)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestParseJSON_UnknownCategoryFallsToOthers(t *testing.T) {
	raw := `{"date":"14/05/2026","expenses":[{"category":"Groceries","description":"supermarket","nominal":120000}]}`
	es, err := vision.ParseJSON([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if es[0].Category != model.CategoryOthers {
		t.Errorf("expected Others, got %v", es[0].Category)
	}
}

func TestParseJSON_InvalidJSON(t *testing.T) {
	_, err := vision.ParseJSON([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseJSON_WithPreamble(t *testing.T) {
	raw := `Here is the result: {"date":"14/05/2026","expenses":[{"category":"Food","description":"lunch","nominal":35000}]}`
	es, err := vision.ParseJSON([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if es[0].Nominal != 35000 {
		t.Errorf("nominal mismatch: %v", es[0].Nominal)
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
