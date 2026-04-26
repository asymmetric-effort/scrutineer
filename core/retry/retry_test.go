package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDoSucceedsFirstAttempt(t *testing.T) {
	calls := 0
	err := Do(context.Background(), DefaultPolicy(), func(ctx context.Context) error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDoSucceedsSecondAttempt(t *testing.T) {
	calls := 0
	err := Do(context.Background(), Policy{
		MaxAttempts: 3,
		InitialWait: 1 * time.Millisecond,
		MaxWait:     100 * time.Millisecond,
		Multiplier:  2.0,
	}, func(ctx context.Context) error {
		calls++
		if calls == 1 {
			return errors.New("transient error")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestDoFailsAllAttempts(t *testing.T) {
	calls := 0
	lastErr := errors.New("final error")
	err := Do(context.Background(), Policy{
		MaxAttempts: 3,
		InitialWait: 1 * time.Millisecond,
		MaxWait:     10 * time.Millisecond,
		Multiplier:  2.0,
	}, func(ctx context.Context) error {
		calls++
		if calls == 3 {
			return lastErr
		}
		return errors.New("error " + string(rune('0'+calls)))
	})
	if err != lastErr {
		t.Fatalf("expected last error, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDoMaxAttemptsOne(t *testing.T) {
	calls := 0
	sentinel := errors.New("only error")
	err := Do(context.Background(), Policy{
		MaxAttempts: 1,
		InitialWait: 1 * time.Millisecond,
		MaxWait:     10 * time.Millisecond,
		Multiplier:  2.0,
	}, func(ctx context.Context) error {
		calls++
		return sentinel
	})
	if err != sentinel {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDoBackoffTimingIncreases(t *testing.T) {
	var timestamps []time.Time
	policy := Policy{
		MaxAttempts: 4,
		InitialWait: 10 * time.Millisecond,
		MaxWait:     1 * time.Second,
		Multiplier:  2.0,
	}
	_ = Do(context.Background(), policy, func(ctx context.Context) error {
		timestamps = append(timestamps, time.Now())
		return errors.New("fail")
	})

	if len(timestamps) != 4 {
		t.Fatalf("expected 4 timestamps, got %d", len(timestamps))
	}

	// Verify delays increase: gap2 >= gap1 (with some tolerance).
	gap1 := timestamps[1].Sub(timestamps[0])
	gap2 := timestamps[2].Sub(timestamps[1])
	gap3 := timestamps[3].Sub(timestamps[2])

	// Each gap should be at least a few ms (the initial wait).
	if gap1 < 5*time.Millisecond {
		t.Errorf("gap1 too small: %v", gap1)
	}
	// gap2 should be larger than gap1 (multiplier=2).
	if gap2 < gap1 {
		t.Errorf("expected gap2 (%v) >= gap1 (%v)", gap2, gap1)
	}
	// gap3 should be larger than or equal to gap2.
	if gap3 < gap2-5*time.Millisecond {
		t.Errorf("expected gap3 (%v) >= gap2 (%v) approximately", gap3, gap2)
	}
}

func TestDoContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := Do(ctx, Policy{
		MaxAttempts: 100,
		InitialWait: 50 * time.Millisecond,
		MaxWait:     1 * time.Second,
		Multiplier:  1.0,
	}, func(ctx context.Context) error {
		calls++
		return errors.New("keep failing")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestDoContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Do(ctx, DefaultPolicy(), func(ctx context.Context) error {
		t.Fatal("fn should not be called")
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestDefaultPolicyValues(t *testing.T) {
	p := DefaultPolicy()
	if p.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts=3, got %d", p.MaxAttempts)
	}
	if p.InitialWait != 100*time.Millisecond {
		t.Errorf("expected InitialWait=100ms, got %v", p.InitialWait)
	}
	if p.MaxWait != 5*time.Second {
		t.Errorf("expected MaxWait=5s, got %v", p.MaxWait)
	}
	if p.Multiplier != 2.0 {
		t.Errorf("expected Multiplier=2.0, got %v", p.Multiplier)
	}
}

func TestDoBackoffCappedByMaxWait(t *testing.T) {
	var timestamps []time.Time
	policy := Policy{
		MaxAttempts: 4,
		InitialWait: 10 * time.Millisecond,
		MaxWait:     15 * time.Millisecond,
		Multiplier:  10.0,
	}
	_ = Do(context.Background(), policy, func(ctx context.Context) error {
		timestamps = append(timestamps, time.Now())
		return errors.New("fail")
	})

	if len(timestamps) != 4 {
		t.Fatalf("expected 4 timestamps, got %d", len(timestamps))
	}

	// With multiplier=10 and maxWait=15ms, later gaps should be capped.
	gap3 := timestamps[3].Sub(timestamps[2])
	if gap3 > 100*time.Millisecond {
		t.Errorf("gap3 too large (should be capped): %v", gap3)
	}
}
