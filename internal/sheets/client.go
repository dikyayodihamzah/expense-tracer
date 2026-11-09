package sheets

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/option"
	sheetsapi "google.golang.org/api/sheets/v4"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

type Client struct {
	svc           *sheetsapi.Service
	spreadsheetID string
	sheetName     string
}

func New(ctx context.Context, httpClient *http.Client, spreadsheetID, sheetName string) (*Client, error) {
	svc, err := sheetsapi.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("sheets.NewService: %w", err)
	}
	return &Client{
		svc:           svc,
		spreadsheetID: spreadsheetID,
		sheetName:     sheetName,
	}, nil
}

// BuildRow converts an Expense into a Sheets row slice.
// Exported so it can be unit-tested without a live Sheets connection.
func BuildRow(e *model.Expense, now time.Time) []any {
	timestamp := now.Format("02/01/2006 15:04:05")
	return []any{
		timestamp,
		e.Date,
		e.Month,
		string(e.Category),
		e.Description,
		e.Nominal,
	}
}

// AppendExpense writes the row directly under the last non-empty cell of column A.
// values.append is avoided on purpose: when the sheet holds a Google Sheets table,
// it anchors to the table's declared end instead of the last data row, so after the
// table is full every new entry lands on the same row.
func (c *Client) AppendExpense(ctx context.Context, e *model.Expense) error {
	resp, err := c.svc.Spreadsheets.Values.
		Get(c.spreadsheetID, c.sheetName+"!A:A").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("find next row: %w", err)
	}

	row := BuildRow(e, time.Now())
	vr := &sheetsapi.ValueRange{
		Values: [][]any{row},
	}
	if _, err := c.svc.Spreadsheets.Values.
		Update(c.spreadsheetID, NextRowRange(c.sheetName, len(resp.Values), len(row)), vr).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do(); err != nil {
		return fmt.Errorf("write row: %w", err)
	}
	return nil
}

// NextRowRange returns the A1 range for the row after usedRows rows, spanning cols columns (max 26).
// Exported for unit testing without a live Sheets connection.
func NextRowRange(sheetName string, usedRows, cols int) string {
	next := usedRows + 1
	last := string(rune('A' + cols - 1))
	return fmt.Sprintf("%s!A%d:%s%d", sheetName, next, last, next)
}

// AddToCell adds amount to a typed-in total such as a pocket tracker "Used" cell
// (cell in A1 notation, e.g. "Monthly Expense!J8"). It appends "+amount" to the
// cell's formula instead of overwriting the total, so every addition stays visible
// and can be corrected by hand.
func (c *Client) AddToCell(ctx context.Context, cell string, amount int64) error {
	resp, err := c.svc.Spreadsheets.Values.
		Get(c.spreadsheetID, cell).
		ValueRenderOption("FORMULA").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("read %s: %w", cell, err)
	}
	var cur any
	if len(resp.Values) > 0 && len(resp.Values[0]) > 0 {
		cur = resp.Values[0][0]
	}
	vr := &sheetsapi.ValueRange{Values: [][]any{{AppendAmount(cur, amount)}}}
	if _, err := c.svc.Spreadsheets.Values.
		Update(c.spreadsheetID, cell, vr).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do(); err != nil {
		return fmt.Errorf("write %s: %w", cell, err)
	}
	return nil
}

// AppendAmount returns the formula for cur plus amount: "=8000" for an empty cell,
// "=74479+8000" for a plain number and "=74479+3000+8000" for a formula.
// Exported for unit testing without a live Sheets connection.
func AppendAmount(cur any, amount int64) string {
	switch v := cur.(type) {
	case float64:
		// FORMULA rendering returns plain numbers as float64; avoid exponent notation.
		return fmt.Sprintf("=%s+%d", strconv.FormatFloat(v, 'f', -1, 64), amount)
	case string:
		v = strings.TrimSpace(v)
		switch {
		case v == "":
		case strings.HasPrefix(v, "="):
			return fmt.Sprintf("%s+%d", v, amount)
		default:
			return fmt.Sprintf("=%s+%d", v, amount)
		}
	}
	return fmt.Sprintf("=%d", amount)
}

func (c *Client) ReadTransactions(ctx context.Context) ([][]any, error) {
	readRange := c.sheetName + "!A2:F"
	resp, err := c.svc.Spreadsheets.Values.
		Get(c.spreadsheetID, readRange).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("read transactions: %w", err)
	}
	return resp.Values, nil
}

func (c *Client) ReadMonthlyExpense(ctx context.Context) ([][]any, error) {
	resp, err := c.svc.Spreadsheets.Values.
		Get(c.spreadsheetID, "Monthly Expense!A:D").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("read monthly expense: %w", err)
	}
	return resp.Values, nil
}
