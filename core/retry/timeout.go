package retry

import (
	"context"
	"fmt"
	"time"
)

// TimeoutError indicates an operation exceeded its time limit.
type TimeoutError struct {
	Duration time.Duration
	Message  string
}

// Error returns a human-readable description of the timeout.
func (e *TimeoutError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("timeout after %s: %s", e.Duration, e.Message)
	}
	return fmt.Sprintf("timeout after %s", e.Duration)
}

// WithTimeout runs fn with a timeout. Returns the error from fn,
// or a TimeoutError if the timeout expires before fn completes.
func WithTimeout(ctx context.Context, d time.Duration, fn func(ctx context.Context) error) error {
	// Check if context is already cancelled.
	if err := ctx.Err(); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, d)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- fn(timeoutCtx)
	}()

	select {
	case err := <-done:
		return err
	case <-timeoutCtx.Done():
		// Determine if it was our timeout or the parent context.
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &TimeoutError{
			Duration: d,
			Message:  "operation did not complete in time",
		}
	}
}
