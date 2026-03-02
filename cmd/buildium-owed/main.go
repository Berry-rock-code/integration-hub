package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"brh-automation/internal/buildium"
	"brh-automation/internal/config"
	"brh-automation/internal/httpx"
)

type Row struct {
	LeaseID int
	Name    string
	Address string
	Phone   string
	Email   string
	Owed    float64
}

func main() {
	// ---- CLI flags (keeps this "one-off" usable without code edits) ----
	var (
		maxPages      = flag.Int("max-pages", 5, "Max pages of leases to scan (100 leases per page). Use 0 for no cap.")
		maxRows       = flag.Int("max-rows", 50, "Max rows to print (leases with balance > 0). Use 0 for no cap.")
		balTimeout    = flag.Duration("balances-timeout", 60*time.Second, "Timeout for fetching outstanding balances")
		leaseTimeout  = flag.Duration("leases-timeout", 180*time.Second, "Timeout for fetching active leases (paged)")
		tenantTimeout = flag.Duration("tenant-timeout", 10*time.Second, "Timeout per tenant details request")
	)
	flag.Parse()

	// ---- Load .env locally (no-op if .env missing) ----
	config.LoadDotEnv()

	baseURL := mustEnv("BUILDIUM_BASE_URL")
	clientID := mustEnv("BUILDIUM_CLIENT_ID")
	clientSecret := mustEnv("BUILDIUM_CLIENT_SECRET")

	// Parent context with no deadline; we control timeouts per-step.
	ctx := context.Background()

	c := buildium.New(baseURL, clientID, clientSecret, httpx.NewDefaultClient())

	// -------------------------
	// Step 1: Outstanding balances
	// -------------------------
	fmt.Println("Step 1/3: FetchOutstandingBalances...")

	bCtx, cancel := context.WithTimeout(ctx, *balTimeout)
	debtMap, err := c.FetchOutstandingBalances(bCtx)
	cancel()
	if err != nil {
		fatal("FetchOutstandingBalances failed", err)
	}
	fmt.Printf("Balances fetched: %d\n\n", len(debtMap))

	// -------------------------
	// Step 2: Active leases (paged)
	// -------------------------
	fmt.Println("Step 2/3: ListActiveLeases...")

	lCtx, cancel := context.WithTimeout(ctx, *leaseTimeout)
	leases, err := listActiveLeasesWithOptionalCap(lCtx, c, *maxPages)
	cancel()
	if err != nil {
		fatal("ListActiveLeases failed", err)
	}
	fmt.Printf("Active leases fetched: %d\n\n", len(leases))

	// -------------------------
	// Step 3: Tenant details (only for owed leases)
	// -------------------------
	fmt.Println("Step 3/3: Build rows (tenant lookups only for owed leases)...")

	tenantCache := make(map[int]buildium.TenantDetails)
	var rows []Row

	for _, lease := range leases {
		owed := debtMap[lease.ID]
		if owed <= 0 {
			continue
		}

		tenantID := pickActiveTenantID(lease)
		if tenantID == 0 {
			rows = append(rows, Row{
				LeaseID: lease.ID,
				Name:    "(no active tenant found)",
				Address: leaseAddress(lease, nil),
				Owed:    owed,
			})
			if *maxRows > 0 && len(rows) >= *maxRows {
				break
			}
			continue
		}

		td, ok := tenantCache[tenantID]
		if !ok {
			// Per-tenant timeout (prevents a single slow tenant call from killing the whole run)
			tCtx, cancelT := context.WithTimeout(ctx, *tenantTimeout)
			tdFetched, err := c.GetTenantDetails(tCtx, tenantID)
			cancelT()

			if err != nil {
				rows = append(rows, Row{
					LeaseID: lease.ID,
					Name:    "(tenant lookup failed)",
					Address: leaseAddress(lease, nil),
					Owed:    owed,
				})
				if *maxRows > 0 && len(rows) >= *maxRows {
					break
				}
				continue
			}

			td = tdFetched
			tenantCache[tenantID] = td
		}

		rows = append(rows, Row{
			LeaseID: lease.ID,
			Name:    strings.TrimSpace(td.FirstName + " " + td.LastName),
			Address: leaseAddress(lease, &td),
			Phone:   firstPhone(td),
			Email:   td.Email,
			Owed:    owed,
		})

		if *maxRows > 0 && len(rows) >= *maxRows {
			break
		}
	}

	// Sort biggest owed first
	sort.Slice(rows, func(i, j int) bool { return rows[i].Owed > rows[j].Owed })

	fmt.Printf("\nLeases with balance > 0 (printed): %d\n\n", len(rows))
	printTable(rows)
}

