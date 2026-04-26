package engine

import (
	"context"
	"testing"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
	"github.com/scrutineer/scrutineer/core/coverage"
	"github.com/scrutineer/scrutineer/core/exitcode"
	"github.com/scrutineer/scrutineer/core/reporter"
	"github.com/scrutineer/scrutineer/core/schema"
	"github.com/scrutineer/scrutineer/core/telemetry"
)

func TestNewEngineDefaults(t *testing.T) {
	e := New()

	if e.parallelism != 1 {
		t.Errorf("default parallelism = %d, want 1", e.parallelism)
	}
	if e.registry != nil {
		t.Error("default registry should be nil")
	}
	if e.reporter != nil {
		t.Error("default reporter should be nil")
	}
}

func TestNewEngineWithOptions(t *testing.T) {
	reg := connector.NewRegistry()
	rep := &mockReporter{}
	tw := &mockTelemetryWriter{}
	cov := coverage.NewTracker()

	e := New(
		WithRegistry(reg),
		WithReporter(rep),
		WithTelemetry(tw),
		WithCoverage(cov),
		WithParallelism(4),
	)

	if e.registry != reg {
		t.Error("registry not set")
	}
	if e.reporter == nil {
		t.Error("reporter not set")
	}
	if e.telemetry == nil {
		t.Error("telemetry not set")
	}
	if e.coverage != cov {
		t.Error("coverage not set")
	}
	if e.parallelism != 4 {
		t.Errorf("parallelism = %d, want 4", e.parallelism)
	}
}

func newTestEngine() (*Engine, *mockReporter) {
	mc := newMockConnector("http")
	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })
	rep := &mockReporter{}

	e := New(
		WithRegistry(reg),
		WithReporter(rep),
		WithParallelism(1),
	)
	return e, rep
}

func TestRunSingleSuiteSingleTest(t *testing.T) {
	e, rep := newTestEngine()

	suites := []schema.TestSuite{
		{
			Suite: "auth",
			Tests: []schema.Test{
				{
					Name:      "login",
					Connector: "http",
					Steps: []schema.TestStep{
						{Action: "POST"},
					},
				},
			},
		},
	}

	results := e.Run(context.Background(), suites)

	if len(results) != 1 {
		t.Fatalf("expected 1 suite result, got %d", len(results))
	}
	sr := results[0]
	if sr.Suite != "auth" {
		t.Errorf("Suite = %q, want %q", sr.Suite, "auth")
	}
	if sr.Passed != 1 {
		t.Errorf("Passed = %d, want 1", sr.Passed)
	}
	if sr.Failed != 0 {
		t.Errorf("Failed = %d, want 0", sr.Failed)
	}

	rep.mu.Lock()
	defer rep.mu.Unlock()
	if len(rep.suiteStarts) != 1 {
		t.Errorf("expected 1 suite start, got %d", len(rep.suiteStarts))
	}
	if len(rep.testStarts) != 1 {
		t.Errorf("expected 1 test start, got %d", len(rep.testStarts))
	}
	if len(rep.testEnds) != 1 {
		t.Errorf("expected 1 test end, got %d", len(rep.testEnds))
	}
}

func TestRunMultipleSuites(t *testing.T) {
	e, _ := newTestEngine()

	suites := []schema.TestSuite{
		{
			Suite: "suite-a",
			Tests: []schema.Test{
				{Name: "test-1", Connector: "http", Steps: []schema.TestStep{{Action: "GET"}}},
			},
		},
		{
			Suite: "suite-b",
			Tests: []schema.Test{
				{Name: "test-2", Connector: "http", Steps: []schema.TestStep{{Action: "GET"}}},
				{Name: "test-3", Connector: "http", Steps: []schema.TestStep{{Action: "POST"}}},
			},
		},
	}

	results := e.Run(context.Background(), suites)

	if len(results) != 2 {
		t.Fatalf("expected 2 suite results, got %d", len(results))
	}

	if results[0].Suite != "suite-a" {
		t.Errorf("first suite = %q, want %q", results[0].Suite, "suite-a")
	}
	if results[1].Suite != "suite-b" {
		t.Errorf("second suite = %q, want %q", results[1].Suite, "suite-b")
	}
	if results[1].Passed != 2 {
		t.Errorf("suite-b passed = %d, want 2", results[1].Passed)
	}
}

