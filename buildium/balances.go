package buildium

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) FetchOutstandingBalances(ctx context.Context) (map[int]float64, error) {
	const limit = 100

	debtMap := make(map[int]float64)
	offset := 0

	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		q.Set("offset", strconv.Itoa(offset))

		var raw json.RawMessage
		if err := c.doJSON(ctx, http.MethodGet, "/leases/outstandingbalances", q, &raw); err != nil {
			return nil, err
		}

		items, err := decodeOutstandingList(raw)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}

		for _, it := range items {
			if it.LeaseID != 0 {
				debtMap[it.LeaseID] = it.TotalBalance
			}
		}

		if len(items) < limit {
			break
		}
		offset += limit
	}

	return debtMap, nil
}

func decodeOutstandingList(raw json.RawMessage) ([]OutstandingBalance, error) {
	// Try array form
	var arr []OutstandingBalance
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}

	// Try object-with-items form
	var obj struct {
		Items []OutstandingBalance `json:"items"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	return obj.Items, nil
}
