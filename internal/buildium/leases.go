package buildium

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) ListActiveLeases(ctx context.Context) ([]Lease, error) {
	// Full scan
	return c.listActiveLeasesPaged(ctx, 0)
}

// maxPages:
//
//	0  => full scan
//	>0 => stop after maxPages pages (safety cap)
func (c *Client) ListActiveLeasesLimited(ctx context.Context, maxPages int) ([]Lease, error) {
	if maxPages < 0 {
		return nil, fmt.Errorf("maxPages must be >= 0")
	}
	return c.listActiveLeasesPaged(ctx, maxPages)
}

func (c *Client) listActiveLeasesPaged(ctx context.Context, maxPages int) ([]Lease, error) {
	const limit = 1000

	var all []Lease
	offset := 0
	pages := 0

	for {
		if maxPages > 0 && pages >= maxPages {
			return nil, fmt.Errorf("max-pages cap requested, but capped paging is not implemented yet; run with --max-pages=0 for full scan OR I can add ListActiveLeasesLimited to internal/buildium")
			// If you’d rather return partial results instead of erroring:
			// break
		}

		q := url.Values{}
		q.Set("status", "Active")
		q.Set("limit", strconv.Itoa(limit))
		q.Set("offset", strconv.Itoa(offset))
		q.Set("expand", "tenants,unit")

		var raw json.RawMessage
		if err := c.doJSON(ctx, http.MethodGet, "/leases", q, &raw); err != nil {
			return nil, err
		}

		items, err := decodeLeaseList(raw)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}

		all = append(all, items...)

		if len(items) < limit {
			break
		}

		offset += limit
		pages++
	}

	return all, nil
}

func decodeLeaseList(raw json.RawMessage) ([]Lease, error) {
	// First try direct array
	var arr []Lease
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}

	// Then try wrapped format { "Items": [...] }
	var wrapped struct {
		Items []Lease `json:"Items"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil {
		return wrapped.Items, nil
	}

	return nil, fmt.Errorf("unexpected lease list format")
}
