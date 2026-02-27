package buildium

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"brh-automation/internal/httpx"
)

func TestPing(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ping" {
			http.NotFound(w, r)
			return
		}

		if r.Header.Get("x-buildium-client-id") != "id123" {
			http.Error(w, "bad client id", http.StatusUnauthorized)
			return
		}

		if r.Header.Get("x-buildium-client-secret") != "sec456" {
			http.Error(w, "bad client secret", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "id123", "sec456", httpx.NewDefaultClient())
	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
}
