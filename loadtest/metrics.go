package loadtest

import (
	"math"
	"slices"
	"sync"
	"time"
)

// MetricsSnapshot is a point-in-time summary of collected metrics.
type MetricsSnapshot struct {
	TotalRequests  int64
	SuccessCount   int64
	ErrorCount     int64
	MeanLatency    time.Duration
	P50Latency     time.Duration
	P95Latency     time.Duration
	P99Latency     time.Duration
	MinLatency     time.Duration
	MaxLatency     time.Duration
	RequestsPerSec float64
	StartTime      time.Time
	ElapsedTime    time.Duration
}

// Metrics collects timing and error data from load test executions.
// It is safe for concurrent use.
type Metrics struct {
	mu        sync.Mutex
	latencies []time.Duration
	errors    int64
	startTime time.Time
}

// NewMetrics returns a new Metrics instance with the start time set to now.
func NewMetrics() *Metrics {
	return &Metrics{
		startTime: time.Now(),
	}
}

// Record records a single request result. If err is non-nil it is counted
// as an error; the latency is always recorded regardless.
func (m *Metrics) Record(elapsed time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.latencies = append(m.latencies, elapsed)
	if err != nil {
		m.errors++
	}
}

// Snapshot returns a point-in-time copy of the current metrics.
func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(m.startTime)
	total := int64(len(m.latencies))

	snap := MetricsSnapshot{
		TotalRequests: total,
		SuccessCount:  total - m.errors,
		ErrorCount:    m.errors,
		StartTime:     m.startTime,
		ElapsedTime:   elapsed,
	}

	if total == 0 {
		return snap
	}

	// Compute requests per second.
	if elapsed > 0 {
		snap.RequestsPerSec = float64(total) / elapsed.Seconds()
	}

	// Sort a copy for percentile calculations.
	sorted := make([]time.Duration, len(m.latencies))
	copy(sorted, m.latencies)
	slices.Sort(sorted)

	snap.MinLatency = sorted[0]
	snap.MaxLatency = sorted[len(sorted)-1]

	// Mean.
	var sum int64
	for _, d := range sorted {
		sum += int64(d)
	}
	snap.MeanLatency = time.Duration(sum / total)

	// Percentiles.
	snap.P50Latency = percentile(sorted, 50)
	snap.P95Latency = percentile(sorted, 95)
	snap.P99Latency = percentile(sorted, 99)

	return snap
}

// percentile returns the p-th percentile from a sorted slice of durations.
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	idx := p / 100.0 * float64(len(sorted)-1)
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))
	if lower == upper {
		return sorted[lower]
	}
	frac := idx - float64(lower)
	return time.Duration(float64(sorted[lower])*(1-frac) + float64(sorted[upper])*frac)
}
