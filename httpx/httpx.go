package httpx

import (
	"net"
	"net/http"
	"time"
)

// NewDefaultClient returns a shared HTTP client with sane timeouts for external APIs.
func NewDefaultClient() *http.Client {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
	}

	return &http.Client{
		Transport: tr,
		Timeout:   60 * time.Second, // bumped to avoid Buildium page timeouts
	}
}
