package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestUntilConditionMetImmediately(t *testing.T) {
	err := Until(context.Background(), PollOptions{
		Interval: 10 * time.Millisecond,
		Timeout:  1 * time.Second,
	}, func(ctx context.Context) (bool, error) {
		return true, nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestUntilConditionMetAfterSeveralPolls(t *testing.T) {
	calls := 0
	err := Until(context.Background(), PollOptions{
		Interval: 5 * time.Millisecond,
		Timeout:  1 * time.Second,
	}, func(ctx context.Context) (bool, error) {
		calls++
		return calls >= 3, nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls < 3 {
		t.Fatalf("expected at least 3 calls, got %d", calls)
	}
}

func TestUntilTimeoutExpires(t *testing.T) {
	start := time.Now()
	err := Until(context.Background(), PollOptions{
		Interval: 5 * time.Millisecond,
		Timeout:  50 * time.Millisecond,
	}, func(ctx context.Context) (bool, error) {
		return false, nil
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "polling timed out" {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed < 30*time.Millisecond {
		t.Errorf("elapsed too short: %v", elapsed)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("elapsed too long: %v", elapsed)
	}
}

func TestUntilFnReturnsError(t *testing.T) {
	sentinel := errors.New("poll error")
	calls := 0
	err := Until(context.Background(), PollOptions{
		Interval: 5 * time.Millisecond,
		Timeout:  1 * time.Second,
	}, func(ctx context.Context) (bool, error) {
		calls++
		if calls == 2 {
			return false, sentinel
		}
		return false, nil
	})
	if err != sentinel {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

func TestUntilContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := Until(ctx, PollOptions{
		Interval: 5 * time.Millisecond,
		Timeout:  5 * time.Second,
	}, func(ctx context.Context) (bool, error) {
		return false, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestUntilContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Until(ctx, PollOptions{
		Interval: 5 * time.Millisecond,
		Timeout:  1 * time.Second,
	}, func(ctx context.Context) (bool, error) {
		t.Fatal("fn should not be called")
		return false, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestUntilZeroInterval(t *testing.T) {
	calls := 0
	err := Until(context.Background(), PollOptions{
		Interval: 0,
		Timeout:  100 * time.Millisecond,
	}, func(ctx context.Context) (bool, error) {
		calls++
		return calls >= 5, nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls < 5 {
		t.Fatalf("expected at least 5 calls, got %d", calls)
	}
}
