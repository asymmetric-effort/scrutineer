package loadtest

import (
	"strings"
	"testing"
	"time"
)

func TestResults_Summary_Format(t *testing.T) {
	r := &Results{
		Config: Config{
			Concurrency: 10,
			Duration:    30 * time.Second,
			RampUp:      5 * time.Second,
		},
		Metrics: MetricsSnapshot{
			TotalRequests:  1000,
			SuccessCount:   950,
			ErrorCount:     50,
			MeanLatency:    50 * time.Millisecond,
			P50Latency:     45 * time.Millisecond,
			P95Latency:     120 * time.Millisecond,
			P99Latency:     200 * time.Millisecond,
			MinLatency:     5 * time.Millisecond,
			MaxLatency:     500 * time.Millisecond,
			RequestsPerSec: 33.33,
		},
		Errors: []string{"timeout", "connection refused"},
	}

	summary := r.Summary()

	// Check header.
	if !strings.Contains(summary, "Load Test Results") {
		t.Error("missing header")
	}

	// Check config values.
	if !strings.Contains(summary, "30s") {
		t.Error("missing duration")
	}
	if !strings.Contains(summary, "10") {
		t.Error("missing concurrency")
	}
	if !strings.Contains(summary, "5s") {
		t.Error("missing ramp-up")
	}

	// Check metrics.
	if !strings.Contains(summary, "1000") {
		t.Error("missing total requests")
	}
	if !strings.Contains(summary, "950") {
		t.Error("missing success count")
	}
	if !strings.Contains(summary, "50") {
		t.Error("missing error count")
	}
	if !strings.Contains(summary, "33.33") {
		t.Error("missing rps")
	}

	// Check latency values.
	if !strings.Contains(summary, "50ms") {
		t.Error("missing mean latency")
	}
	if !strings.Contains(summary, "45ms") {
		t.Error("missing P50")
	}
	if !strings.Contains(summary, "120ms") {
		t.Error("missing P95")
	}
	if !strings.Contains(summary, "200ms") {
		t.Error("missing P99")
	}
	if !strings.Contains(summary, "5ms") {
		t.Error("missing min latency")
	}
	if !strings.Contains(summary, "500ms") {
		t.Error("missing max latency")
	}

	// Check errors.
	if !strings.Contains(summary, "timeout") {
		t.Error("missing error message 'timeout'")
	}
	if !strings.Contains(summary, "connection refused") {
		t.Error("missing error message 'connection refused'")
	}
	if !strings.Contains(summary, "Unique Errors") {
		t.Error("missing errors section header")
	}
}

func TestResults_Summary_ZeroMetrics(t *testing.T) {
	r := &Results{
		Config:  Config{},
		Metrics: MetricsSnapshot{},
	}

	summary := r.Summary()

	if !strings.Contains(summary, "Load Test Results") {
		t.Error("missing header")
	}
	if !strings.Contains(summary, "Total Requests:  0") {
		t.Error("missing zero total")
	}
	// Should not have errors section.
	if strings.Contains(summary, "Unique Errors") {
		t.Error("should not have errors section with no errors")
	}
}

func TestResults_Summary_NoErrors(t *testing.T) {
	r := &Results{
		Config: Config{
			Concurrency: 5,
			Duration:    time.Second,
		},
		Metrics: MetricsSnapshot{
			TotalRequests:  100,
			SuccessCount:   100,
			RequestsPerSec: 100.0,
		},
	}

	summary := r.Summary()
	if strings.Contains(summary, "Unique Errors") {
		t.Error("should not show errors section when no errors")
	}
}

func TestResults_Summary_SingleError(t *testing.T) {
	r := &Results{
		Config: Config{
			Concurrency: 1,
			Duration:    time.Second,
		},
		Metrics: MetricsSnapshot{
			TotalRequests: 10,
			ErrorCount:    1,
			SuccessCount:  9,
		},
		Errors: []string{"test error"},
	}

	summary := r.Summary()
	if !strings.Contains(summary, "test error") {
		t.Error("missing error in summary")
	}
}
