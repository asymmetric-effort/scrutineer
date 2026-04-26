package loadtest

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_StartsCorrectNumberOfWorkers(t *testing.T) {
	var count atomic.Int32
	var maxSeen atomic.Int32

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	results := make(chan WorkerResult, 100)
	pool := NewPool(5)

	pool.Start(ctx, func(ctx context.Context) error {
		c := count.Add(1)
		for {
			old := maxSeen.Load()
			if c <= old || maxSeen.CompareAndSwap(old, c) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		count.Add(-1)
		return nil
	}, results)

	// Wait a bit for workers to start.
	time.Sleep(200 * time.Millisecond)
	pool.Stop()
	close(results)

	// Drain results.
	var got int
	for range results {
		got++
	}

	if got == 0 {
		t.Error("expected some results, got 0")
	}

	if maxSeen.Load() > 5 {
		t.Errorf("more than 5 concurrent workers observed: %d", maxSeen.Load())
	}
}

func TestPool_StopWaitsForCompletion(t *testing.T) {
	var completed atomic.Int32

	ctx := context.Background()
	results := make(chan WorkerResult, 100)
	pool := NewPool(3)

	pool.Start(ctx, func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		completed.Add(1)
		return nil
	}, results)

	// Let workers run a bit.
	time.Sleep(100 * time.Millisecond)
	pool.Stop()
	close(results)

	// After Stop returns, completed should have a value (workers finished).
	if completed.Load() == 0 {
		t.Error("expected some completions after Stop")
	}
}

func TestPool_WorkersRespectContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	results := make(chan WorkerResult, 100)
	pool := NewPool(3)

	var iterations atomic.Int32
	pool.Start(ctx, func(ctx context.Context) error {
		iterations.Add(1)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
		return nil
	}, results)

	time.Sleep(50 * time.Millisecond)
	cancel()

	// Stop should return quickly since context is cancelled.
	done := make(chan struct{})
	go func() {
		pool.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Good, Stop returned.
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not return within timeout after context cancellation")
	}
	close(results)
}

func TestPool_ResultsContainErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	results := make(chan WorkerResult, 100)
	pool := NewPool(2)

	pool.Start(ctx, func(ctx context.Context) error {
		return fmt.Errorf("test error")
	}, results)

	<-ctx.Done()
	pool.Stop()
	close(results)

	var errorCount int
	for r := range results {
		if r.Error != nil {
			errorCount++
		}
	}
	if errorCount == 0 {
		t.Error("expected error results")
	}
}

func TestPool_ResultsHaveElapsedTime(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	results := make(chan WorkerResult, 100)
	pool := NewPool(1)

	pool.Start(ctx, func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}, results)

	<-ctx.Done()
	pool.Stop()
	close(results)

	for r := range results {
		if r.Elapsed < 10*time.Millisecond {
			t.Errorf("expected elapsed >= 10ms, got %s", r.Elapsed)
		}
	}
}

func TestNewPool(t *testing.T) {
	p := NewPool(7)
	if p.size != 7 {
		t.Errorf("expected size 7, got %d", p.size)
	}
}
