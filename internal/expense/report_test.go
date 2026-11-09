package expense_test

import (
	"strings"
	"testing"
	"time"

	"github.com/dikyayodihamzah/expense-tracer/internal/expense"
)

func TestFormatLeft(t *testing.T) {
	// Monthly Expense rows: day, daily expense, cumulative expense, cumulative budget.
	// November has 30 days and a 3.000.000 budget, so 100.000 per day.
	var rows [][]any
	for d := 1; d <= 30; d++ {
		cum := any("")
		if d <= 9 {
			cum = float64(d * 90000)
		}
		rows = append(rows, []any{float64(d), float64(90000), cum, float64(d * 100000)})
	}
	now := time.Date(2026, 11, 9, 20, 0, 0, 0, time.UTC)
	out := expense.FormatLeft(rows, now)
	for _, want := range []string{
		"Day 9",
		"Rp 2.190.000 left of Rp 3.000.000",      // 3.000.000 - 810.000
		"Rp 99.545 per day for the next 22 days", // 2.190.000 / 22 (9th..30th)
		"Rp 90.000 under plan",                   // planned 900.000 - spent 810.000
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	rows[8][2] = float64(950000)
	if out := expense.FormatLeft(rows, now); !strings.Contains(out, "Rp 50.000 over plan") {
		t.Errorf("over plan:\n%s", out)
	}
	if out := expense.FormatLeft(nil, now); !strings.Contains(out, "No budget data") {
		t.Errorf("empty: %s", out)
	}
}

func TestFormatRecent(t *testing.T) {
	rows := [][]any{
		{"ts", "06/11/2026", "November", "Eating Out", "warteg", "16000"},
		{"ts", "07/11/2026", "November", "Date", "Kali Lereng", "62000"},
		{"ts", "08/11/2026", "November", "Transportation", "MRT", "10000"},
	}
	out := expense.FormatRecent(rows, 2)
	if strings.Contains(out, "warteg") {
		t.Errorf("should only show the last 2:\n%s", out)
	}
	for _, want := range []string{"07/11 · Date · Kali Lereng · Rp 62.000", "08/11 · Transportation · MRT · Rp 10.000"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Index(out, "MRT") > strings.Index(out, "Kali Lereng") {
		t.Errorf("newest first:\n%s", out)
	}
	if out := expense.FormatRecent(nil, 5); !strings.Contains(out, "No expenses") {
		t.Errorf("empty: %s", out)
	}
}
