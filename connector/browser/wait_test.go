package browser

import (
	"context"
	"testing"
	"time"
)

func TestWaitForSelector_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := waitForSelector(ctx, nil, "css", "#test", 5*time.Second)
	if err == nil {
		t.Error("expected error on canceled context")
	}
}

func TestDefaultWaitTimeout(t *testing.T) {
	if defaultWaitTimeout != 30*time.Second {
		t.Errorf("defaultWaitTimeout = %v, want 30s", defaultWaitTimeout)
	}
}
