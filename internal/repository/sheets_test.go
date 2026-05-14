package repository_test

import (
	"testing"
	"time"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
	"github.com/dikyayodihamzah/expense-tracer/internal/repository"
)

func TestBuildRow_FieldOrder(t *testing.T) {
	e := &model.Expense{
		Date:        "14/05/2026",
		Month:       "May",
		Category:    model.CategoryFood,
		Description: "lunch",
		Nominal:     50000,
	}
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, time.Local)
	row := repository.BuildRow(e, now)

	if len(row) != 6 {
		t.Fatalf("expected 6 columns, got %d", len(row))
	}
	if row[1] != "14/05/2026" {
		t.Errorf("Date column mismatch: %v", row[1])
	}
	if row[2] != "May" {
		t.Errorf("Month column mismatch: %v", row[2])
	}
	if row[3] != "Food" {
		t.Errorf("Category column mismatch: %v", row[3])
	}
	if row[4] != "lunch" {
		t.Errorf("Description column mismatch: %v", row[4])
	}
	if row[5] != int64(50000) {
		t.Errorf("Nominal column mismatch: %v", row[5])
	}
}
