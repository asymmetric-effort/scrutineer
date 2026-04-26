package loadtest

import (
	"context"
	"time"
)

// Config defines the parameters for a load test run.
type Config struct {
	Concurrency int           // number of concurrent virtual users
	Duration    time.Duration // total test duration
	RampUp      time.Duration // time to reach full concurrency
}

// Orchestrator coordinates a load test run: it manages ramp-up scheduling,
// worker pools, metrics collection, and result aggregation.
type Orchestrator struct {
	cfg Config
}

// NewOrchestrator creates a new load test orchestrator with the given config.
func NewOrchestrator(cfg Config) *Orchestrator {
	return &Orchestrator{cfg: cfg}
}

// Run executes the load test by calling workFn concurrently according to the
// configured concurrency, ramp-up, and duration. It returns aggregated results.
func (o *Orchestrator) Run(ctx context.Context, workFn func(ctx context.Context) error) *Results {
	metrics := NewMetrics()
	results := make(chan WorkerResult, o.cfg.Concurrency*2)

	// Create a context bounded by the test duration.
	testCtx, testCancel := context.WithTimeout(ctx, o.cfg.Duration)
	defer testCancel()

	// Compute ramp-up schedule.
	steps := Schedule(o.cfg.Concurrency, o.cfg.RampUp)

	// Collect results in background.
	errorSet := make(map[string]struct{})
	done := make(chan struct{})
	go func() {
		for r := range results {
			metrics.Record(r.Elapsed, r.Error)
			if r.Error != nil {
				errorSet[r.Error.Error()] = struct{}{}
			}
		}
		close(done)
	}()

	// Launch workers according to ramp-up schedule.
	var pools []*Pool
	activeWorkers := 0

	for i, step := range steps {
		// Wait until it's time for this step (relative to test start).
		if step.At > 0 {
			timer := time.NewTimer(step.At - elapsed(metrics.startTime))
			select {
			case <-testCtx.Done():
				timer.Stop()
				goto cleanup
			case <-timer.C:
				timer.Stop()
			}
		}

		// Start additional workers to reach the target count.
		newWorkers := step.Workers - activeWorkers
		if newWorkers > 0 {
			pool := NewPool(newWorkers)
			pool.Start(testCtx, workFn, results)
			pools = append(pools, pool)
			activeWorkers = step.Workers
		}

		_ = i // suppress unused warning in some Go versions
	}

	// Wait for the test duration to expire.
	<-testCtx.Done()

cleanup:
	// Stop all pools.
	for _, pool := range pools {
		pool.Stop()
	}
	close(results)
	<-done

	// Build unique error list.
	var errs []string
	for e := range errorSet {
		errs = append(errs, e)
	}

	snap := metrics.Snapshot()

	return &Results{
		Config:  o.cfg,
		Metrics: snap,
		Errors:  errs,
	}
}

// elapsed returns the time since a given start time. Helper used for
// computing how long to wait before the next ramp step.
func elapsed(start time.Time) time.Duration {
	return time.Since(start)
}
