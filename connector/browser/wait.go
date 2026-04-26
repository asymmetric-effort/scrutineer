package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/scrutineer/scrutineer/connector/browser/cdp"
)

// defaultWaitTimeout is the default timeout for wait operations.
const defaultWaitTimeout = 30 * time.Second

// waitForSelector polls until an element matching the selector appears in the DOM.
func waitForSelector(ctx context.Context, session *cdp.Session, selectorType, selectorValue string, timeout time.Duration) error {
	if timeout == 0 {
		timeout = defaultWaitTimeout
	}

	deadline := time.Now().Add(timeout)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		expr := resolveSelector(selectorType, selectorValue)
		jsExpr := fmt.Sprintf(`(function() { var el = %s; return el !== null && el !== undefined; })()`, expr)

		val, err := pageEvaluate(session, jsExpr, true)
		if err == nil && val == true {
			return nil
		}

		time.Sleep(pollInterval)
	}

	return fmt.Errorf("browser: timeout waiting for selector: %s(%s)", selectorType, selectorValue)
}

// waitForNavigation waits for Page.loadEventFired or the context deadline.
func waitForNavigation(ctx context.Context, client *cdp.Client, timeout time.Duration) error {
	if timeout == 0 {
		timeout = defaultWaitTimeout
	}

	done := make(chan struct{}, 1)
	client.Subscribe("Page.loadEventFired", func(params map[string]any) {
		select {
		case done <- struct{}{}:
		default:
		}
	})

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(timeout):
		return fmt.Errorf("browser: timeout waiting for navigation")
	}
}