func TestRunSkippedTest(t *testing.T) {
	e, _ := newTestEngine()

	suites := []schema.TestSuite{
		{
			Suite: "s",
			Tests: []schema.Test{
				{Name: "skipped", Connector: "http", Skip: true, Steps: []schema.TestStep{{Action: "GET"}}},
				{Name: "runs", Connector: "http", Steps: []schema.TestStep{{Action: "GET"}}},
			},
		},
	}

	results := e.Run(context.Background(), suites)

	if results[0].Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", results[0].Skipped)
	}
	if results[0].Passed != 1 {
		t.Errorf("Passed = %d, want 1", results[0].Passed)
	}
	if len(results[0].Results) != 1 {
		t.Errorf("expected 1 result (skipped tests don't produce results), got %d", len(results[0].Results))
	}
}

func TestExitCodeAllPass(t *testing.T) {
	e := New()

	results := []SuiteResult{
		{Passed: 3, Failed: 0},
		{Passed: 2, Failed: 0},
	}

	if code := e.ExitCode(results); code != exitcode.OK {
		t.Errorf("ExitCode = %d, want %d", code, exitcode.OK)
	}
}

func TestExitCodeAnyFail(t *testing.T) {
	e := New()

	results := []SuiteResult{
		{Passed: 3, Failed: 0},
		{Passed: 2, Failed: 1},
	}

	if code := e.ExitCode(results); code != exitcode.TestFailure {
		t.Errorf("ExitCode = %d, want %d", code, exitcode.TestFailure)
	}
}

func TestExitCodeEmpty(t *testing.T) {
	e := New()

	if code := e.ExitCode(nil); code != exitcode.OK {
		t.Errorf("ExitCode for nil = %d, want %d", code, exitcode.OK)
	}

	if code := e.ExitCode([]SuiteResult{}); code != exitcode.OK {
		t.Errorf("ExitCode for empty = %d, want %d", code, exitcode.OK)
	}
}

func TestRunWithParallelism(t *testing.T) {
	mc := newMockConnector("http")
	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })
	rep := &mockReporter{}

	e := New(
		WithRegistry(reg),
		WithReporter(rep),
		WithParallelism(4),
	)

	suites := make([]schema.TestSuite, 8)
	for i := range suites {
		suites[i] = schema.TestSuite{
			Suite: "suite-" + string(rune('a'+i)),
			Tests: []schema.Test{
				{
					Name:      "test",
					Connector: "http",
					Steps:     []schema.TestStep{{Action: "GET"}},
				},
			},
		}
	}

	results := e.Run(context.Background(), suites)

	if len(results) != 8 {
		t.Fatalf("expected 8 results, got %d", len(results))
	}

	totalPassed := 0
	for _, sr := range results {
		totalPassed += sr.Passed
	}
	if totalPassed != 8 {
		t.Errorf("total passed = %d, want 8", totalPassed)
	}
}

func TestRunWithTelemetry(t *testing.T) {
	mc := newMockConnector("http")
	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })
	rep := &mockReporter{}
	tw := &mockTelemetryWriter{}

	e := New(
		WithRegistry(reg),
		WithReporter(rep),
		WithTelemetry(tw),
	)

	suites := []schema.TestSuite{
		{
			Suite: "s",
			Tests: []schema.Test{
				{Name: "t", Connector: "http", Steps: []schema.TestStep{{Action: "GET"}}},
			},
		},
	}

	e.Run(context.Background(), suites)

	tw.mu.Lock()
	defer tw.mu.Unlock()

	if len(tw.records) == 0 {
		t.Error("expected telemetry records")
	}
}

