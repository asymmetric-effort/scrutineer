package engine

import (
	"context"
	"testing"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
	"github.com/scrutineer/scrutineer/core/coverage"
	"github.com/scrutineer/scrutineer/core/exitcode"
	"github.com/scrutineer/scrutineer/core/expression"
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

// --- Execution block + Interactions integration tests ---

func TestRunSuiteNoExecutionBackwardCompat(t *testing.T) {
	e, rep := newTestEngine()
	suites := []schema.TestSuite{
		{
			Suite: "simple",
			Tests: []schema.Test{
				{
					Name:      "t1",
					Connector: "http",
					Steps:     []schema.TestStep{{Action: "request"}},
				},
			},
		},
	}
	results := e.Run(context.Background(), suites)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Passed != 1 {
		t.Errorf("passed = %d, want 1", results[0].Passed)
	}
	rep.mu.Lock()
	defer rep.mu.Unlock()
	if len(rep.suiteStarts) != 1 {
		t.Errorf("suite starts = %d", len(rep.suiteStarts))
	}
}

func TestRunSuiteSequentialExecution(t *testing.T) {
	e, _ := newTestEngine()
	suites := []schema.TestSuite{
		{
			Suite: "exec",
			Execution: &schema.Execution{
				Mode:   schema.ModeSequential,
				Repeat: 2,
			},
			Tests: []schema.Test{
				{
					Name:      "t1",
					Connector: "http",
					Steps:     []schema.TestStep{{Action: "request"}},
				},
			},
		},
	}
	results := e.Run(context.Background(), suites)
	// 2 passes x 1 test = 2 results
	if results[0].Passed != 2 {
		t.Errorf("passed = %d, want 2", results[0].Passed)
	}
}

func TestRunSuiteWithRepeat(t *testing.T) {
	e, _ := newTestEngine()
	suites := []schema.TestSuite{
		{
			Suite: "repeat",
			Execution: &schema.Execution{
				Mode:   schema.ModeSequential,
				Repeat: 5,
			},
			Tests: []schema.Test{
				{
					Name:      "t1",
					Connector: "http",
					Steps:     []schema.TestStep{{Action: "request"}},
				},
			},
		},
	}
	results := e.Run(context.Background(), suites)
	if results[0].Passed != 5 {
		t.Errorf("passed = %d, want 5", results[0].Passed)
	}
}

func TestRunSuiteWithDuration(t *testing.T) {
	e, _ := newTestEngine()
	suites := []schema.TestSuite{
		{
			Suite: "duration",
			Execution: &schema.Execution{
				Mode:     schema.ModeSequential,
				Duration: "100ms",
			},
			Tests: []schema.Test{
				{
					Name:      "t1",
					Connector: "http",
					Steps:     []schema.TestStep{{Action: "request"}},
				},
			},
		},
	}
	start := time.Now()
	results := e.Run(context.Background(), suites)
	elapsed := time.Since(start)
	if elapsed > 300*time.Millisecond {
		t.Errorf("took too long: %v", elapsed)
	}
	if results[0].Passed < 1 {
		t.Errorf("expected at least 1 pass, got %d", results[0].Passed)
	}
}

func TestRunSuiteWithInteractions(t *testing.T) {
	e, _ := newTestEngine()
	suites := []schema.TestSuite{
		{
			Suite: "interactions",
			Execution: &schema.Execution{
				Mode:   schema.ModeSequential,
				Repeat: 1,
			},
			Interactions: []schema.Interaction{
				{
					Name: "Browse",
					Mode: schema.ModeSequential,
					Tests: []schema.Test{
						{Name: "login", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
						{Name: "browse", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
					},
				},
				{
					Name: "Admin",
					Mode: schema.ModeSequential,
					Tests: []schema.Test{
						{Name: "report", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
					},
				},
			},
		},
	}
	results := e.Run(context.Background(), suites)
	// 3 tests total across 2 interactions
	if results[0].Passed != 3 {
		t.Errorf("passed = %d, want 3", results[0].Passed)
	}
}

func TestRunSuiteInteractionSkip(t *testing.T) {
	e, _ := newTestEngine()
	suites := []schema.TestSuite{
		{
			Suite: "skip",
			Execution: &schema.Execution{
				Mode:   schema.ModeSequential,
				Repeat: 1,
			},
			Interactions: []schema.Interaction{
				{
					Name: "I1",
					Mode: schema.ModeSequential,
					Tests: []schema.Test{
						{Name: "t1", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
						{Name: "t2", Connector: "http", Skip: true, Steps: []schema.TestStep{{Action: "request"}}},
					},
				},
			},
		},
	}
	results := e.Run(context.Background(), suites)
	if results[0].Passed != 1 {
		t.Errorf("passed = %d, want 1", results[0].Passed)
	}
	if results[0].Skipped != 1 {
		t.Errorf("skipped = %d, want 1", results[0].Skipped)
	}
}

func TestRunSuiteRandomExecution(t *testing.T) {
	e, _ := newTestEngine()
	suites := []schema.TestSuite{
		{
			Suite: "random",
			Execution: &schema.Execution{
				Mode:   schema.ModeRandom,
				Repeat: 1,
			},
			Tests: []schema.Test{
				{Name: "t1", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
				{Name: "t2", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
				{Name: "t3", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
			},
		},
	}
	results := e.Run(context.Background(), suites)
	if results[0].Passed != 3 {
		t.Errorf("passed = %d, want 3", results[0].Passed)
	}
}

func TestRunSuiteConcurrentExecution(t *testing.T) {
	e, _ := newTestEngine()
	suites := []schema.TestSuite{
		{
			Suite: "concurrent",
			Execution: &schema.Execution{
				Mode:        schema.ModeConcurrent,
				Concurrency: 5,
				Repeat:      1,
			},
			Tests: []schema.Test{
				{Name: "t1", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
				{Name: "t2", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
				{Name: "t3", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
			},
		},
	}
	results := e.Run(context.Background(), suites)
	if results[0].Passed != 3 {
		t.Errorf("passed = %d, want 3", results[0].Passed)
	}
}

func TestRunSuiteWeightedExecution(t *testing.T) {
	e, _ := newTestEngine()
	suites := []schema.TestSuite{
		{
			Suite: "weighted",
			Execution: &schema.Execution{
				Mode:   schema.ModeWeighted,
				Repeat: 10,
			},
			Interactions: []schema.Interaction{
				{
					Name:   "heavy",
					Weight: 8,
					Mode:   schema.ModeSequential,
					Tests: []schema.Test{
						{Name: "t1", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
					},
				},
				{
					Name:   "light",
					Weight: 2,
					Mode:   schema.ModeSequential,
					Tests: []schema.Test{
						{Name: "t2", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
					},
				},
			},
		},
	}
	results := e.Run(context.Background(), suites)
	// 10 passes, each selecting one interaction = 10 test results.
	if results[0].Passed != 10 {
		t.Errorf("passed = %d, want 10", results[0].Passed)
	}
}

func TestWithConnectorConfigs(t *testing.T) {
	configs := map[string]map[string]any{
		"http": {"base_url": "http://localhost:8080"},
	}
	e := New(WithConnectorConfigs(configs))
	if e.connectorConfigs == nil {
		t.Fatal("connectorConfigs not set")
	}
	if e.connectorConfigs["http"]["base_url"] != "http://localhost:8080" {
		t.Error("config value not preserved")
	}
}

func TestWithExpressionRegistry(t *testing.T) {
	reg := expression.NewRegistry()
	e := New(WithExpressionRegistry(reg))
	if e.exprRegistry != reg {
		t.Error("exprRegistry not set")
	}
}

func TestRunSuiteSimpleWithExpressionRegistry(t *testing.T) {
	mc := newMockConnector("http")
	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })
	rep := &mockReporter{}
	exprReg := expression.NewRegistry()

	e := New(
		WithRegistry(reg),
		WithReporter(rep),
		WithExpressionRegistry(exprReg),
	)

	suites := []schema.TestSuite{
		{
			Suite: "s",
			Tests: []schema.Test{
				{Name: "t", Connector: "http", Steps: []schema.TestStep{{Action: "GET"}}},
			},
		},
	}

	results := e.Run(context.Background(), suites)
	if results[0].Passed != 1 {
		t.Errorf("passed = %d, want 1", results[0].Passed)
	}
}

func TestRunSuiteSimpleWithConnectorConfigs(t *testing.T) {
	mc := newMockConnector("http")
	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })
	rep := &mockReporter{}

	e := New(
		WithRegistry(reg),
		WithReporter(rep),
		WithConnectorConfigs(map[string]map[string]any{
			"http": {"base_url": "http://localhost"},
		}),
	)

	suites := []schema.TestSuite{
		{
			Suite: "s",
			Tests: []schema.Test{
				{Name: "t", Connector: "http", Steps: []schema.TestStep{{Action: "GET"}}},
			},
		},
	}

	results := e.Run(context.Background(), suites)
	if results[0].Passed != 1 {
		t.Errorf("passed = %d, want 1", results[0].Passed)
	}
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if mc.lastConfig["base_url"] != "http://localhost" {
		t.Errorf("connector config not passed, got %v", mc.lastConfig)
	}
}

func TestRunSuiteWithExecutionTelemetry(t *testing.T) {
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
			Suite: "telemetry-exec",
			Execution: &schema.Execution{
				Mode:   schema.ModeSequential,
				Repeat: 1,
			},
			Tests: []schema.Test{
				{Name: "t1", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
			},
		},
	}

	e.Run(context.Background(), suites)

	tw.mu.Lock()
	defer tw.mu.Unlock()

	hasSuiteStart := false
	hasSuiteEnd := false
	for _, rec := range tw.records {
		if rec.EventType == telemetry.SuiteStart {
			hasSuiteStart = true
		}
		if rec.EventType == telemetry.SuiteEnd {
			hasSuiteEnd = true
		}
	}
	if !hasSuiteStart {
		t.Error("expected SuiteStart telemetry event in execution path")
	}
	if !hasSuiteEnd {
		t.Error("expected SuiteEnd telemetry event in execution path")
	}
}

func TestRunSuiteWithExecutionRepeatZeroDurationZeroFallback(t *testing.T) {
	mc := newMockConnector("http")
	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })
	rep := &mockReporter{}

	e := New(
		WithRegistry(reg),
		WithReporter(rep),
	)

	// Execution with repeat=0 and no duration — should fallback to repeat=1.
	suites := []schema.TestSuite{
		{
			Suite: "fallback",
			Execution: &schema.Execution{
				Mode:   schema.ModeSequential,
				Repeat: 0,
			},
			Tests: []schema.Test{
				{Name: "t1", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
			},
		},
	}

	results := e.Run(context.Background(), suites)
	if results[0].Passed != 1 {
		t.Errorf("passed = %d, want 1 (safety fallback)", results[0].Passed)
	}
}

func TestRunInteractionWithExpressionRegistry(t *testing.T) {
	mc := newMockConnector("http")
	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })
	rep := &mockReporter{}
	exprReg := expression.NewRegistry()

	e := New(
		WithRegistry(reg),
		WithReporter(rep),
		WithExpressionRegistry(exprReg),
	)

	suites := []schema.TestSuite{
		{
			Suite: "expr-inter",
			Execution: &schema.Execution{
				Mode:   schema.ModeSequential,
				Repeat: 1,
			},
			Interactions: []schema.Interaction{
				{
					Name: "I1",
					Mode: schema.ModeSequential,
					Tests: []schema.Test{
						{Name: "t1", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
					},
				},
			},
		},
	}

	results := e.Run(context.Background(), suites)
	if results[0].Passed != 1 {
		t.Errorf("passed = %d, want 1", results[0].Passed)
	}
}

func TestRunInteractionDefaultMode(t *testing.T) {
	mc := newMockConnector("http")
	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })
	rep := &mockReporter{}

	e := New(
		WithRegistry(reg),
		WithReporter(rep),
	)

	// Interaction with empty mode — should default to sequential.
	suites := []schema.TestSuite{
		{
			Suite: "default-mode",
			Execution: &schema.Execution{
				Mode:   schema.ModeSequential,
				Repeat: 1,
			},
			Interactions: []schema.Interaction{
				{
					Name: "I1",
					Mode: "", // empty — should default to sequential
					Tests: []schema.Test{
						{Name: "t1", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
						{Name: "t2", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
					},
				},
			},
		},
	}

	results := e.Run(context.Background(), suites)
	if results[0].Passed != 2 {
		t.Errorf("passed = %d, want 2", results[0].Passed)
	}
}

func TestRunInteractionWithConnectorConfigs(t *testing.T) {
	mc := newMockConnector("http")
	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })
	rep := &mockReporter{}

	e := New(
		WithRegistry(reg),
		WithReporter(rep),
		WithConnectorConfigs(map[string]map[string]any{
			"http": {"timeout": 30},
		}),
	)

	suites := []schema.TestSuite{
		{
			Suite: "inter-config",
			Execution: &schema.Execution{
				Mode:   schema.ModeSequential,
				Repeat: 1,
			},
			Interactions: []schema.Interaction{
				{
					Name: "I1",
					Mode: schema.ModeSequential,
					Tests: []schema.Test{
						{Name: "t1", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
					},
				},
			},
		},
	}

	results := e.Run(context.Background(), suites)
	if results[0].Passed != 1 {
		t.Errorf("passed = %d, want 1", results[0].Passed)
	}
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if mc.lastConfig["timeout"] != 30 {
		t.Errorf("connector config not passed in interaction, got %v", mc.lastConfig)
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"", 0},
		{"100ms", 100 * time.Millisecond},
		{"1s", time.Second},
		{"invalid", 0}, // invalid returns 0
	}
	for _, tt := range tests {
		got := parseDuration(tt.input)
		if got != tt.want {
			t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestRunSuiteExecutionNilDefaultsToSinglePass(t *testing.T) {
	e, _ := newTestEngine()
	// Has interactions but no execution block — should default to
	// sequential single pass.
	suites := []schema.TestSuite{
		{
			Suite: "nilexec",
			Interactions: []schema.Interaction{
				{
					Name: "I1",
					Mode: schema.ModeSequential,
					Tests: []schema.Test{
						{Name: "t1", Connector: "http", Steps: []schema.TestStep{{Action: "request"}}},
					},
				},
			},
		},
	}
	results := e.Run(context.Background(), suites)
	if results[0].Passed != 1 {
		t.Errorf("passed = %d, want 1", results[0].Passed)
	}
}
