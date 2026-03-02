package sheets

import (
	"context"
	"fmt"

	"google.golang.org/api/sheets/v4"
)

func (c *Client) ReadRange(ctx context.Context, a1Range string) ([][]interface{}, error) {
	resp, err := c.svc.Spreadsheets.Values.Get(c.SpreadsheetID, a1Range).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("ReadRange %q: %w", a1Range, err)
	}
	return resp.Values, nil
}

// WriteRange overwrites a range (like A1:Z100) with the provided values.
func (c *Client) WriteRange(ctx context.Context, a1Range string, values [][]interface{}) error {
	vr := &sheets.ValueRange{
		Range:  a1Range,
		Values: values,
	}
	_, err := c.svc.Spreadsheets.Values.Update(c.SpreadsheetID, a1Range, vr).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("WriteRange %q: %w", a1Range, err)
	}
	return nil
}

// AppendRows appends rows to the bottom of a sheet (tab). Range should be like "Sheet1!A:Z".
func (c *Client) AppendRows(ctx context.Context, a1Range string, values [][]interface{}) error {
	vr := &sheets.ValueRange{
		Range:  a1Range,
		Values: values,
	}
	_, err := c.svc.Spreadsheets.Values.Append(c.SpreadsheetID, a1Range, vr).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("AppendRows %q: %w", a1Range, err)
	}
	return nil
}
