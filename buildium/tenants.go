package buildium

import (
	"context"
	"fmt"
	"net/http"
)

func (c *Client) GetTenantDetails(ctx context.Context, tenantID int) (TenantDetails, error) {
	if tenantID <= 0 {
		return TenantDetails{}, fmt.Errorf("tenantID must be > 0")
	}

	var out TenantDetails
	path := fmt.Sprintf("/leases/tenants/%d", tenantID)

	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return TenantDetails{}, err
	}
	return out, nil
}
