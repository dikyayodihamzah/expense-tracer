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
	rows := [][]interface{}{
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
