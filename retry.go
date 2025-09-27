package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	StatusCode int
	Status     string
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s - %s", e.StatusCode, e.Status, e.Message)
}

// IsRetryableHTTPError checks if an HTTP error should be retried
func IsRetryableHTTPError(err error) bool {
	// Unwrap the error if it's wrapped
	for err != nil {
		if httpErr, ok := err.(*HTTPError); ok {
			switch httpErr.StatusCode {
			case http.StatusServiceUnavailable, // 503
				http.StatusTooManyRequests,     // 429
				http.StatusBadGateway,          // 502
				http.StatusGatewayTimeout:      // 504
				return true
			}
		}
		// Try to unwrap the error further
		if unwrapped, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapped.Unwrap()
		} else {
			break
		}
	}
	return false
}

// RetryWithBackoff retries the given function with exponential backoff.
func RetryWithBackoff(maxRetries int, initialDelaySec int, fn func() error) error {
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		// Calculate delay with exponential backoff and some jitter
		delay := time.Duration(initialDelaySec) * time.Second * (1 << attempt)
		jitter := time.Duration(rand.Intn(100)) * time.Millisecond
		time.Sleep(delay + jitter)
	}
	return fmt.Errorf("failed after %d attempts: %w", maxRetries, err)
}

// RetryWithBackoffForHTTP retries HTTP requests with smart error handling
func RetryWithBackoffForHTTP(maxRetries int, initialDelay time.Duration, maxDelay time.Duration, fn func() error) error {
	return RetryWithBackoffForHTTPWithDebug(maxRetries, initialDelay, maxDelay, fn, false)
}

// RetryWithBackoffForHTTPWithDebug retries HTTP requests with smart error handling and optional debug logging
func RetryWithBackoffForHTTPWithDebug(maxRetries int, initialDelay time.Duration, maxDelay time.Duration, fn func() error, debug bool) error {
	var lastErr error

	if maxRetries == 0 { // If no retries, just execute once
		return fn()
	}

	for attempt := 0; attempt < maxRetries; attempt++ {		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Check if this is a retryable HTTP error
		if !IsRetryableHTTPError(lastErr) {
			return lastErr // Don't retry non-retryable errors
		}

		if attempt == maxRetries-1 {
			break // Don't sleep on the last attempt
		}

		// Calculate delay with exponential backoff and jitter
		delay := initialDelay * time.Duration(1<<uint(attempt))
		if delay > maxDelay {
			delay = maxDelay
		}
		
		// Add jitter (±25% of delay)
		jitter := time.Duration(rand.Int63n(int64(delay/2))) - delay/4
		finalDelay := delay + jitter
		
		if finalDelay < 0 {
			finalDelay = delay
		}

		// Only log retry messages in debug mode
		if debug {
			log.Printf("HTTP request failed (attempt %d/%d): %v. Retrying in %v", 
				attempt+1, maxRetries, lastErr, finalDelay)
		}
		
		time.Sleep(finalDelay)
	}
	
	return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
