package engine

import (
	"context"
	"time"
)

// LoopController manages repeated execution passes with optional
// duration bounds and inter-pass intervals.
type LoopController struct {
	repeat   int           // 0 = unlimited (requires duration)
	duration time.Duration // 0 = no time bound
	interval time.Duration // delay between passes
}

// NewLoopController creates a LoopController.
func NewLoopController(repeat int, duration, interval time.Duration) *LoopController {
	return &LoopController{
		repeat:   repeat,
		duration: duration,
		interval: interval,
	}
}

// Run executes passFn repeatedly according to repeat/duration/interval.
// passNum starts at 1. Returns when:
//   - repeat count is reached (if repeat > 0)
//   - duration expires (if duration > 0)
//   - context is cancelled
//
// Whichever condition is hit first stops execution.
func (lc *LoopController) Run(ctx context.Context, passFn func(ctx context.Context, passNum int)) {
	var cancel context.CancelFunc
	runCtx := ctx

	if lc.duration > 0 {
		runCtx, cancel = context.WithTimeout(ctx, lc.duration)
		defer cancel()
	}

	passNum := 0
	for {
		if runCtx.Err() != nil {
			return
		}

		passNum++

		// Check repeat bound.
		if lc.repeat > 0 && passNum > lc.repeat {
			return
		}

		passFn(runCtx, passNum)

		// Don't sleep after the last pass or if we're about to exit.
		if lc.repeat > 0 && passNum >= lc.repeat {
			return
		}

		if lc.interval > 0 {
			select {
			case <-runCtx.Done():
				return
			case <-time.After(lc.interval):
			}
		}
	}
}
