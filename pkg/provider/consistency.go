package provider

import (
	"context"
	"fmt"
	"time"
)

// ConsistencyConfig options for adaptive diff retry polling.
type ConsistencyConfig struct {
	MaxRetries int
	RetryDelay time.Duration
}

// DefaultConsistencyConfig returns default retry parameters.
func DefaultConsistencyConfig() ConsistencyConfig {
	return ConsistencyConfig{
		MaxRetries: 4,
		RetryDelay: 1 * time.Second,
	}
}

// RetryOnStaleDiff executes a diff fetch operation with exponential backoff if stale.
func RetryOnStaleDiff[T any](ctx context.Context, cfg ConsistencyConfig, fetchFn func() (T, string, error), expectedSHA string) (T, error) {
	var zero T
	delay := cfg.RetryDelay
	if delay <= 0 {
		delay = 1 * time.Second
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 4
	}

	for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {
		result, actualSHA, err := fetchFn()
		if err == nil {
			// If expectedSHA provided, verify match
			if expectedSHA == "" || actualSHA == "" || expectedSHA == actualSHA || (len(expectedSHA) >= 8 && len(actualSHA) >= 8 && expectedSHA[:8] == actualSHA[:8]) {
				return result, nil
			}
			// SHA mismatch indicates API caching older commit
		}

		if attempt == cfg.MaxRetries {
			if err != nil {
				return zero, fmt.Errorf("diff fetch failed after %d attempts: %w", attempt, err)
			}
			// Return result with warning if SHA still mismatched
			return result, nil
		}

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
			delay *= 2 // Exponential backoff
		}
	}

	return zero, fmt.Errorf("diff fetch timed out")
}
