package loadtest

import "fmt"

// Results holds the final output of a load test run.
type Results struct {
	Config  Config
	Metrics MetricsSnapshot
	Errors  []string // unique error messages
}

// Summary returns a human-readable summary of the load test results.
func (r *Results) Summary() string {
	s := "Load Test Results\n"
	s += fmt.Sprintf("  Duration:        %s\n", r.Config.Duration)
	s += fmt.Sprintf("  Concurrency:     %d\n", r.Config.Concurrency)
	s += fmt.Sprintf("  Ramp-up:         %s\n", r.Config.RampUp)
	s += "\n"
	s += fmt.Sprintf("  Total Requests:  %d\n", r.Metrics.TotalRequests)
	s += fmt.Sprintf("  Successes:       %d\n", r.Metrics.SuccessCount)
	s += fmt.Sprintf("  Errors:          %d\n", r.Metrics.ErrorCount)
	s += fmt.Sprintf("  Requests/sec:    %.2f\n", r.Metrics.RequestsPerSec)
	s += "\n"
	s += fmt.Sprintf("  Mean Latency:    %s\n", r.Metrics.MeanLatency)
	s += fmt.Sprintf("  P50 Latency:     %s\n", r.Metrics.P50Latency)
	s += fmt.Sprintf("  P95 Latency:     %s\n", r.Metrics.P95Latency)
	s += fmt.Sprintf("  P99 Latency:     %s\n", r.Metrics.P99Latency)
	s += fmt.Sprintf("  Min Latency:     %s\n", r.Metrics.MinLatency)
	s += fmt.Sprintf("  Max Latency:     %s\n", r.Metrics.MaxLatency)

	if len(r.Errors) > 0 {
		s += "\n  Unique Errors:\n"
		for _, e := range r.Errors {
			s += fmt.Sprintf("    - %s\n", e)
		}
	}

	return s
}
