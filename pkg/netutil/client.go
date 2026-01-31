package netutil

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"time"
)

// NewRobustHTTPClient creates an HTTP client that is more resilient to DNS issues (common in Termux)
func NewRobustHTTPClient(timeout time.Duration, insecure bool) *http.Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// Custom resolver that tries system DNS and falls back to Cloudflare/Google if it fails
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			// Try the requested address first (system DNS)
			conn, err := dialer.DialContext(ctx, network, address)
			if err == nil {
				return conn, nil
			}

			// If it fails (like "connection refused" on [::1]:53), fallback to a public DNS
			// We try Cloudflare (1.1.1.1)
			if !strings.Contains(address, "1.1.1.1") && !strings.Contains(address, "8.8.8.8") {
				return dialer.DialContext(ctx, network, "1.1.1.1:53")
			}
			
			return nil, err
		},
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}

	// Hook the custom resolver into the dialer used by the transport
	dialer.Resolver = resolver
	transport.DialContext = dialer.DialContext

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}
