package sheets

import (
	"context"
	"fmt"
	"net/http"
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

func (c *Client) DeleteLastRow(ctx context.Context) error {
	readRange := c.sheetName + "!A:A"
	resp, err := c.svc.Spreadsheets.Values.
		Get(c.spreadsheetID, readRange).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("read rows: %w", err)
	}

	lastRow := len(resp.Values)
	if lastRow <= 1 {
		return fmt.Errorf("no data rows to delete")
	}

	sheetID, err := c.getSheetID(ctx)
	if err != nil {
		return err
	}

	req := &sheetsapi.BatchUpdateSpreadsheetRequest{
		Requests: []*sheetsapi.Request{
			{
				DeleteDimension: &sheetsapi.DeleteDimensionRequest{
					Range: &sheetsapi.DimensionRange{
						SheetId:    sheetID,
						Dimension:  "ROWS",
						StartIndex: int64(lastRow - 1),
						EndIndex:   int64(lastRow),
					},
				},
			},
		},
	}
	if _, err := c.svc.Spreadsheets.BatchUpdate(c.spreadsheetID, req).Context(ctx).Do(); err != nil {
		return fmt.Errorf("delete row: %w", err)
	}
	return nil
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

func (c *Client) getSheetID(ctx context.Context) (int64, error) {
	ss, err := c.svc.Spreadsheets.Get(c.spreadsheetID).Context(ctx).Do()
	if err != nil {
		return 0, fmt.Errorf("get spreadsheet: %w", err)
	}
	for _, s := range ss.Sheets {
		if s.Properties.Title == c.sheetName {
			return s.Properties.SheetId, nil
		}
	}
	return 0, fmt.Errorf("sheet %q not found", c.sheetName)
}
