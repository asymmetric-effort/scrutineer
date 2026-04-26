package loadtest

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMetrics_RecordSuccess(t *testing.T) {
	m := NewMetrics()
	m.Record(100*time.Millisecond, nil)
	m.Record(200*time.Millisecond, nil)

	snap := m.Snapshot()

	if snap.TotalRequests != 2 {
		t.Errorf("expected 2 total requests, got %d", snap.TotalRequests)
	}
	if snap.SuccessCount != 2 {
		t.Errorf("expected 2 successes, got %d", snap.SuccessCount)
	}
	if snap.ErrorCount != 0 {
		t.Errorf("expected 0 errors, got %d", snap.ErrorCount)
	}
}

func TestMetrics_RecordErrors(t *testing.T) {
	m := NewMetrics()
	m.Record(100*time.Millisecond, fmt.Errorf("fail"))
	m.Record(200*time.Millisecond, nil)
	m.Record(300*time.Millisecond, fmt.Errorf("fail2"))

	snap := m.Snapshot()

	if snap.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", snap.TotalRequests)
	}
	if snap.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", snap.SuccessCount)
	}
	if snap.ErrorCount != 2 {
		t.Errorf("expected 2 errors, got %d", snap.ErrorCount)
	}
}

func TestMetrics_Percentiles_KnownData(t *testing.T) {
	m := NewMetrics()

	// Record 100 values: 1ms, 2ms, ..., 100ms.
	for i := 1; i <= 100; i++ {
		m.Record(time.Duration(i)*time.Millisecond, nil)
	}

	snap := m.Snapshot()

	// Mean: sum(1..100)ms / 100 = 5050ms / 100 = 50.5ms.
	expectedMean := 50*time.Millisecond + 500*time.Microsecond
	if snap.MeanLatency != expectedMean {
		t.Errorf("expected mean %s, got %s", expectedMean, snap.MeanLatency)
	}

	// P50 should be ~50.5ms.
	if snap.P50Latency < 50*time.Millisecond || snap.P50Latency > 51*time.Millisecond {
		t.Errorf("P50 %s not in expected range [50ms, 51ms]", snap.P50Latency)
	}

	// P95 should be ~95.05ms.
	if snap.P95Latency < 95*time.Millisecond || snap.P95Latency > 96*time.Millisecond {
		t.Errorf("P95 %s not in expected range [95ms, 96ms]", snap.P95Latency)
	}

	// P99 should be ~99.01ms.
	if snap.P99Latency < 99*time.Millisecond || snap.P99Latency > 100*time.Millisecond {
		t.Errorf("P99 %s not in expected range [99ms, 100ms]", snap.P99Latency)
	}

	if snap.MinLatency != time.Millisecond {
		t.Errorf("expected min 1ms, got %s", snap.MinLatency)
	}
	if snap.MaxLatency != 100*time.Millisecond {
		t.Errorf("expected max 100ms, got %s", snap.MaxLatency)
	}
}

func TestMetrics_ConcurrentRecording(t *testing.T) {
	m := NewMetrics()
	var wg sync.WaitGroup

	count := 1000
	wg.Add(count)
	for i := range count {
		go func(i int) {
			defer wg.Done()
			var err error
			if i%10 == 0 {
				err = fmt.Errorf("err")
			}
			m.Record(time.Duration(i)*time.Microsecond, err)
		}(i)
	}
	wg.Wait()

	snap := m.Snapshot()
	if snap.TotalRequests != int64(count) {
		t.Errorf("expected %d total requests, got %d", count, snap.TotalRequests)
	}
	if snap.ErrorCount != int64(count/10) {
		t.Errorf("expected %d errors, got %d", count/10, snap.ErrorCount)
	}
	if snap.SuccessCount != int64(count-count/10) {
		t.Errorf("expected %d successes, got %d", count-count/10, snap.SuccessCount)
	}
}

func TestMetrics_RequestsPerSec(t *testing.T) {
	m := NewMetrics()
	// Backdate start time by 1 second.
	m.mu.Lock()
	m.startTime = time.Now().Add(-time.Second)
	m.mu.Unlock()

	for range 100 {
		m.Record(time.Millisecond, nil)
	}

	snap := m.Snapshot()

	// Should be approximately 100 rps (within a reasonable tolerance).
	if snap.RequestsPerSec < 50 || snap.RequestsPerSec > 200 {
		t.Errorf("expected ~100 rps, got %.2f", snap.RequestsPerSec)
	}
}

func TestMetrics_Empty(t *testing.T) {
	m := NewMetrics()
	snap := m.Snapshot()

	if snap.TotalRequests != 0 {
		t.Errorf("expected 0 total requests, got %d", snap.TotalRequests)
	}
	if snap.SuccessCount != 0 {
		t.Errorf("expected 0 successes, got %d", snap.SuccessCount)
	}
	if snap.ErrorCount != 0 {
		t.Errorf("expected 0 errors, got %d", snap.ErrorCount)
	}
	if snap.RequestsPerSec != 0 {
		t.Errorf("expected 0 rps, got %.2f", snap.RequestsPerSec)
	}
	if snap.MeanLatency != 0 {
		t.Errorf("expected 0 mean latency, got %s", snap.MeanLatency)
	}
}

func TestMetrics_SingleRecord(t *testing.T) {
	m := NewMetrics()
	m.Record(42*time.Millisecond, nil)

	snap := m.Snapshot()
	if snap.MeanLatency != 42*time.Millisecond {
		t.Errorf("expected mean 42ms, got %s", snap.MeanLatency)
	}
	if snap.P50Latency != 42*time.Millisecond {
		t.Errorf("expected P50 42ms, got %s", snap.P50Latency)
	}
	if snap.P95Latency != 42*time.Millisecond {
		t.Errorf("expected P95 42ms, got %s", snap.P95Latency)
	}
	if snap.P99Latency != 42*time.Millisecond {
		t.Errorf("expected P99 42ms, got %s", snap.P99Latency)
	}
	if snap.MinLatency != 42*time.Millisecond {
		t.Errorf("expected min 42ms, got %s", snap.MinLatency)
	}
	if snap.MaxLatency != 42*time.Millisecond {
		t.Errorf("expected max 42ms, got %s", snap.MaxLatency)
	}
}

func TestPercentile_EmptySlice(t *testing.T) {
	result := percentile(nil, 50)
	if result != 0 {
		t.Errorf("expected 0, got %s", result)
	}
}

func TestMetrics_StartTimeSet(t *testing.T) {
	before := time.Now()
	m := NewMetrics()
	after := time.Now()

	snap := m.Snapshot()
	if snap.StartTime.Before(before) || snap.StartTime.After(after) {
		t.Errorf("start time %s not between %s and %s", snap.StartTime, before, after)
	}
}

func TestMetrics_ElapsedTime(t *testing.T) {
	m := NewMetrics()
	m.mu.Lock()
	m.startTime = time.Now().Add(-500 * time.Millisecond)
	m.mu.Unlock()

	snap := m.Snapshot()
	if snap.ElapsedTime < 400*time.Millisecond || snap.ElapsedTime > 700*time.Millisecond {
		t.Errorf("elapsed time %s not in expected range", snap.ElapsedTime)
	}
}
