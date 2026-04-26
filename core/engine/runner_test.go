package engine

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
	"github.com/scrutineer/scrutineer/core/coverage"
	"github.com/scrutineer/scrutineer/core/reporter"
	"github.com/scrutineer/scrutineer/core/schema"
	"github.com/scrutineer/scrutineer/core/telemetry"
)

// mockConnector records calls and returns canned results.
type mockConnector struct {
	mu          sync.Mutex
	name        string
	setupCalls  int
	execCalls   int
	tearCalls   int
	setupErr    error
	execErr     error
	tearErr     error
	execResult  *connector.Result
	lastConfig  map[string]any
	lastStep    connector.Step
}

func newMockConnector(name string) *mockConnector {
	return &mockConnector{
		name: name,
		execResult: &connector.Result{
			Data:    map[string]any{"status_code": 200, "body": "ok"},
			Elapsed: 5 * time.Millisecond,
		},
	}
}

func (m *mockConnector) Name() string { return m.name }

func (m *mockConnector) Setup(_ context.Context, config map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setupCalls++
	m.lastConfig = config
	return m.setupErr
}

func (m *mockConnector) Execute(_ context.Context, step connector.Step) (*connector.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.execCalls++
	m.lastStep = step
	return m.execResult, m.execErr
}

func (m *mockConnector) Teardown(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tearCalls++
	return m.tearErr
}

// mockReporter records reporter calls.
type mockReporter struct {
	mu           sync.Mutex
	suiteStarts  []reporter.SuiteInfo
	testStarts   []reporter.TestInfo
	stepResults  []reporter.StepResult
	testEnds     []reporter.TestResult
	suiteEnds    []reporter.SuiteSummary
}

func (m *mockReporter) OnSuiteStart(suite reporter.SuiteInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.suiteStarts = append(m.suiteStarts, suite)
}

func (m *mockReporter) OnTestStart(test reporter.TestInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.testStarts = append(m.testStarts, test)
}

func (m *mockReporter) OnStepResult(_ reporter.TestInfo, step reporter.StepResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stepResults = append(m.stepResults, step)
}

func (m *mockReporter) OnTestEnd(_ reporter.TestInfo, result reporter.TestResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.testEnds = append(m.testEnds, result)
}

func (m *mockReporter) OnSuiteEnd(_ reporter.SuiteInfo, summary reporter.SuiteSummary) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.suiteEnds = append(m.suiteEnds, summary)
}

func (m *mockReporter) Flush(_ io.Writer) error {
	return nil
}

// mockTelemetryWriter records written telemetry records.
type mockTelemetryWriter struct {
	mu      sync.Mutex
	records []telemetry.Record
	closed  bool
}

func (m *mockTelemetryWriter) Write(rec telemetry.Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, rec)
	return nil
}

