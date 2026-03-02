package buildium

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) ListActiveLeases(ctx context.Context) ([]Lease, error) {
	const limit = 100

	var all []Lease
	offset := 0

	for {
		q := url.Values{}
		q.Set("status", "Active")
		q.Set("limit", strconv.Itoa(limit))
		q.Set("offset", strconv.Itoa(offset))
		q.Set("expand", "tenants,unit")

		// We decode into raw JSON first so we can support either:
		// - []Lease
		// - {"items":[...]}
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
	}

	return all, nil
}

func decodeLeaseList(raw json.RawMessage) ([]Lease, error) {
	// Try array form
	var arr []Lease
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}

	// Try object-with-items form
	var obj struct {
		Items []Lease `json:"items"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	return obj.Items, nil
}
