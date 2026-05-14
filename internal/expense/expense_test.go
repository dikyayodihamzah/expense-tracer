package expense_test

import (
	"testing"
	"time"

	"github.com/dikyayodihamzah/expense-tracer/internal/expense"
	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

func TestParseAddCommand_AllArgs(t *testing.T) {
	e, err := expense.ParseAddCommand("makan siang 35000 Food")
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
	_, err := expense.ParseAddCommand("makan siang")
	if err == nil {
		t.Error("expected error for missing args")
	}
}

func TestParseAddCommand_InvalidNominal(t *testing.T) {
	_, err := expense.ParseAddCommand("makan abc Food")
	if err == nil {
		t.Error("expected error for non-numeric nominal")
	}
}

func TestFormatBudget_MatchesToday(t *testing.T) {
	rows := [][]any{
		{"Day", "Daily", "Cumulative", "Budget"},
		{"13", "50,000", "600,000", "650,000"},
		{"14", "35,000", "635,000", "700,000"},
		{"15", "0", "635,000", "750,000"},
	}
	now := time.Date(2026, 5, 14, 0, 0, 0, 0, time.Local)
	out := expense.FormatBudget(rows, now)
	if !containsStr(out, "Day 14") {
		t.Errorf("expected Day 14, got: %s", out)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && (s[:len(sub)] == sub || containsStr(s[1:], sub)))
}

func TestFormatSummary_GroupsByCategory(t *testing.T) {
	rows := [][]any{
		{"ts", "14/05/2026", "May", "Food", "lunch", "35000"},
		{"ts", "14/05/2026", "May", "Food", "dinner", "50000"},
		{"ts", "14/05/2026", "May", "Transportation", "grab", "25000"},
	}
	out := expense.FormatSummary(rows, "May")
	if out == "" {
		t.Error("expected non-empty summary")
	}
}

func TestFormatSummary_Empty(t *testing.T) {
	out := expense.FormatSummary([][]any{}, "May")
	if !containsStr(out, "No expenses") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestFormatSummary_WrongMonth(t *testing.T) {
	rows := [][]any{
		{"ts", "14/04/2026", "April", "Food", "lunch", "35000"},
		{"ts", "14/04/2026"},
	}
	out := expense.FormatSummary(rows, "May")
	if !containsStr(out, "No expenses") {
		t.Errorf("expected empty message for wrong month, got: %s", out)
	}
}

func TestMonthFromDate_ValidDate(t *testing.T) {
	got, err := expense.MonthFromDate("14/05/2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "May" {
		t.Errorf("expected May, got %v", got)
	}
}

func TestFillExpenseMeta_SetsMonthFromDate(t *testing.T) {
	e := &model.Expense{Date: "14/05/2026"}
	err := expense.FillExpenseMeta(e, time.Date(2026, 5, 14, 0, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Month != "May" {
		t.Errorf("expected May, got %v", e.Month)
	}
}

func TestFillExpenseMeta_AutoSetsDate(t *testing.T) {
	e := &model.Expense{}
	now := time.Date(2026, 5, 14, 0, 0, 0, 0, time.Local)
	if err := expense.FillExpenseMeta(e, now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Date != "14/05/2026" {
		t.Errorf("expected 14/05/2026, got %v", e.Date)
	}
}

func TestFillExpenseMeta_InvalidDate(t *testing.T) {
	e := &model.Expense{Date: "not-a-date"}
	if err := expense.FillExpenseMeta(e, time.Now()); err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestMonthFromDate_InvalidDate(t *testing.T) {
	_, err := expense.MonthFromDate("bad-date")
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestFormatConfirmation(t *testing.T) {
	e := &model.Expense{
		Date:        "14/05/2026",
		Category:    model.CategoryFood,
		Description: "lunch",
		Nominal:     35000,
	}
	out := expense.FormatConfirmation(e)
	if !containsStr(out, "14/05/2026") {
		t.Errorf("expected date in output, got: %s", out)
	}
	if !containsStr(out, "35.000") {
		t.Errorf("expected formatted nominal in output, got: %s", out)
	}
}

func TestFormatToday_WithData(t *testing.T) {
	rows := [][]any{
		{"ts", "14/05/2026", "May", "Food", "lunch", "35000"},
		{"ts", "14/05/2026", "May", "Food", "dinner", "50000"},
		{"ts", "13/05/2026", "May", "Food", "breakfast", "20000"},
	}
	out := expense.FormatToday(rows, "14/05/2026")
	if !containsStr(out, "lunch") {
		t.Errorf("expected lunch in output, got: %s", out)
	}
	if !containsStr(out, "85.000") {
		t.Errorf("expected total 85.000 in output, got: %s", out)
	}
}

func TestFormatToday_Empty(t *testing.T) {
	rows := [][]any{
		{"ts", "13/05/2026", "May", "Food", "lunch", "35000"},
	}
	out := expense.FormatToday(rows, "14/05/2026")
	if !containsStr(out, "No expenses") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestFormatBudget_EmptyRows(t *testing.T) {
	out := expense.FormatBudget([][]any{}, time.Now())
	if !containsStr(out, "No budget data") {
		t.Errorf("expected no data message, got: %s", out)
	}
}

func TestFormatBudget_NoMatchingDay(t *testing.T) {
	rows := [][]any{
		{"Day", "Daily", "Cumulative", "Budget"},
		{"1", "10,000", "10,000", "50,000"},
	}
	now := time.Date(2026, 5, 15, 0, 0, 0, 0, time.Local)
	out := expense.FormatBudget(rows, now)
	if !containsStr(out, "No budget data found") {
		t.Errorf("expected no match message, got: %s", out)
	}
}

func TestFormatBudget_ShortRow(t *testing.T) {
	rows := [][]any{
		{"Day", "Daily"},
		{"15", "10,000"},
	}
	now := time.Date(2026, 5, 15, 0, 0, 0, 0, time.Local)
	out := expense.FormatBudget(rows, now)
	if !containsStr(out, "No budget data found") {
		t.Errorf("expected no match message for short row, got: %s", out)
	}
}
