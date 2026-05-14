package repository_test

import (
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
