package main

import (
	"fmt"
	"math/rand"
	"time"
)

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
