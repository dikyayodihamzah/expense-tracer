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
