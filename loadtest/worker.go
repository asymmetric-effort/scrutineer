package loadtest

import (
	"context"
	"sync"
	"time"
)

// WorkerResult captures the outcome of a single work function invocation.
type WorkerResult struct {
	Elapsed time.Duration
	Error   error
}

// Pool manages a set of concurrent workers that execute a work function
// repeatedly until the context is cancelled.
type Pool struct {
	size   int
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// NewPool creates a new worker pool of the given size.
func NewPool(size int) *Pool {
	return &Pool{size: size}
}

// Start launches workers that repeatedly call work and send results on the
// results channel. Workers stop when the context is cancelled.
// The caller is responsible for closing the results channel after calling Stop.
func (p *Pool) Start(ctx context.Context, work func(ctx context.Context) error, results chan<- WorkerResult) {
	var childCtx context.Context
	childCtx, p.cancel = context.WithCancel(ctx)

	p.wg.Add(p.size)
	for range p.size {
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-childCtx.Done():
					return
				default:
				}

				start := time.Now()
				err := work(childCtx)
				elapsed := time.Since(start)

				// Check again after work completes so we don't block
				// on sending when context is already done.
				select {
				case <-childCtx.Done():
					return
				case results <- WorkerResult{Elapsed: elapsed, Error: err}:
				}
			}
		}()
	}
}

// Stop signals all workers to stop and waits for them to finish.
func (p *Pool) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}
