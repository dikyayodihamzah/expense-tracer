package sheets_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
	"github.com/dikyayodihamzah/expense-tracer/internal/sheets"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func mockClient(handler http.Handler) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			return rec.Result(), nil
		}),
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func errResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{"code": 500, "message": "internal error", "status": "INTERNAL"},
	})
}

func spreadsheetMeta(sheetTitle string) map[string]any {
	return map[string]any{
		"spreadsheetId": "test-id",
		"sheets": []any{
			map[string]any{"properties": map[string]any{"sheetId": 1, "title": sheetTitle}},
		},
	}
}

// TestBuildRow_FieldOrder verifies column ordering and timestamp format.
func TestBuildRow_FieldOrder(t *testing.T) {
	e := &model.Expense{
		Date:        "14/05/2026",
		Month:       "May",
		Category:    model.CategoryFood,
		Description: "lunch",
		Nominal:     50000,
	}
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, time.Local)
	row := sheets.BuildRow(e, now)

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

func TestNew_CreatesClient(t *testing.T) {
	c, err := sheets.New(context.Background(), mockClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})), "test-id", "Sheet1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Error("expected non-nil client")
	}
}

func TestReadTransactions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"values": []any{[]any{"ts", "14/05/2026", "May", "Food", "lunch", "35000"}},
		})
	})
	c, _ := sheets.New(context.Background(), mockClient(handler), "test-id", "Sheet1")
	rows, err := c.ReadTransactions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
}

func TestReadTransactions_Error(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { errResponse(w) })
	c, _ := sheets.New(context.Background(), mockClient(handler), "test-id", "Sheet1")
	if _, err := c.ReadTransactions(context.Background()); err == nil {
		t.Error("expected error")
	}
}

func TestReadMonthlyExpense(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"values": []any{
				[]any{"Day", "Daily", "Cumulative", "Budget"},
				[]any{"14", "35,000", "35,000", "50,000"},
			},
		})
	})
	c, _ := sheets.New(context.Background(), mockClient(handler), "test-id", "Sheet1")
	rows, err := c.ReadMonthlyExpense(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
}

func TestReadMonthlyExpense_Error(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { errResponse(w) })
	c, _ := sheets.New(context.Background(), mockClient(handler), "test-id", "Sheet1")
	if _, err := c.ReadMonthlyExpense(context.Background()); err == nil {
		t.Error("expected error")
	}
}

func TestAppendExpense(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"spreadsheetId": "test-id"})
	})
	c, _ := sheets.New(context.Background(), mockClient(handler), "test-id", "Sheet1")
	e := &model.Expense{Date: "14/05/2026", Month: "May", Category: model.CategoryFood, Description: "lunch", Nominal: 35000}
	if err := c.AppendExpense(context.Background(), e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppendExpense_Error(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { errResponse(w) })
	c, _ := sheets.New(context.Background(), mockClient(handler), "test-id", "Sheet1")
	if err := c.AppendExpense(context.Background(), &model.Expense{}); err == nil {
		t.Error("expected error")
	}
}

func deleteHandler(valuesResp, metaResp, batchResp any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, ":batchUpdate"):
			json.NewEncoder(w).Encode(batchResp)
		case strings.Contains(path, "/values/"):
			json.NewEncoder(w).Encode(valuesResp)
		default:
			json.NewEncoder(w).Encode(metaResp)
		}
	}
}

func TestDeleteLastRow(t *testing.T) {
	handler := deleteHandler(
		map[string]any{"values": []any{[]any{"header"}, []any{"data"}}},
		spreadsheetMeta("Sheet1"),
		map[string]any{"spreadsheetId": "test-id"},
	)
	c, _ := sheets.New(context.Background(), mockClient(handler), "test-id", "Sheet1")
	if err := c.DeleteLastRow(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteLastRow_NoDataRows(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"values": []any{[]any{"header"}}})
	})
	c, _ := sheets.New(context.Background(), mockClient(handler), "test-id", "Sheet1")
	if err := c.DeleteLastRow(context.Background()); err == nil {
		t.Error("expected error for no data rows")
	}
}

func TestDeleteLastRow_ReadError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { errResponse(w) })
	c, _ := sheets.New(context.Background(), mockClient(handler), "test-id", "Sheet1")
	if err := c.DeleteLastRow(context.Background()); err == nil {
		t.Error("expected error")
	}
}

func TestDeleteLastRow_GetSheetIDError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/values/") {
			writeJSON(w, map[string]any{"values": []any{[]any{"header"}, []any{"data"}}})
		} else {
			errResponse(w)
		}
	})
	c, _ := sheets.New(context.Background(), mockClient(handler), "test-id", "Sheet1")
	if err := c.DeleteLastRow(context.Background()); err == nil {
		t.Error("expected error from getSheetID")
	}
}

func TestDeleteLastRow_SheetNotFound(t *testing.T) {
	handler := deleteHandler(
		map[string]any{"values": []any{[]any{"header"}, []any{"data"}}},
		spreadsheetMeta("OtherSheet"),
		map[string]any{"spreadsheetId": "test-id"},
	)
	c, _ := sheets.New(context.Background(), mockClient(handler), "test-id", "Sheet1")
	if err := c.DeleteLastRow(context.Background()); err == nil {
		t.Error("expected error when sheet not found")
	}
}

func TestDeleteLastRow_BatchUpdateError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, ":batchUpdate"):
			errResponse(w)
		case strings.Contains(path, "/values/"):
			writeJSON(w, map[string]any{"values": []any{[]any{"header"}, []any{"data"}}})
		default:
			writeJSON(w, spreadsheetMeta("Sheet1"))
		}
	})
	c, _ := sheets.New(context.Background(), mockClient(handler), "test-id", "Sheet1")
	if err := c.DeleteLastRow(context.Background()); err == nil {
		t.Error("expected error from batchUpdate")
	}
}
