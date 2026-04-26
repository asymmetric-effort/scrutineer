package engine

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolRunSequential(t *testing.T) {
	p := NewPool(1)
	results := make([]int, 5)

	p.Run(5, func(index int) {
		results[index] = index * 2
	})

	for i, v := range results {
		if v != i*2 {
			t.Errorf("results[%d] = %d, want %d", i, v, i*2)
		}
	}
}

func TestPoolRunSizeGreaterThanCount(t *testing.T) {
	p := NewPool(10)
	results := make([]int, 3)

	p.Run(3, func(index int) {
		results[index] = index + 1
	})

	for i, v := range results {
		if v != i+1 {
			t.Errorf("results[%d] = %d, want %d", i, v, i+1)
		}
	}
}

func TestPoolRunSizeLessThanCount(t *testing.T) {
	p := NewPool(2)
	var mu sync.Mutex
	results := make([]bool, 10)

	p.Run(10, func(index int) {
		mu.Lock()
		results[index] = true
		mu.Unlock()
	})

	for i, v := range results {
		if !v {
			t.Errorf("results[%d] = false, expected true", i)
		}
	}
}

func TestPoolRunAllFunctionsExecute(t *testing.T) {
	p := NewPool(3)
	var count atomic.Int64

	p.Run(100, func(_ int) {
		count.Add(1)
	})

	if count.Load() != 100 {
		t.Errorf("executed %d functions, want 100", count.Load())
	}
}

func TestPoolRunBoundedConcurrency(t *testing.T) {
	maxConcurrency := 3
	p := NewPool(maxConcurrency)

	var current atomic.Int64
	var peak atomic.Int64

	p.Run(20, func(_ int) {
		c := current.Add(1)

		// Track peak concurrency.
		for {
			old := peak.Load()
			if c <= old || peak.CompareAndSwap(old, c) {
				break
			}
		}

		// Do some work to allow other goroutines to run.
		time.Sleep(time.Millisecond)

		current.Add(-1)
	})

	observed := peak.Load()
	if observed > int64(maxConcurrency) {
		t.Errorf("peak concurrency = %d, exceeds pool size %d", observed, maxConcurrency)
	}
	if observed < 1 {
		t.Errorf("peak concurrency = %d, expected at least 1", observed)
	}
}

func TestPoolRunZeroCount(t *testing.T) {
	p := NewPool(5)

	// Should not panic or block.
	p.Run(0, func(_ int) {
		t.Error("should not be called with count 0")
	})
}

func TestPoolRunNegativeCount(t *testing.T) {
	p := NewPool(5)

	// Should not panic or block.
	p.Run(-1, func(_ int) {
		t.Error("should not be called with negative count")
	})
}

func TestNewPoolMinSize(t *testing.T) {
	p := NewPool(0)
	if p.size != 1 {
		t.Errorf("Pool(0).size = %d, want 1", p.size)
	}

	p = NewPool(-5)
	if p.size != 1 {
		t.Errorf("Pool(-5).size = %d, want 1", p.size)
	}

	p = NewPool(1)
	if p.size != 1 {
		t.Errorf("Pool(1).size = %d, want 1", p.size)
	}
}

func TestPoolRunSingleItem(t *testing.T) {
	p := NewPool(4)
	var called bool

	p.Run(1, func(index int) {
		if index != 0 {
			t.Errorf("index = %d, want 0", index)
		}
		called = true
	})

	if !called {
		t.Error("function was not called")
	}
}

func TestPoolRunOrderIndependence(t *testing.T) {
	// With parallelism, execution order is non-deterministic.
	// But all indices must execute exactly once.
	p := NewPool(4)
	seen := make([]atomic.Bool, 50)

	p.Run(50, func(index int) {
		if !seen[index].CompareAndSwap(false, true) {
			t.Errorf("index %d executed more than once", index)
		}
	})

	for i := range seen {
		if !seen[i].Load() {
			t.Errorf("index %d was not executed", i)
		}
	}
}
