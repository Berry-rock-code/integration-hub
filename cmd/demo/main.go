package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"brh-automation/internal/buildium"
	"brh-automation/internal/httpx"
)

func main() {
	// Demo server (no real credentials needed)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ping" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer srv.Close()

	c := buildium.New(srv.URL, "anything", "anything", httpx.NewDefaultClient())

	if err := c.Ping(context.Background()); err != nil {
		fmt.Println("demo FAILED:", err)
		return
	}

	fmt.Println("demo OK ✅ (cmd/demo is calling internal/buildium successfully)")
}
