package buildium

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL      string
	clientID     string
	clientSecret string
	http         *http.Client

	// Retry policy (production knobs)
	maxRetries  int
	baseBackoff time.Duration
}

func New(baseURL, clientID, clientSecret string, httpClient *http.Client) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{
		baseURL:      baseURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		http:         httpClient,
		maxRetries:   5,
		baseBackoff:  2 * time.Second,
	}
}

// doJSON executes an HTTP request (no body for now; GET-only MVP),
// handles retries for 429 / 5xx, and decodes JSON into `out`.
func (c *Client) doJSON(ctx context.Context, method, p string, q url.Values, out any) error {
	u := c.baseURL + p
	if q != nil && len(q) > 0 {
		u += "?" + q.Encode()
	}

	var lastErr error
	backoff := c.baseBackoff

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, u, nil)
		if err != nil {
			return err
		}

		// Centralized Buildium auth headers ( Python did the same) :contentReference[oaicite:2]{index=2}
		req.Header.Set("x-buildium-client-id", c.clientID)
		req.Header.Set("x-buildium-client-secret", c.clientSecret)
		req.Header.Set("accept", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		func() {
			defer resp.Body.Close()

			// Success
			if resp.StatusCode/100 == 2 {
				if out == nil {
					lastErr = nil
					return
				}
				lastErr = json.NewDecoder(resp.Body).Decode(out)
				return
			}

			// Read small error body for diagnostics
			const maxErrBody = 8 << 10
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBody))
			bodySnippet := strings.TrimSpace(string(bodyBytes))

			// Retry rules matching your Python:
			// 429 => backoff and retry :contentReference[oaicite:3]{index=3}
			// 5xx => retry :contentReference[oaicite:4]{index=4}
			if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
				lastErr = fmt.Errorf("buildium transient error: %s %s -> %s: %s", method, p, resp.Status, bodySnippet)
				return
			}

			// Non-retryable (401/403/400 etc.) — surface it immediately
			lastErr = fmt.Errorf("buildium api error: %s %s -> %s: %s", method, p, resp.Status, bodySnippet)
		}()

		// If lastErr is nil, we decoded successfully.
		if lastErr == nil {
			return nil
		}

		// Only sleep+retry if it's a transient error type (we encode that as containing "transient").
		// If it's a non-retryable API error, return immediately.
		if strings.Contains(lastErr.Error(), "transient error") {
			time.Sleep(backoff)
			backoff *= 2
			continue
		}
		return lastErr
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("buildium request failed after retries: %s %s", method, p)
	}
	return lastErr
}
