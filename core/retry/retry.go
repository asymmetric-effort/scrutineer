// Package retry provides auto-wait, retry, and polling utilities.
package retry

import (
	"context"
	"math"
	"time"
)

// Policy defines retry behavior.
type Policy struct {
	MaxAttempts int           // maximum number of attempts (including first try)
	InitialWait time.Duration // wait before first retry
	MaxWait     time.Duration // maximum wait between retries
	Multiplier  float64       // backoff multiplier (e.g. 2.0 for exponential)
}

// DefaultPolicy returns a sensible default retry policy.
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts: 3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     5 * time.Second,
		Multiplier:  2.0,
	}
}

// Do executes fn with retries according to the policy.
// Returns the last error if all attempts fail.
// Respects context cancellation.
func Do(ctx context.Context, policy Policy, fn func(ctx context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		// Check context before each attempt.
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		// Don't wait after the last attempt.
		if attempt+1 >= policy.MaxAttempts {
			break
		}

		// Calculate backoff: wait = min(InitialWait * Multiplier^attempt, MaxWait)
		wait := time.Duration(float64(policy.InitialWait) * math.Pow(policy.Multiplier, float64(attempt)))
		if wait > policy.MaxWait {
			wait = policy.MaxWait
		}

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return lastErr
}
