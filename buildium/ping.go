package buildium

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) Ping(ctx context.Context) error {
	// If Buildium has a real "ping" endpoint you use, keep it.
	// Otherwise, you can point this at a cheap endpoint you know is safe.
	// For now, we keep "/ping" because your test server uses it.
	var out struct {
		OK bool `json:"ok"`
	}

	if err := c.doJSON(ctx, http.MethodGet, "/ping", url.Values{}, &out); err != nil {
		return err
	}
	if !out.OK {
		return fmt.Errorf("ping returned ok=false")
	}
	return nil
}
