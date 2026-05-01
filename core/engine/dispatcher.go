package engine

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	mrand "math/rand/v2"
	"sync"

	"github.com/scrutineer/scrutineer/core/schema"
)

// Dispatcher controls the order and concurrency of test or interaction execution.
type Dispatcher interface {
	// Dispatch runs workFn for each index according to the dispatch strategy.
	// count is the number of items to dispatch.
	// Dispatch blocks until complete or the context is cancelled.
	Dispatch(ctx context.Context, count int, workFn func(ctx context.Context, index int))
}

// SequentialDispatcher runs items in manifest order: 0, 1, ..., count-1.
type SequentialDispatcher struct{}

func (d *SequentialDispatcher) Dispatch(ctx context.Context, count int, workFn func(ctx context.Context, index int)) {
	for i := 0; i < count; i++ {
		if ctx.Err() != nil {
			return
		}
		workFn(ctx, i)
	}
}

// RandomDispatcher shuffles items and runs them sequentially.
type RandomDispatcher struct{}

func (d *RandomDispatcher) Dispatch(ctx context.Context, count int, workFn func(ctx context.Context, index int)) {
	indices := make([]int, count)
	for i := range indices {
		indices[i] = i
	}
	// Fisher-Yates shuffle using crypto/rand for unpredictability.
	for i := count - 1; i > 0; i-- {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		j := int(n.Int64())
		indices[i], indices[j] = indices[j], indices[i]
	}
	for _, idx := range indices {
		if ctx.Err() != nil {
			return
		}
		workFn(ctx, idx)
	}
}

// ConcurrentDispatcher runs items in parallel with bounded concurrency.
// Items from different passes can overlap — there is no pass barrier.
type ConcurrentDispatcher struct {
	concurrency int
}

func NewConcurrentDispatcher(concurrency int) *ConcurrentDispatcher {
	if concurrency < 1 {
		concurrency = 1
	}
	return &ConcurrentDispatcher{concurrency: concurrency}
}

// PanicError records a panic that occurred during concurrent dispatch.
type PanicError struct {
	Index int
	Value any
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic in concurrent dispatch item %d: %v", e.Index, e.Value)
}

func (d *ConcurrentDispatcher) Dispatch(ctx context.Context, count int, workFn func(ctx context.Context, index int)) {
	sem := make(chan struct{}, d.concurrency)
	var wg sync.WaitGroup
	var panicMu sync.Mutex
	var panics []PanicError

	for i := 0; i < count; i++ {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{} // acquire slot
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }() // release slot
			defer func() {
				if r := recover(); r != nil {
					panicMu.Lock()
					panics = append(panics, PanicError{Index: idx, Value: r})
					panicMu.Unlock()
				}
			}()
			if ctx.Err() != nil {
				return
			}
			workFn(ctx, idx)
		}(i)
	}
	wg.Wait()

	// Re-panic with the first captured panic so the caller sees it.
	if len(panics) > 0 {
		panic(&panics[0])
	}
}

// WeightedDispatcher selects items probabilistically based on weights.
// It runs count selections — each pick chooses an item by probability.
type WeightedDispatcher struct {
	weights []int
}

func NewWeightedDispatcher(weights []int) *WeightedDispatcher {
	return &WeightedDispatcher{weights: weights}
}

func (d *WeightedDispatcher) Dispatch(ctx context.Context, count int, workFn func(ctx context.Context, index int)) {
	total := 0
	for _, w := range d.weights {
		total += w
	}
	if total == 0 {
		return
	}

	for i := 0; i < count; i++ {
		if ctx.Err() != nil {
			return
		}
		idx := selectByWeight(d.weights, total)
		workFn(ctx, idx)
	}
}

// selectByWeight picks an index using weighted random selection.
func selectByWeight(weights []int, total int) int {
	r := mrand.IntN(total)
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			return i
		}
	}
	return len(weights) - 1
}

// NewDispatcher creates a Dispatcher from an ExecutionMode and parameters.
func NewDispatcher(mode schema.ExecutionMode, concurrency int, weights []int) Dispatcher {
	switch mode {
	case schema.ModeRandom:
		return &RandomDispatcher{}
	case schema.ModeConcurrent:
		return NewConcurrentDispatcher(concurrency)
	case schema.ModeWeighted:
		return NewWeightedDispatcher(weights)
	default:
		return &SequentialDispatcher{}
	}
}
