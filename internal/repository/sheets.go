package repository

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

type SheetsClient struct {
	svc           *sheets.Service
	spreadsheetID string
	sheetName     string
}

func NewSheetsClient(ctx context.Context, credentialsPath, spreadsheetID, sheetName string) (*SheetsClient, error) {
	svc, err := sheets.NewService(ctx,
		option.WithCredentialsFile(credentialsPath),
		option.WithScopes(sheets.SpreadsheetsScope),
	)
	if err != nil {
		return nil, fmt.Errorf("sheets.NewService: %w", err)
	}
	return &SheetsClient{
		svc:           svc,
		spreadsheetID: spreadsheetID,
		sheetName:     sheetName,
	}, nil
}

// BuildRow converts an Expense into a Sheets row slice.
// Exported so it can be unit-tested without a live Sheets connection.
func BuildRow(e *model.Expense, now time.Time) []interface{} {
	timestamp := now.Format("02/01/2006 15:04:05")
	return []interface{}{
		timestamp,
		e.Date,
		e.Month,
		string(e.Category),
		e.Description,
		e.Nominal,
	}
}

// AppendExpense appends an expense row to the configured sheet.
func (c *SheetsClient) AppendExpense(ctx context.Context, e *model.Expense) error {
	row := BuildRow(e, time.Now())
	vr := &sheets.ValueRange{
		Values: [][]interface{}{row},
	}
	_, err := c.svc.Spreadsheets.Values.
		Append(c.spreadsheetID, c.sheetName, vr).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("append row: %w", err)
	}
	return nil
}

// DeleteLastRow removes the last data row from the sheet.
func (c *SheetsClient) DeleteLastRow(ctx context.Context) error {
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

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteDimension: &sheets.DeleteDimensionRequest{
					Range: &sheets.DimensionRange{
						SheetId:    sheetID,
						Dimension:  "ROWS",
						StartIndex: int64(lastRow - 1),
						EndIndex:   int64(lastRow),
					},
				},
			},
		},
	}
	_, err = c.svc.Spreadsheets.BatchUpdate(c.spreadsheetID, req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("delete row: %w", err)
	}
	return nil
}

// ReadTransactions reads all data rows from the transaction sheet.
func (c *SheetsClient) ReadTransactions(ctx context.Context) ([][]interface{}, error) {
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

// ReadMonthlyExpense reads columns A-D from Monthly Expense sheet.
func (c *SheetsClient) ReadMonthlyExpense(ctx context.Context) ([][]interface{}, error) {
	resp, err := c.svc.Spreadsheets.Values.
		Get(c.spreadsheetID, "Monthly Expense!A:D").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("read monthly expense: %w", err)
	}
	return resp.Values, nil
}

func (c *SheetsClient) getSheetID(ctx context.Context) (int64, error) {
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
