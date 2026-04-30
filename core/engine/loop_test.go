package engine

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoopSinglePass(t *testing.T) {
	lc := NewLoopController(1, 0, 0)
	var count int
	lc.Run(context.Background(), func(_ context.Context, passNum int) {
		count++
		if passNum != 1 {
			t.Errorf("passNum = %d, want 1", passNum)
		}
	})
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestLoopFixedRepeat(t *testing.T) {
	lc := NewLoopController(5, 0, 0)
	var count int
	lc.Run(context.Background(), func(_ context.Context, passNum int) {
		count++
	})
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}
}

func TestLoopDurationBound(t *testing.T) {
	lc := NewLoopController(0, 100*time.Millisecond, 0)
	var count atomic.Int32
	start := time.Now()
	lc.Run(context.Background(), func(_ context.Context, passNum int) {
		count.Add(1)
		time.Sleep(10 * time.Millisecond)
	})
	elapsed := time.Since(start)
	if elapsed > 200*time.Millisecond {
		t.Errorf("took too long: %v", elapsed)
	}
	if count.Load() < 2 {
		t.Errorf("expected at least 2 passes, got %d", count.Load())
	}
}

func TestLoopDurationAndRepeat(t *testing.T) {
	// Repeat 100 but duration is only 50ms — should stop early.
	lc := NewLoopController(100, 50*time.Millisecond, 0)
	var count int
	lc.Run(context.Background(), func(_ context.Context, passNum int) {
		count++
		time.Sleep(10 * time.Millisecond)
	})
	if count >= 100 {
		t.Errorf("expected early termination, got %d passes", count)
	}
}

func TestLoopInterval(t *testing.T) {
	lc := NewLoopController(3, 0, 50*time.Millisecond)
	start := time.Now()
	var count int
	lc.Run(context.Background(), func(_ context.Context, passNum int) {
		count++
	})
	elapsed := time.Since(start)
	// 3 passes with 50ms interval between first two = ~100ms minimum.
	if elapsed < 80*time.Millisecond {
		t.Errorf("expected interval delay, elapsed = %v", elapsed)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestLoopContextCancel(t *testing.T) {
	lc := NewLoopController(0, 10*time.Second, 0)
	ctx, cancel := context.WithCancel(context.Background())
	var count int
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()
	lc.Run(ctx, func(runCtx context.Context, passNum int) {
		count++
		time.Sleep(10 * time.Millisecond)
	})
	if count > 10 {
		t.Errorf("expected early stop, got %d", count)
	}
}

func TestLoopZeroDurationSinglePass(t *testing.T) {
	// repeat=1, duration=0 — should run exactly once.
	lc := NewLoopController(1, 0, 0)
	var count int
	lc.Run(context.Background(), func(_ context.Context, passNum int) {
		count++
	})
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestLoopIntervalWithCancel(t *testing.T) {
	// Cancel during interval sleep.
	lc := NewLoopController(0, 0, 5*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	var count int
	lc.Run(ctx, func(_ context.Context, passNum int) {
		count++
	})
	// Should run 1 pass, then context times out during interval.
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}
