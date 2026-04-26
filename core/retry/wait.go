package retry

import (
	"context"
	"errors"
	"time"
)

// PollOptions configures polling behavior.
type PollOptions struct {
	Interval time.Duration // how often to check
	Timeout  time.Duration // maximum time to wait
}

// Until polls fn at the given interval until it returns true or timeout expires.
// Returns nil if condition was met, error if timeout or context cancelled.
func Until(ctx context.Context, opts PollOptions, fn func(ctx context.Context) (bool, error)) error {
	deadline := time.After(opts.Timeout)

	for {
		// Check context first.
		if err := ctx.Err(); err != nil {
			return err
		}

		ok, err := fn(ctx)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}

		select {
		case <-deadline:
			return errors.New("polling timed out")
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(opts.Interval):
			// continue polling
		}
	}
}
