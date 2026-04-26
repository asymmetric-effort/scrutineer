package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithTimeoutFnCompletesBefore(t *testing.T) {
	err := WithTimeout(context.Background(), 1*time.Second, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestWithTimeoutFnExceedsTimeout(t *testing.T) {
	start := time.Now()
	err := WithTimeout(context.Background(), 50*time.Millisecond, func(ctx context.Context) error {
		<-ctx.Done()
		// Simulate slow cleanup after context cancellation.
		time.Sleep(500 * time.Millisecond)
		return ctx.Err()
	})
	elapsed := time.Since(start)

	var te *TimeoutError
	if !errors.As(err, &te) {
		t.Fatalf("expected TimeoutError, got %T: %v", err, err)
	}
	if te.Duration != 50*time.Millisecond {
		t.Errorf("expected duration 50ms, got %v", te.Duration)
	}
	// Should return quickly (at timeout), not wait for fn to finish.
	if elapsed > 300*time.Millisecond {
		t.Errorf("WithTimeout should return at timeout, took %v", elapsed)
	}
}

func TestWithTimeoutFnReturnsOwnError(t *testing.T) {
	sentinel := errors.New("fn error")
	err := WithTimeout(context.Background(), 1*time.Second, func(ctx context.Context) error {
		return sentinel
	})
	if err != sentinel {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

func TestWithTimeoutContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := WithTimeout(ctx, 1*time.Second, func(ctx context.Context) error {
		t.Fatal("fn should not be called")
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestWithTimeoutParentContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	err := WithTimeout(ctx, 5*time.Second, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	// Parent cancellation should propagate.
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestTimeoutErrorMessage(t *testing.T) {
	te := &TimeoutError{
		Duration: 2 * time.Second,
		Message:  "test op",
	}
	expected := "timeout after 2s: test op"
	if te.Error() != expected {
		t.Errorf("expected %q, got %q", expected, te.Error())
	}
}

func TestTimeoutErrorMessageEmpty(t *testing.T) {
	te := &TimeoutError{
		Duration: 500 * time.Millisecond,
	}
	expected := "timeout after 500ms"
	if te.Error() != expected {
		t.Errorf("expected %q, got %q", expected, te.Error())
	}
}
