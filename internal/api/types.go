package api

import (
	"net/http"
	"sync"
	"time"
)

type DabAPI struct {
	endpoint       string
	outputLocation string
	client         *http.Client
	mu             sync.Mutex // Mutex to protect rate limiter
	rateLimiter    *time.Ticker // Rate limiter for API requests
}

// Endpoint returns the API endpoint
func (api *DabAPI) Endpoint() string {
	return api.endpoint
}
