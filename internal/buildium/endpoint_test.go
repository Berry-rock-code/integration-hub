package buildium

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"brh-automation/internal/httpx"
)

func TestListActiveLeases_PaginatesAndSetsQuery(t *testing.T) {
	t.Parallel()

	var seenOffsets []int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// --- Path check ---
		if r.URL.Path != "/leases" {
			http.NotFound(w, r)
			return
		}

		// --- Auth header checks ---
		if r.Header.Get("x-buildium-client-id") != "id123" {
			http.Error(w, "bad client id", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("x-buildium-client-secret") != "sec456" {
			http.Error(w, "bad client secret", http.StatusUnauthorized)
			return
		}

		// --- Query checks ---
		q := r.URL.Query()
		if q.Get("status") != "Active" {
			http.Error(w, "missing status=Active", http.StatusBadRequest)
			return
		}
		if q.Get("expand") != "tenants,unit" {
			http.Error(w, "missing expand=tenants,unit", http.StatusBadRequest)
			return
		}

		limit, _ := strconv.Atoi(q.Get("limit"))
		offset, _ := strconv.Atoi(q.Get("offset"))
		seenOffsets = append(seenOffsets, offset)

		if limit != 100 {
			http.Error(w, "expected limit=100", http.StatusBadRequest)
			return
		}

		// --- Return paginated results ---
		// First call offset=0 -> return 100 items
		// Second call offset=100 -> return 1 item
		// Third call should not happen (because len < limit ends loop)
		switch offset {
		case 0:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(makeLeaseArrayJSON(100)))
		case 100:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(makeLeaseArrayJSON(1)))
		default:
			http.Error(w, "unexpected offset", http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "id123", "sec456", httpx.NewDefaultClient())
	leases, err := c.ListActiveLeases(context.Background())
	if err != nil {
		t.Fatalf("ListActiveLeases error: %v", err)
	}

	if len(leases) != 101 {
		t.Fatalf("expected 101 leases, got %d", len(leases))
	}

	// Ensure it paginated exactly twice
	if len(seenOffsets) != 2 || seenOffsets[0] != 0 || seenOffsets[1] != 100 {
		t.Fatalf("expected offsets [0,100], got %v", seenOffsets)
	}
}

func TestFetchOutstandingBalances_BuildsDebtMap(t *testing.T) {
	t.Parallel()

	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/leases/outstandingbalances" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("x-buildium-client-id") != "id123" ||
			r.Header.Get("x-buildium-client-secret") != "sec456" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		q := r.URL.Query()
		limit := q.Get("limit")
		offset := q.Get("offset")
		if limit != "100" {
			http.Error(w, "expected limit=100", http.StatusBadRequest)
			return
		}

		calls++

		// offset=0 -> 2 items
		// offset=100 -> 0 items to stop
		switch offset {
		case "0":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				{"LeaseId": 10, "TotalBalance": 123.45},
				{"LeaseId": 11, "TotalBalance": 0.00}
			]`))
		case "100":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
		default:
			http.Error(w, "unexpected offset", http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "id123", "sec456", httpx.NewDefaultClient())
	debt, err := c.FetchOutstandingBalances(context.Background())
	if err != nil {
		t.Fatalf("FetchOutstandingBalances error: %v", err)
	}

	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}

	if debt[10] != 123.45 {
		t.Fatalf("expected lease 10 balance 123.45, got %v", debt[10])
	}
	if debt[11] != 0.0 {
		t.Fatalf("expected lease 11 balance 0.0, got %v", debt[11])
	}
}

func TestGetTenantDetails_HitsCorrectPathAndDecodes(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/leases/tenants/") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path != "/leases/tenants/777" {
			http.Error(w, "wrong tenant id path", http.StatusBadRequest)
			return
		}

		if r.Header.Get("x-buildium-client-id") != "id123" ||
			r.Header.Get("x-buildium-client-secret") != "sec456" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"Id": 777,
			"FirstName": "Jane",
			"LastName": "Doe",
			"Email": "jane@example.com",
			"Address": {"AddressLine1": "123 Main St"},
			"PhoneNumbers": [{"Number": "555-111-2222"}]
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "id123", "sec456", &http.Client{Timeout: 2 * time.Second})
	td, err := c.GetTenantDetails(context.Background(), 777)
	if err != nil {
		t.Fatalf("GetTenantDetails error: %v", err)
	}

	if td.ID != 777 || td.FirstName != "Jane" || td.LastName != "Doe" {
		t.Fatalf("unexpected tenant decoded: %+v", td)
	}
	if td.Address == nil || td.Address.AddressLine1 != "123 Main St" {
		t.Fatalf("unexpected address decoded: %+v", td.Address)
	}
	if len(td.PhoneNumbers) != 1 || td.PhoneNumbers[0].Number != "555-111-2222" {
		t.Fatalf("unexpected phones decoded: %+v", td.PhoneNumbers)
	}
}

// makeLeaseArrayJSON builds a JSON array of `count` Lease objects.
// We keep minimal fields the decoder expects.
func makeLeaseArrayJSON(count int) string {
	// Minimal, but includes Tenants and Unit so we verify decoding won't choke.
	var b strings.Builder
	b.WriteString("[")
	for i := 0; i < count; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		// Note: LeaseToDate is a string in our type; we return a simple date-like string.
		b.WriteString(`{"Id":`)
		b.WriteString(strconv.Itoa(1000 + i))
		b.WriteString(`,"LeaseToDate":"2026-12-31","Tenants":[{"Id":1,"Status":"Active"}],"Unit":{"Address":{"AddressLine1":"X"}}}`)
	}
	b.WriteString("]")
	return b.String()
}
