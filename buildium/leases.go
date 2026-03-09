package buildium

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) ListAllLeases(ctx context.Context) ([]Lease, error) {
	return c.listAllLeasesPaged(ctx, 0)
}

func (c *Client) ListActiveLeases(ctx context.Context) ([]Lease, error) {
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
			// hit safety cap: return partial results
			break
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
		pages++

		if len(items) < limit {
			break
		}

		offset += limit
	}

	return all, nil
}

// =========================================================================================
// NEW CODE FOR FETCHING ALL LEASES, NOT JUST ACTIVE ONLY
// =========================================================================================

// maxPages:
//
// 0 => full scan
// >0 => stop after maxPages pages (safety cap)
func (c *Client) ListAllLeasesLimited(ctx context.Context, maxPages int) ([]Lease, error) {
	if maxPages < 0 {
		return nil, fmt.Errorf("maxPages must be >= 0")
	}
	return c.listAllLeasesPaged(ctx, maxPages)
}

func (c *Client) listAllLeasesPaged(ctx context.Context, maxPages int) ([]Lease, error) {
	const limit = 1000

	var all []Lease
	offset := 0
	pages := 0

	for {
		if maxPages > 0 && pages >= maxPages {
			// hit safety cap: return partial results
			break
		}

		q := url.Values{}
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
		pages++

		if len(items) < limit {
			break
		}

		offset += limit
	}

	return all, nil
}

func decodeLeaseList(raw json.RawMessage) ([]Lease, error) {
	// Try direct array
	var arr []Lease
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}

	// Try wrapped format { "Items": [...] }
	var wrapped struct {
		Items []Lease `json:"Items"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil {
		return wrapped.Items, nil
	}

	return nil, fmt.Errorf("unexpected lease list format")
}