func TestRunWithCoverage(t *testing.T) {
	mc := newMockConnector("http")
	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })
	rep := &mockReporter{}
	cov := coverage.NewTracker()

	e := New(
		WithRegistry(reg),
		WithReporter(rep),
		WithCoverage(cov),
	)

	suites := []schema.TestSuite{
		{
			Suite: "s",
			Tests: []schema.Test{
				{Name: "t", Connector: "http", Steps: []schema.TestStep{{Action: "GET"}}},
			},
		},
	}

	e.Run(context.Background(), suites)

	if cov.TotalPercent() != 100 {
		t.Errorf("coverage = %.1f%%, want 100%%", cov.TotalPercent())
	}
}

func TestRunWithFixtures(t *testing.T) {
	mc := newMockConnector("http")
	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })
	rep := &mockReporter{}

	e := New(
		WithRegistry(reg),
		WithReporter(rep),
	)

	suites := []schema.TestSuite{
		{
			Suite:    "s",
			Fixtures: map[string]any{"base": "http://localhost"},
			Tests: []schema.Test{
				{
					Name:      "t",
					Connector: "http",
					Steps: []schema.TestStep{
						{
							Action:     "GET",
							Parameters: map[string]any{"url": "${fixture.base}/api"},
						},
					},
				},
			},
		},
	}

	results := e.Run(context.Background(), suites)

	if !results[0].Results[0].Passed {
		t.Errorf("expected pass with fixtures, error: %v", results[0].Results[0].Steps[0].Error)
	}
}

func TestRunWithSkippedAndCoverage(t *testing.T) {
	mc := newMockConnector("http")
	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })
	rep := &mockReporter{}
	cov := coverage.NewTracker()

	e := New(
		WithRegistry(reg),
		WithReporter(rep),
		WithCoverage(cov),
	)

	suites := []schema.TestSuite{
		{
			Suite: "s",
			Tests: []schema.Test{
				{Name: "skipped", Connector: "http", Skip: true, Steps: []schema.TestStep{{Action: "GET"}}},
			},
		},
	}

	e.Run(context.Background(), suites)

	results := cov.Results()
	if len(results) == 0 || len(results[0].Tests) == 0 {
		t.Fatal("no coverage data")
	}
	if !results[0].Tests[0].Skipped {
		t.Error("expected test to be marked as skipped")
	}
}

func TestRunWithSkippedAndTelemetry(t *testing.T) {
	mc := newMockConnector("http")
	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })
	rep := &mockReporter{}
	tw := &mockTelemetryWriter{}

	e := New(
		WithRegistry(reg),
		WithReporter(rep),
		WithTelemetry(tw),
	)

	suites := []schema.TestSuite{
		{
			Suite: "s",
			Tests: []schema.Test{
				{Name: "skipped", Connector: "http", Skip: true},
			},
		},
	}

	e.Run(context.Background(), suites)

	tw.mu.Lock()
	defer tw.mu.Unlock()

	hasSkipEvent := false
	for _, rec := range tw.records {
		if rec.EventType == telemetry.TestSkip {
			hasSkipEvent = true
		}
	}
	if !hasSkipEvent {
		t.Error("expected TestSkip telemetry event")
	}
}

func TestSuiteResultSummarise(t *testing.T) {
	sr := SuiteResult{
		Results: []reporter.TestResult{
			{Passed: true, Elapsed: time.Millisecond},
			{Passed: false, Elapsed: time.Millisecond},
			{Passed: true, Elapsed: time.Millisecond},
		},
	}
	sr.summarise()

	if sr.Passed != 2 {
		t.Errorf("Passed = %d, want 2", sr.Passed)
	}
	if sr.Failed != 1 {
		t.Errorf("Failed = %d, want 1", sr.Failed)
	}
}

func TestSuiteResultSummariseEmpty(t *testing.T) {
	sr := SuiteResult{}
	sr.summarise()

	if sr.Passed != 0 || sr.Failed != 0 {
		t.Errorf("expected 0/0, got %d/%d", sr.Passed, sr.Failed)
	}
}
