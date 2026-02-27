package buildium

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Client struct {
	baseURL      string
	clientID     string
	clientSecret string
	http         *http.Client
}

func New(baseURL, clientID, clientSecret string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		http:         httpClient,
	}
}

// Ping is a smoke test to prove:
// - the app can import this package
// - HTTP works
// - auth headers are being set correctly
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/ping", nil)
	if err != nil {
		return err
	}

	// Buildium uses header based auth, we centralize it here
	req.Header.Set("x-buildium-client-id", c.clientID)
	req.Header.Set("x-buildium-client-secret", c.clientSecret)
	req.Header.Set("accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("Buildium ping failed: %s", resp.Status)
	}

	var out struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if !out.OK {
		return fmt.Errorf("buildium ping returned ok=false")
	}

	return nil
}