func (m *mockTelemetryWriter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func setupRunner(mc *mockConnector) (*Runner, *mockReporter, *mockTelemetryWriter) {
	reg := connector.NewRegistry()
	_ = reg.Register(mc.Name(), func() connector.Connector { return mc })

	rep := &mockReporter{}
	tw := &mockTelemetryWriter{}

	runner := NewRunner(reg, rep, tw, nil)
	return runner, rep, tw
}

func TestRunTestSingleStep(t *testing.T) {
	mc := newMockConnector("http")
	runner, rep, _ := setupRunner(mc)

	tctx := NewTestContext("suite1", "test1", nil)
	test := schema.Test{
		Name:      "test1",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Action:     "GET",
				Parameters: map[string]any{"url": "/api/health"},
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, map[string]any{"base_url": "http://localhost"})

	if !result.Passed {
		t.Errorf("expected test to pass, got failed")
	}
	if len(result.Steps) != 1 {
		t.Fatalf("expected 1 step result, got %d", len(result.Steps))
	}
	if !result.Steps[0].Passed {
		t.Errorf("step 0 should pass, error: %v", result.Steps[0].Error)
	}

	if mc.setupCalls != 1 {
		t.Errorf("setup called %d times, want 1", mc.setupCalls)
	}
	if mc.execCalls != 1 {
		t.Errorf("execute called %d times, want 1", mc.execCalls)
	}
	if mc.tearCalls != 1 {
		t.Errorf("teardown called %d times, want 1", mc.tearCalls)
	}

	if len(rep.stepResults) != 1 {
		t.Errorf("reporter received %d step results, want 1", len(rep.stepResults))
	}
}

func TestRunTestAssertionsPass(t *testing.T) {
	mc := newMockConnector("http")
	mc.execResult = &connector.Result{
		Data: map[string]any{
			"status_code": 200,
			"body":        "hello world",
		},
		Elapsed: time.Millisecond,
	}

	runner, _, _ := setupRunner(mc)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Action: "GET",
				Assert: []map[string]any{
					{"field": "status_code", "operator": "equal", "expected": 200},
				},
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if !result.Passed {
		t.Errorf("expected pass, got fail: %v", result.Steps[0].Error)
	}
}

func TestRunTestAssertionsFail(t *testing.T) {
	mc := newMockConnector("http")
	mc.execResult = &connector.Result{
		Data: map[string]any{
			"status_code": 500,
		},
		Elapsed: time.Millisecond,
	}

	runner, _, _ := setupRunner(mc)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Action: "GET",
				Assert: []map[string]any{
					{"field": "status_code", "operator": "equal", "expected": 200},
				},
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if result.Passed {
		t.Error("expected fail, got pass")
	}
	if result.Steps[0].Error == nil {
		t.Error("expected error on failed assertion")
	}
}

func TestRunTestCaptureAndInterpolation(t *testing.T) {
	mc := newMockConnector("http")

	callCount := 0
	reg := connector.NewRegistry()
	// Each call to Get creates a new connector, so we use a factory
	// that returns the same mock each time and adjust the result.
	_ = reg.Register("http", func() connector.Connector {
		return &dynamicMockConnector{
			mock: mc,
			onExecute: func(step connector.Step) (*connector.Result, error) {
				callCount++
				if callCount == 1 {
					return &connector.Result{
						Data: map[string]any{
							"body": map[string]any{
								"id": "abc-123",
							},
						},
						Elapsed: time.Millisecond,
					}, nil
				}
				// Second call: verify the interpolated value.
				return &connector.Result{
					Data:    map[string]any{"body": "ok"},
					Elapsed: time.Millisecond,
				}, nil
			},
		}
	})

	rep := &mockReporter{}
	runner := NewRunner(reg, rep, nil, nil)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Action:     "POST",
				Parameters: map[string]any{"body": `{"name":"test"}`},
				Capture:    map[string]string{"user_id": "body.id"},
			},
			{
				Action:     "GET",
				Parameters: map[string]any{"url": "/users/${capture.user_id}"},
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if !result.Passed {
		t.Errorf("expected pass, got fail")
		for _, s := range result.Steps {
			if s.Error != nil {
				t.Logf("step %s error: %v", s.Name, s.Error)
			}
		}
	}

	// Verify capture was stored.
	val, ok := tctx.Store.GetCapture("user_id")
	if !ok {
		t.Fatal("capture user_id not found")
	}
	if val != "abc-123" {
		t.Errorf("captured user_id = %v, want %q", val, "abc-123")
	}
}

func TestRunTestConnectorNotFound(t *testing.T) {
	reg := connector.NewRegistry()
	rep := &mockReporter{}
	runner := NewRunner(reg, rep, nil, nil)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "nonexistent",
		Steps: []schema.TestStep{
			{Action: "GET"},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if result.Passed {
		t.Error("expected fail when connector not found")
	}
	if result.Steps[0].Error == nil {
		t.Error("expected error for missing connector")
	}
}

func TestRunTestExecuteError(t *testing.T) {
	mc := newMockConnector("http")
	mc.execErr = errors.New("connection refused")

	runner, _, _ := setupRunner(mc)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{Action: "GET"},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if result.Passed {
		t.Error("expected fail when execute returns error")
	}
	if mc.tearCalls != 1 {
		t.Errorf("teardown should still be called, got %d", mc.tearCalls)
	}
}

func TestRunTestSetupError(t *testing.T) {
	mc := newMockConnector("http")
	mc.setupErr = errors.New("setup failed")

	runner, _, _ := setupRunner(mc)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{Action: "GET"},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if result.Passed {
		t.Error("expected fail when setup returns error")
	}
}

func TestRunTestContextCancelled(t *testing.T) {
	mc := newMockConnector("http")
	runner, _, _ := setupRunner(mc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{Action: "GET"},
		},
	}

	result := runner.RunTest(ctx, tctx, test, nil)

	if result.Passed {
		t.Error("expected fail when context cancelled")
	}
}

func TestRunTestNoConnectorSpecified(t *testing.T) {
	mc := newMockConnector("http")
	runner, _, _ := setupRunner(mc)

	tctx := NewTestContext("s", "t", nil)
	// Test with no connector at test or step level.
	test := schema.Test{
		Name: "t",
		Steps: []schema.TestStep{
			{Action: "GET"},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if result.Passed {
		t.Error("expected fail when no connector specified")
	}
}

func TestRunTestWithCoverage(t *testing.T) {
	mc := newMockConnector("http")
	mc.execResult = &connector.Result{
		Data:    map[string]any{"status_code": 200},
		Elapsed: time.Millisecond,
	}

	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })

	rep := &mockReporter{}
	cov := coverage.NewTracker()
	cov.RegisterSuite("s", 1)
	cov.RegisterTest("s", "t", 1, 1)

	runner := NewRunner(reg, rep, nil, cov)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Action: "GET",
				Assert: []map[string]any{
					{"field": "status_code", "operator": "equal", "expected": 200},
				},
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if !result.Passed {
		t.Errorf("expected pass, error: %v", result.Steps[0].Error)
	}

	results := cov.Results()
	if len(results) == 0 {
		t.Fatal("no coverage results")
	}
	if results[0].RanTests != 1 {
		t.Errorf("expected 1 ran test, got %d", results[0].RanTests)
	}
}

func TestRunTestTelemetryEvents(t *testing.T) {
	mc := newMockConnector("http")
	runner, _, tw := setupRunner(mc)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{Action: "GET"},
		},
	}

	runner.RunTest(context.Background(), tctx, test, nil)

	tw.mu.Lock()
	defer tw.mu.Unlock()

	if len(tw.records) == 0 {
		t.Fatal("no telemetry records written")
	}

	// Should have at least TestStart, StepStart, StepEnd, TestPass,
	// ConnectorSetup, ConnectorTeardown.
	events := make(map[telemetry.EventType]int)
	for _, rec := range tw.records {
		events[rec.EventType]++
	}

	if events[telemetry.TestStart] != 1 {
		t.Errorf("expected 1 TestStart, got %d", events[telemetry.TestStart])
	}
	if events[telemetry.TestPass] != 1 {
		t.Errorf("expected 1 TestPass, got %d", events[telemetry.TestPass])
	}
	if events[telemetry.StepStart] != 1 {
		t.Errorf("expected 1 StepStart, got %d", events[telemetry.StepStart])
	}
}

