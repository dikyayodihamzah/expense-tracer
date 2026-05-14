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

func TestAllCategories(t *testing.T) {
	cats := model.AllCategories()
	if len(cats) == 0 {
		t.Fatal("expected non-empty category list")
	}
	found := false
	for _, c := range cats {
		if c == model.CategoryFood {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Food in AllCategories")
	}
}
