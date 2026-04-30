package engine

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/scrutineer/scrutineer/core/schema"
)

func TestSequentialOrder(t *testing.T) {
	d := &SequentialDispatcher{}
	var order []int
	d.Dispatch(context.Background(), 5, func(_ context.Context, idx int) {
		order = append(order, idx)
	})
	if len(order) != 5 {
		t.Fatalf("expected 5 items, got %d", len(order))
	}
	for i, v := range order {
		if v != i {
			t.Errorf("order[%d] = %d, want %d", i, v, i)
		}
	}
}

func TestSequentialContextCancel(t *testing.T) {
	d := &SequentialDispatcher{}
	ctx, cancel := context.WithCancel(context.Background())
	var count int
	d.Dispatch(ctx, 100, func(_ context.Context, idx int) {
		count++
		if count == 3 {
			cancel()
		}
	})
	if count > 4 { // may get one more after cancel
		t.Errorf("expected ~3 items, got %d", count)
	}
}

func TestRandomAllItemsRun(t *testing.T) {
	d := &RandomDispatcher{}
	seen := make(map[int]bool)
	d.Dispatch(context.Background(), 10, func(_ context.Context, idx int) {
		seen[idx] = true
	})
	if len(seen) != 10 {
		t.Errorf("expected 10 items, got %d", len(seen))
	}
}

func TestRandomOrderDiffers(t *testing.T) {
	d := &RandomDispatcher{}
	sameOrder := 0
	for trial := 0; trial < 20; trial++ {
		var order []int
		d.Dispatch(context.Background(), 10, func(_ context.Context, idx int) {
			order = append(order, idx)
		})
		sequential := true
		for i, v := range order {
			if v != i {
				sequential = false
				break
			}
		}
		if sequential {
			sameOrder++
		}
	}
	// With 10 items, probability of random == sequential is 1/10! ≈ 0.
	// Allow up to 2 coincidental matches.
	if sameOrder > 2 {
		t.Errorf("random order matched sequential %d/20 times — not random enough", sameOrder)
	}
}

func TestRandomContextCancel(t *testing.T) {
	d := &RandomDispatcher{}
	ctx, cancel := context.WithCancel(context.Background())
	var count int
	d.Dispatch(ctx, 100, func(_ context.Context, idx int) {
		count++
		if count == 5 {
			cancel()
		}
	})
	if count > 6 {
		t.Errorf("expected ~5, got %d", count)
	}
}

func TestConcurrentAllItemsRun(t *testing.T) {
	d := NewConcurrentDispatcher(4)
	var mu sync.Mutex
	seen := make(map[int]bool)
	d.Dispatch(context.Background(), 20, func(_ context.Context, idx int) {
		mu.Lock()
		seen[idx] = true
		mu.Unlock()
	})
	if len(seen) != 20 {
		t.Errorf("expected 20 items, got %d", len(seen))
	}
}

func TestConcurrentBoundsConcurrency(t *testing.T) {
	maxConcurrency := 3
	d := NewConcurrentDispatcher(maxConcurrency)
	var active atomic.Int32
	var maxSeen atomic.Int32

	d.Dispatch(context.Background(), 50, func(_ context.Context, idx int) {
		cur := active.Add(1)
		for {
			old := maxSeen.Load()
			if cur <= old || maxSeen.CompareAndSwap(old, cur) {
				break
			}
		}
		// Small work to allow goroutine overlap.
		for i := 0; i < 1000; i++ {
			_ = i
		}
		active.Add(-1)
	})

	if maxSeen.Load() > int32(maxConcurrency) {
		t.Errorf("max concurrent = %d, want <= %d", maxSeen.Load(), maxConcurrency)
	}
}

func TestConcurrentContextCancel(t *testing.T) {
	d := NewConcurrentDispatcher(2)
	ctx, cancel := context.WithCancel(context.Background())
	var count atomic.Int32
	d.Dispatch(ctx, 1000, func(_ context.Context, idx int) {
		if count.Add(1) == 5 {
			cancel()
		}
	})
	// Should stop much sooner than 1000.
	if count.Load() > 50 {
		t.Errorf("expected early stop, got %d", count.Load())
	}
}

func TestConcurrentDefaultConcurrency(t *testing.T) {
	d := NewConcurrentDispatcher(0) // should default to 1
	if d.concurrency != 1 {
		t.Errorf("concurrency = %d, want 1", d.concurrency)
	}
}

func TestWeightedDistribution(t *testing.T) {
	weights := []int{8, 2}
	d := NewWeightedDispatcher(weights)
	counts := make(map[int]int)
	d.Dispatch(context.Background(), 10000, func(_ context.Context, idx int) {
		counts[idx]++
	})
	// Item 0 (weight 8) should be selected ~80% of the time.
	ratio := float64(counts[0]) / 10000.0
	if ratio < 0.7 || ratio > 0.9 {
		t.Errorf("item 0 selected %.1f%% (expected ~80%%)", ratio*100)
	}
}

func TestWeightedZeroWeight(t *testing.T) {
	weights := []int{0, 10}
	d := NewWeightedDispatcher(weights)
	counts := make(map[int]int)
	d.Dispatch(context.Background(), 1000, func(_ context.Context, idx int) {
		counts[idx]++
	})
	if counts[0] > 0 {
		t.Errorf("item with weight 0 was selected %d times", counts[0])
	}
}

func TestWeightedAllZero(t *testing.T) {
	weights := []int{0, 0}
	d := NewWeightedDispatcher(weights)
	var count int
	d.Dispatch(context.Background(), 10, func(_ context.Context, idx int) {
		count++
	})
	// Total weight is 0, so nothing should run.
	if count != 0 {
		t.Errorf("expected 0 items, got %d", count)
	}
}

func TestWeightedContextCancel(t *testing.T) {
	d := NewWeightedDispatcher([]int{5, 5})
	ctx, cancel := context.WithCancel(context.Background())
	var count int
	d.Dispatch(ctx, 10000, func(_ context.Context, idx int) {
		count++
		if count == 10 {
			cancel()
		}
	})
	if count > 11 {
		t.Errorf("expected ~10, got %d", count)
	}
}

func TestNewDispatcherSequential(t *testing.T) {
	d := NewDispatcher(schema.ModeSequential, 0, nil)
	if _, ok := d.(*SequentialDispatcher); !ok {
		t.Errorf("expected SequentialDispatcher, got %T", d)
	}
}

func TestNewDispatcherRandom(t *testing.T) {
	d := NewDispatcher(schema.ModeRandom, 0, nil)
	if _, ok := d.(*RandomDispatcher); !ok {
		t.Errorf("expected RandomDispatcher, got %T", d)
	}
}

func TestNewDispatcherConcurrent(t *testing.T) {
	d := NewDispatcher(schema.ModeConcurrent, 10, nil)
	cd, ok := d.(*ConcurrentDispatcher)
	if !ok {
		t.Fatalf("expected ConcurrentDispatcher, got %T", d)
	}
	if cd.concurrency != 10 {
		t.Errorf("concurrency = %d, want 10", cd.concurrency)
	}
}

func TestNewDispatcherWeighted(t *testing.T) {
	d := NewDispatcher(schema.ModeWeighted, 0, []int{5, 5})
	if _, ok := d.(*WeightedDispatcher); !ok {
		t.Errorf("expected WeightedDispatcher, got %T", d)
	}
}

func TestNewDispatcherDefault(t *testing.T) {
	d := NewDispatcher("", 0, nil)
	if _, ok := d.(*SequentialDispatcher); !ok {
		t.Errorf("expected SequentialDispatcher for empty mode, got %T", d)
	}
}
