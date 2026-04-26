// Package loadtest provides a distributed load testing framework with
// configurable concurrency, ramp-up scheduling, and metrics collection.
package loadtest

import "time"

// RampStep represents a point in the ramp-up schedule indicating how many
// workers should be active at a given time offset from the test start.
type RampStep struct {
	At      time.Duration
	Workers int
}

// Schedule computes a linear ramp-up schedule from 1 worker at time 0 to the
// full concurrency at the end of the ramp-up duration. Steps are evenly spaced.
//
// If concurrency is 1, a single step at time 0 with 1 worker is returned.
// If rampUp is 0, a single step at time 0 with full concurrency is returned.
func Schedule(concurrency int, rampUp time.Duration) []RampStep {
	if concurrency <= 0 {
		return nil
	}

	if concurrency == 1 || rampUp <= 0 {
		return []RampStep{{At: 0, Workers: concurrency}}
	}

	steps := make([]RampStep, concurrency)
	for i := range concurrency {
		workers := i + 1
		at := time.Duration(int64(rampUp) * int64(i) / int64(concurrency-1))
		steps[i] = RampStep{At: at, Workers: workers}
	}
	return steps
}
