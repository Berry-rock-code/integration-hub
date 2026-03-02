package sheets

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/sheets/v4"
)

type UpsertOptions struct {
	SheetTitle string

	HeaderRow int // e.g. 2
	DataRow   int // e.g. 3

	KeyHeader string // e.g. "Lease ID"

	// If true, ensures header row matches the provided headers (overwrites that row).
	// If false, reads existing headers and uses them for key lookup.
	EnsureHeaders bool
	Headers       []string

	// Optional: limit read range width; if 0, derived from headers length.
	NumColumns int
}

// UpsertRows upserts rows by KeyHeader.
// - reads existing sheet values once (from HeaderRow downward)
// - builds map key -> row index
// - batch-updates existing matches and appends new keys
func (c *Client) UpsertRows(ctx context.Context, opts UpsertOptions, rows [][]interface{}) error {
	if opts.SheetTitle == "" {
		return fmt.Errorf("UpsertRows: SheetTitle required")
	}
	if opts.HeaderRow <= 0 || opts.DataRow <= 0 || opts.DataRow <= opts.HeaderRow {
		return fmt.Errorf("UpsertRows: invalid HeaderRow/DataRow (header=%d data=%d)", opts.HeaderRow, opts.DataRow)
	}
	if strings.TrimSpace(opts.KeyHeader) == "" {
		return fmt.Errorf("UpsertRows: KeyHeader required")
	}

	if err := c.EnsureSheet(ctx, opts.SheetTitle); err != nil {
		return err
	}

	// Determine headers
	var headers []string
	if opts.EnsureHeaders {
		if len(opts.Headers) == 0 {
			return fmt.Errorf("UpsertRows: EnsureHeaders true but Headers is empty")
		}
		// Write header row
		hVals := make([]interface{}, 0, len(opts.Headers))
		for _, h := range opts.Headers {
			hVals = append(hVals, h)
		}
		a1 := fmt.Sprintf("%s!%s:%s", opts.SheetTitle, cellA1(1, opts.HeaderRow), cellA1(len(opts.Headers), opts.HeaderRow))
		if err := c.WriteRange(ctx, a1, [][]interface{}{hVals}); err != nil {
			return err
		}
		headers = opts.Headers
	} else {
		// Read header row
		// read a “wide enough” range - we can read A:ZZ safely
		hA1 := fmt.Sprintf("%s!A%d:ZZ%d", opts.SheetTitle, opts.HeaderRow, opts.HeaderRow)
		vals, err := c.ReadRange(ctx, hA1)
		if err != nil {
			return err
		}
		if len(vals) == 0 || len(vals[0]) == 0 {
			return fmt.Errorf("UpsertRows: header row %d is empty", opts.HeaderRow)
		}
		headers = ParseHeaderRow(vals[0])
	}

	keyIdx := FindHeaderIndex(headers, opts.KeyHeader)
	if keyIdx < 0 {
		return fmt.Errorf("UpsertRows: key header %q not found in headers", opts.KeyHeader)
	}

	numCols := opts.NumColumns
	if numCols <= 0 {
		if len(headers) == 0 {
			return fmt.Errorf("UpsertRows: cannot infer NumColumns")
		}
		numCols = len(headers)
	}

	// Read existing data from DataRow down, limited to numCols
	readA1 := fmt.Sprintf("%s!%s:%s", opts.SheetTitle, cellA1(1, opts.DataRow), cellA1(numCols, 50000))
	existing, err := c.ReadRange(ctx, readA1)
	if err != nil {
		return err
	}

	// Build key -> absolute row number
	keyToRow := make(map[string]int, len(existing))
	for i, r := range existing {
		// absolute sheet row number:
		sheetRow := opts.DataRow + i
		if keyIdx >= len(r) {
			continue
		}
		k := strings.TrimSpace(fmt.Sprintf("%v", r[keyIdx]))
		if k == "" {
			continue
		}
		// first occurrence wins
		if _, ok := keyToRow[k]; !ok {
			keyToRow[k] = sheetRow
		}
	}

	// Prepare batch updates & appends
	var updateRanges []*sheets.ValueRange
	var toAppend [][]interface{}

	for _, newRow := range rows {
		// normalize length
		norm := make([]interface{}, numCols)
		for i := 0; i < numCols && i < len(newRow); i++ {
			norm[i] = newRow[i]
		}

		if keyIdx >= len(norm) {
			continue
		}
		k := strings.TrimSpace(fmt.Sprintf("%v", norm[keyIdx]))
		if k == "" {
			continue
		}

		if rowNum, ok := keyToRow[k]; ok {
			// update entire row
			a1 := fmt.Sprintf("%s!%s:%s", opts.SheetTitle, cellA1(1, rowNum), cellA1(numCols, rowNum))
			updateRanges = append(updateRanges, &sheets.ValueRange{
				Range:  a1,
				Values: [][]interface{}{norm},
			})
		} else {
			toAppend = append(toAppend, norm)
		}
	}

	// Batch update existing rows
	if len(updateRanges) > 0 {
		req := &sheets.BatchUpdateValuesRequest{
			ValueInputOption: "USER_ENTERED",
			Data:             updateRanges,
		}
		_, err := c.svc.Spreadsheets.Values.BatchUpdate(c.SpreadsheetID, req).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("UpsertRows batch update: %w", err)
		}
	}

	// Append new rows
	if len(toAppend) > 0 {
		appendA1 := fmt.Sprintf("%s!%s:%s", opts.SheetTitle, cellA1(1, opts.DataRow), cellA1(numCols, opts.DataRow))
		if err := c.AppendRows(ctx, appendA1, toAppend); err != nil {
			return err
		}
	}

	return nil
}
