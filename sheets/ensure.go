package sheets

import (
	"context"
	"fmt"

	"google.golang.org/api/sheets/v4"
)

// EnsureSheet ensures a tab exists. If it doesn't, it creates it.
func (c *Client) EnsureSheet(ctx context.Context, sheetTitle string) error {
	ss, err := c.svc.Spreadsheets.Get(c.SpreadsheetID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("EnsureSheet get spreadsheet: %w", err)
	}

	for _, sh := range ss.Sheets {
		if sh.Properties != nil && sh.Properties.Title == sheetTitle {
			return nil
		}
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{
						Title: sheetTitle,
					},
				},
			},
		},
	}
	_, err = c.svc.Spreadsheets.BatchUpdate(c.SpreadsheetID, req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("EnsureSheet add sheet %q: %w", sheetTitle, err)
	}
	return nil
}
