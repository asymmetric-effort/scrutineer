package engine

import "sync"

// Pool runs tests in parallel with bounded concurrency.
type Pool struct {
	size int
}

// NewPool creates a Pool with the given concurrency limit.
// If size is less than 1, it defaults to 1.
func NewPool(size int) *Pool {
	if size < 1 {
		size = 1
	}
	return &Pool{size: size}
}

// Run executes count functions with bounded parallelism.
// Each function receives its index. Run blocks until all functions complete.
func (p *Pool) Run(count int, fn func(index int)) {
	if count <= 0 {
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, p.size)

	for i := 0; i < count; i++ {
		wg.Add(1)
		sem <- struct{}{} // acquire slot
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }() // release slot
			fn(idx)
		}(i)
	}

	wg.Wait()
}