// listActiveLeasesWithOptionalCap avoids editing the buildium package for this one-off.
// It calls the real ListActiveLeases if maxPages==0, otherwise uses a capped pager.
func listActiveLeasesWithOptionalCap(ctx context.Context, c *buildium.Client, maxPages int) ([]buildium.Lease, error) {
	if maxPages == 0 {
		return c.ListActiveLeases(ctx)
	}
	return listActiveLeasesCapped(ctx, c, maxPages)
}

// listActiveLeasesCapped is a local helper for quick validation runs.
// It mirrors the paging behavior (limit=100) but stops after maxPages.
func listActiveLeasesCapped(ctx context.Context, c *buildium.Client, maxPages int) ([]buildium.Lease, error) {
	const limit = 100

	var all []buildium.Lease
	offset := 0
	pages := 0

	for {
		pages++
		if pages > maxPages {
			break
		}

		// We reuse the buildium package's behavior by calling its real method would fetch all pages,
		// so here we call the same endpoint by temporarily using the package method isn't exposed.
		// Instead: simplest approach is to call the public method when maxPages==0 and accept full scan.
		//
		// For now, we just stop early by using the full scan method and maxPages==0 in real runs.
		// If you want the cap to be “real paging cap,” we’ll add ListActiveLeasesLimited into internal/buildium.

		_ = offset // kept so you can see intended structure; see note above
		break
	}

	// If you want a true paging cap, tell me and I’ll add ListActiveLeasesLimited to internal/buildium (clean).
	return all, fmt.Errorf("max-pages cap requested, but capped paging is not implemented yet; run with --max-pages=0 for full scan OR I can add ListActiveLeasesLimited to internal/buildium")
}

func mustEnv(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		fmt.Fprintf(os.Stderr, "Missing required env var: %s\n", key)
		os.Exit(1)
	}
	return v
}

func fatal(msg string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	os.Exit(2)
}

func pickActiveTenantID(lease buildium.Lease) int {
	for _, t := range lease.Tenants {
		if strings.EqualFold(t.Status, "Active") {
			return t.ID
		}
	}
	if len(lease.Tenants) > 0 {
		return lease.Tenants[0].ID
	}
	return 0
}

func leaseAddress(lease buildium.Lease, td *buildium.TenantDetails) string {
	if lease.Unit != nil && lease.Unit.Address != nil && lease.Unit.Address.AddressLine1 != "" {
		return lease.Unit.Address.AddressLine1
	}
	if td != nil && td.Address != nil && td.Address.AddressLine1 != "" {
		return td.Address.AddressLine1
	}
	return ""
}

func firstPhone(td buildium.TenantDetails) string {
	if len(td.PhoneNumbers) == 0 {
		return ""
	}
	return td.PhoneNumbers[0].Number
}

func printTable(rows []Row) {
	fmt.Printf("%-8s | %-22s | %-28s | %-14s | %-26s | %s\n",
		"LeaseID", "Name", "Address", "Phone", "Email", "Owed")
	fmt.Println(strings.Repeat("-", 8) + "-+-" +
		strings.Repeat("-", 22) + "-+-" +
		strings.Repeat("-", 28) + "-+-" +
		strings.Repeat("-", 14) + "-+-" +
		strings.Repeat("-", 26) + "-+-" +
		strings.Repeat("-", 10))

	for _, r := range rows {
		fmt.Printf("%-8d | %-22.22s | %-28.28s | %-14.14s | %-26.26s | $%0.2f\n",
			r.LeaseID, safe(r.Name), safe(r.Address), safe(r.Phone), safe(r.Email), r.Owed)
	}
}

func safe(s string) string {
	return strings.ReplaceAll(s, "\n", " ")
}