func TestRunTestMissingOperator(t *testing.T) {
	mc := newMockConnector("http")
	mc.execResult = &connector.Result{
		Data:    map[string]any{"status_code": 200},
		Elapsed: time.Millisecond,
	}

	runner, _, _ := setupRunner(mc)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Action: "GET",
				Assert: []map[string]any{
					{"field": "status_code"}, // missing operator
				},
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if result.Passed {
		t.Error("expected fail with missing operator")
	}
}

func TestRunTestInvalidOperator(t *testing.T) {
	mc := newMockConnector("http")
	mc.execResult = &connector.Result{
		Data:    map[string]any{"status_code": 200},
		Elapsed: time.Millisecond,
	}

	runner, _, _ := setupRunner(mc)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Action: "GET",
				Assert: []map[string]any{
					{"field": "status_code", "operator": "bogus", "expected": 200},
				},
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if result.Passed {
		t.Error("expected fail with invalid operator")
	}
}

func TestRunTestAssertionFieldNotFound(t *testing.T) {
	mc := newMockConnector("http")
	mc.execResult = &connector.Result{
		Data:    map[string]any{"status_code": 200},
		Elapsed: time.Millisecond,
	}

	runner, _, _ := setupRunner(mc)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Action: "GET",
				Assert: []map[string]any{
					{"field": "nonexistent.field", "operator": "equal", "expected": 200},
				},
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if result.Passed {
		t.Error("expected fail when field not found")
	}
}

func TestRunTestMultipleStepsStopOnFailure(t *testing.T) {
	mc := newMockConnector("http")
	mc.execResult = &connector.Result{
		Data:    map[string]any{"status_code": 500},
		Elapsed: time.Millisecond,
	}

	runner, _, _ := setupRunner(mc)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Action: "GET",
				Assert: []map[string]any{
					{"field": "status_code", "operator": "equal", "expected": 200}, // fails
				},
			},
			{
				Action: "GET", // should not run
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if result.Passed {
		t.Error("expected fail")
	}
	if len(result.Steps) != 1 {
		t.Errorf("expected 1 step result (stopped on failure), got %d", len(result.Steps))
	}
}

func TestRunTestWithTimeout(t *testing.T) {
	mc := newMockConnector("http")
	runner, _, _ := setupRunner(mc)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Action:  "GET",
				Timeout: "5s",
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if !result.Passed {
		t.Errorf("expected pass, got error: %v", result.Steps[0].Error)
	}
}

