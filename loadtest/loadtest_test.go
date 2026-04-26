package loadtest

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestOrchestrator_SimpleWorkFn(t *testing.T) {
	cfg := Config{
		Concurrency: 2,
		Duration:    200 * time.Millisecond,
		RampUp:      0,
	}

	orch := NewOrchestrator(cfg)
	results := orch.Run(context.Background(), func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if results.Metrics.TotalRequests == 0 {
		t.Error("expected some requests")
	}
	if results.Metrics.SuccessCount == 0 {
		t.Error("expected some successes")
	}
	if results.Metrics.ErrorCount != 0 {
		t.Errorf("expected 0 errors, got %d", results.Metrics.ErrorCount)
	}
	if results.Config.Concurrency != cfg.Concurrency {
		t.Errorf("config mismatch: expected concurrency %d, got %d", cfg.Concurrency, results.Config.Concurrency)
	}
}

func TestOrchestrator_ContextCancellation(t *testing.T) {
	cfg := Config{
		Concurrency: 2,
		Duration:    10 * time.Second,
		RampUp:      0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	orch := NewOrchestrator(cfg)

	// Cancel after a short delay.
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	results := orch.Run(ctx, func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	elapsed := time.Since(start)

	// Should finish well before the 10s duration.
	if elapsed > 2*time.Second {
		t.Errorf("expected cancellation to stop test quickly, took %s", elapsed)
	}

	if results.Metrics.TotalRequests == 0 {
		t.Error("expected some requests before cancellation")
	}
}

func TestOrchestrator_DurationExpiry(t *testing.T) {
	cfg := Config{
		Concurrency: 2,
		Duration:    100 * time.Millisecond,
		RampUp:      0,
	}

	orch := NewOrchestrator(cfg)
	start := time.Now()
	results := orch.Run(context.Background(), func(ctx context.Context) error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})
	elapsed := time.Since(start)

	// Should finish close to the configured duration.
	if elapsed > 500*time.Millisecond {
		t.Errorf("test ran too long: %s", elapsed)
	}
	if results.Metrics.TotalRequests == 0 {
		t.Error("expected some requests")
	}
}

func TestOrchestrator_WorkFnErrors(t *testing.T) {
	cfg := Config{
		Concurrency: 2,
		Duration:    200 * time.Millisecond,
		RampUp:      0,
	}

	orch := NewOrchestrator(cfg)
	results := orch.Run(context.Background(), func(ctx context.Context) error {
		return fmt.Errorf("test failure")
	})

	if results.Metrics.ErrorCount == 0 {
		t.Error("expected errors")
	}
	if results.Metrics.SuccessCount != 0 {
		t.Errorf("expected 0 successes, got %d", results.Metrics.SuccessCount)
	}
	if len(results.Errors) == 0 {
		t.Error("expected unique errors list")
	}

	found := false
	for _, e := range results.Errors {
		if e == "test failure" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'test failure' in errors, got %v", results.Errors)
	}
}

func TestOrchestrator_WithRampUp(t *testing.T) {
	cfg := Config{
		Concurrency: 3,
		Duration:    300 * time.Millisecond,
		RampUp:      100 * time.Millisecond,
	}

	orch := NewOrchestrator(cfg)
	var maxConcurrent atomic.Int32
	var current atomic.Int32

	results := orch.Run(context.Background(), func(ctx context.Context) error {
		c := current.Add(1)
		for {
			old := maxConcurrent.Load()
			if c <= old || maxConcurrent.CompareAndSwap(old, c) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		current.Add(-1)
		return nil
	})

	if results.Metrics.TotalRequests == 0 {
		t.Error("expected some requests")
	}
}

func TestNewOrchestrator(t *testing.T) {
	cfg := Config{
		Concurrency: 5,
		Duration:    time.Second,
		RampUp:      100 * time.Millisecond,
	}
	orch := NewOrchestrator(cfg)
	if orch.cfg != cfg {
		t.Error("orchestrator config mismatch")
	}
}

func TestOrchestrator_CancelDuringRampUp(t *testing.T) {
	cfg := Config{
		Concurrency: 20,
		Duration:    10 * time.Second,
		RampUp:      5 * time.Second, // long ramp-up
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel very quickly so we interrupt during ramp-up.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	orch := NewOrchestrator(cfg)
	start := time.Now()
	results := orch.Run(ctx, func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	elapsed := time.Since(start)

	// Should finish well before 5s ramp-up.
	if elapsed > 2*time.Second {
		t.Errorf("expected early exit during ramp-up, took %s", elapsed)
	}
	_ = results
}

func TestOrchestrator_MixedErrors(t *testing.T) {
	cfg := Config{
		Concurrency: 2,
		Duration:    200 * time.Millisecond,
		RampUp:      0,
	}

	var count atomic.Int32
	orch := NewOrchestrator(cfg)
	results := orch.Run(context.Background(), func(ctx context.Context) error {
		n := count.Add(1)
		if n%2 == 0 {
			return fmt.Errorf("even error")
		}
		return nil
	})

	if results.Metrics.TotalRequests == 0 {
		t.Error("expected requests")
	}
	if results.Metrics.SuccessCount == 0 {
		t.Error("expected some successes")
	}
	if results.Metrics.ErrorCount == 0 {
		t.Error("expected some errors")
	}
}
