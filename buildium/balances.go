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

// FetchOutstandingBalancesForLeaseIDs fetches balances ONLY for the provided lease IDs.
// This powers quick-mode in collections-sync.
//
// Buildium accepts repeated query params:
// /leases/outstandingbalances?leaseids=1&leaseids=2...
func (c *Client) FetchOutstandingBalancesForLeaseIDs(ctx context.Context, leaseIDs []int) (map[int]float64, error) {
	out := make(map[int]float64, len(leaseIDs))
	if len(leaseIDs) == 0 {
		return out, nil
	}

	const chunkSize = 50

	for i := 0; i < len(leaseIDs); i += chunkSize {
		end := i + chunkSize
		if end > len(leaseIDs) {
			end = len(leaseIDs)
		}

		q := url.Values{}
		for _, id := range leaseIDs[i:end] {
			if id > 0 {
				q.Add("leaseids", strconv.Itoa(id))
			}
		}

		var raw json.RawMessage
		if err := c.doJSON(ctx, http.MethodGet, "/leases/outstandingbalances", q, &raw); err != nil {
			return nil, err
		}

		items, err := decodeOutstandingList(raw)
		if err != nil {
			return nil, err
		}

		for _, it := range items {
			if it.LeaseID != 0 {
				out[it.LeaseID] = it.TotalBalance
			}
		}
	}

	return out, nil
}

func decodeOutstandingList(raw json.RawMessage) ([]OutstandingBalance, error) {
	// API sometimes returns an array
	var arr []OutstandingBalance
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}

	// Sometimes wrapped in { items: [] }
	var obj struct {
		Items []OutstandingBalance `json:"items"`
	}

	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}

	return obj.Items, nil
}