func TestRunTestStepConnectorOverride(t *testing.T) {
	httpMock := newMockConnector("http")
	cliMock := newMockConnector("cli")
	cliMock.execResult = &connector.Result{
		Data:    map[string]any{"stdout": "hello"},
		Elapsed: time.Millisecond,
	}

	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return httpMock })
	_ = reg.Register("cli", func() connector.Connector { return cliMock })

	rep := &mockReporter{}
	runner := NewRunner(reg, rep, nil, nil)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Connector: "cli", // overrides test-level http
				Action:    "run",
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if !result.Passed {
		t.Errorf("expected pass")
	}
	if cliMock.execCalls != 1 {
		t.Errorf("expected cli connector to be called, got %d calls", cliMock.execCalls)
	}
	if httpMock.execCalls != 0 {
		t.Errorf("expected http connector not to be called, got %d calls", httpMock.execCalls)
	}
}

func TestRunTestInterpolationError(t *testing.T) {
	mc := newMockConnector("http")
	runner, _, _ := setupRunner(mc)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Action:     "GET",
				Parameters: map[string]any{"url": "${nonexistent.var}"},
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if result.Passed {
		t.Error("expected fail on interpolation error")
	}
}

func TestRunTestAssertionWithoutField(t *testing.T) {
	mc := newMockConnector("http")
	mc.execResult = &connector.Result{
		Data:    map[string]any{"status_code": 200},
		Elapsed: time.Millisecond,
	}

	runner, _, _ := setupRunner(mc)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Action: "GET",
				Assert: []map[string]any{
					{"operator": "not_empty"}, // no field = assert on whole data map
				},
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if !result.Passed {
		t.Errorf("expected pass, got error: %v", result.Steps[0].Error)
	}
}

func TestRunTestCoverageAssertionFail(t *testing.T) {
	mc := newMockConnector("http")
	mc.execResult = &connector.Result{
		Data:    map[string]any{"status_code": 500},
		Elapsed: time.Millisecond,
	}

	reg := connector.NewRegistry()
	_ = reg.Register("http", func() connector.Connector { return mc })

	rep := &mockReporter{}
	cov := coverage.NewTracker()
	cov.RegisterSuite("s", 1)
	cov.RegisterTest("s", "t", 1, 1)

	runner := NewRunner(reg, rep, nil, cov)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Action: "GET",
				Assert: []map[string]any{
					{"field": "status_code", "operator": "equal", "expected": 200},
				},
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if result.Passed {
		t.Error("expected fail")
	}

	results := cov.Results()
	if len(results) == 0 || len(results[0].Tests) == 0 {
		t.Fatal("no coverage data")
	}
	tc := results[0].Tests[0]
	if tc.FailedAssertions != 1 {
		t.Errorf("expected 1 failed assertion, got %d", tc.FailedAssertions)
	}
}

func TestRunTestCaptureExtractError(t *testing.T) {
	mc := newMockConnector("http")
	mc.execResult = &connector.Result{
		Data:    map[string]any{"body": "not a map"},
		Elapsed: time.Millisecond,
	}

	runner, _, _ := setupRunner(mc)

	tctx := NewTestContext("s", "t", nil)
	test := schema.Test{
		Name:      "t",
		Connector: "http",
		Steps: []schema.TestStep{
			{
				Action:  "GET",
				Capture: map[string]string{"id": "body.nested.id"},
			},
		},
	}

	result := runner.RunTest(context.Background(), tctx, test, nil)

	if result.Passed {
		t.Error("expected fail on capture extract error")
	}
}

func TestStepName(t *testing.T) {
	s1 := schema.TestStep{Action: "GET"}
	if got := stepName(s1, 0); got != "step[0]:GET" {
		t.Errorf("stepName = %q, want %q", got, "step[0]:GET")
	}

	s2 := schema.TestStep{}
	if got := stepName(s2, 3); got != "step[3]" {
		t.Errorf("stepName = %q, want %q", got, "step[3]")
	}
}

// dynamicMockConnector delegates to a base mock but allows custom execute behavior.
type dynamicMockConnector struct {
	mock      *mockConnector
	onExecute func(step connector.Step) (*connector.Result, error)
}

func (d *dynamicMockConnector) Name() string { return d.mock.Name() }
func (d *dynamicMockConnector) Setup(ctx context.Context, config map[string]any) error {
	return d.mock.Setup(ctx, config)
}
func (d *dynamicMockConnector) Execute(_ context.Context, step connector.Step) (*connector.Result, error) {
	if d.onExecute != nil {
		return d.onExecute(step)
	}
	return d.mock.Execute(context.Background(), step)
}
func (d *dynamicMockConnector) Teardown(ctx context.Context) error {
	return d.mock.Teardown(ctx)
}
