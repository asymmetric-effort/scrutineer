package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
	"github.com/scrutineer/scrutineer/core/coverage"
	"github.com/scrutineer/scrutineer/core/exitcode"
	"github.com/scrutineer/scrutineer/core/expression"
	"github.com/scrutineer/scrutineer/core/fleet"
	"github.com/scrutineer/scrutineer/core/reporter"
	"github.com/scrutineer/scrutineer/core/schema"
	"github.com/scrutineer/scrutineer/core/telemetry"
)

// Option configures the engine.
type Option func(*Engine)

// WithRegistry sets the connector registry.
func WithRegistry(r *connector.Registry) Option {
	return func(e *Engine) { e.registry = r }
}

// WithReporter sets the reporter for test output.
func WithReporter(r reporter.Reporter) Option {
	return func(e *Engine) { e.reporter = r }
}

// WithTelemetry sets the telemetry record writer.
func WithTelemetry(w telemetry.RecordWriter) Option {
	return func(e *Engine) { e.telemetry = w }
}

// WithCoverage sets the coverage tracker.
func WithCoverage(t *coverage.Tracker) Option {
	return func(e *Engine) { e.coverage = t }
}

// WithParallelism sets the number of parallel test workers.
func WithParallelism(n int) Option {
	return func(e *Engine) { e.parallelism = n }
}

// WithConnectorConfigs sets per-connector configuration from scrutineer.yaml.
func WithConnectorConfigs(configs map[string]map[string]any) Option {
	return func(e *Engine) { e.connectorConfigs = configs }
}

// WithExpressionRegistry sets the expression function registry for fn: evaluation.
func WithExpressionRegistry(r *expression.Registry) Option {
	return func(e *Engine) { e.exprRegistry = r }
}

// WithFleetRegistry sets the fleet provider registry for distributed execution.
func WithFleetRegistry(r *fleet.Registry) Option {
	return func(e *Engine) { e.fleetRegistry = r }
}

// Engine orchestrates test execution across suites.
type Engine struct {
	registry         *connector.Registry
	reporter         reporter.Reporter
	telemetry        telemetry.RecordWriter
	coverage         *coverage.Tracker
	parallelism      int
	connectorConfigs map[string]map[string]any
	exprRegistry     *expression.Registry
	fleetRegistry    *fleet.Registry
}

// New creates an Engine with the provided options.
func New(opts ...Option) *Engine {
	e := &Engine{
		parallelism: 1,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Run executes all test suites and returns results.
func (e *Engine) Run(ctx context.Context, suites []schema.TestSuite) []SuiteResult {
	results := make([]SuiteResult, len(suites))
	var mu sync.Mutex

	pool := NewPool(e.parallelism)
	pool.Run(len(suites), func(index int) {
		sr := e.runSuite(ctx, suites[index])
		mu.Lock()
		results[index] = sr
		mu.Unlock()
	})

	return results
}

// runSuite executes a single test suite, choosing the execution path
// based on whether an Execution block or Interactions are present.
func (e *Engine) runSuite(ctx context.Context, suite schema.TestSuite) SuiteResult {
	if suite.Execution == nil && len(suite.Interactions) == 0 {
		return e.runSuiteSimple(ctx, suite)
	}
	return e.runSuiteWithExecution(ctx, suite)
}

// runSuiteSimple is the original sequential single-pass execution path.
// It is used when no Execution block or Interactions are present,
// preserving backward compatibility.
func (e *Engine) runSuiteSimple(ctx context.Context, suite schema.TestSuite) SuiteResult {
	start := time.Now()

	info := reporter.SuiteInfo{
		Name:      suite.Suite,
		TestCount: len(suite.Tests),
	}
	e.reporter.OnSuiteStart(info)

	if e.telemetry != nil {
		_ = e.telemetry.Write(telemetry.Record{
			Timestamp: telemetry.NowNano(),
			EventType: telemetry.SuiteStart,
			Tags:      map[string]string{"suite": suite.Suite},
		})
	}

	if e.coverage != nil {
		// Count total assertions for registration.
		for _, test := range suite.Tests {
			totalAssertions := 0
			for _, step := range test.Steps {
				totalAssertions += len(step.Assert)
			}
			e.coverage.RegisterSuite(suite.Suite, len(suite.Tests))
			e.coverage.RegisterTest(suite.Suite, test.Name, len(test.Steps), totalAssertions)
		}
	}

	runner := NewRunner(e.registry, e.reporter, e.telemetry, e.coverage)

	sr := SuiteResult{
		Suite:   suite.Suite,
		Results: make([]reporter.TestResult, 0, len(suite.Tests)),
	}

	for _, test := range suite.Tests {
		if test.Skip {
			sr.Skipped++
			if e.coverage != nil {
				e.coverage.RecordTestSkip(suite.Suite, test.Name)
			}
			if e.telemetry != nil {
				_ = e.telemetry.Write(telemetry.Record{
					Timestamp: telemetry.NowNano(),
					EventType: telemetry.TestSkip,
					Tags:      map[string]string{"suite": suite.Suite, "test": test.Name},
				})
			}
			continue
		}

		tctx := NewTestContext(suite.Suite, test.Name, suite.Fixtures)
		if e.exprRegistry != nil {
			tctx.Store.SetExpressionRegistry(e.exprRegistry)
		}

		testInfo := reporter.TestInfo{
			Name:  test.Name,
			Suite: suite.Suite,
			Tags:  test.Tags,
		}
		e.reporter.OnTestStart(testInfo)

		// Build connector config from engine-level connector configs.
		connName := test.Connector
		connectorConfig := make(map[string]any)
		if e.connectorConfigs != nil && connName != "" {
			if cc, ok := e.connectorConfigs[connName]; ok {
				for k, v := range cc {
					connectorConfig[k] = v
				}
			}
		}

		result := runner.RunTest(ctx, tctx, test, connectorConfig)
		sr.Results = append(sr.Results, result)

		e.reporter.OnTestEnd(testInfo, result)
	}

	sr.summarise()
	sr.Elapsed = time.Since(start)

	summary := reporter.SuiteSummary{
		Passed:  sr.Passed,
		Failed:  sr.Failed,
		Skipped: sr.Skipped,
		Elapsed: sr.Elapsed,
	}
	e.reporter.OnSuiteEnd(info, summary)

	if e.telemetry != nil {
		_ = e.telemetry.Write(telemetry.Record{
			Timestamp: telemetry.NowNano(),
			EventType: telemetry.SuiteEnd,
			Tags:      map[string]string{"suite": suite.Suite},
		})
	}

	return sr
}

// runSuiteWithExecution handles suites that have an Execution block
// and/or Interactions. It uses dispatchers for mode-based ordering
// and a loop controller for repeat/duration.
func (e *Engine) runSuiteWithExecution(ctx context.Context, suite schema.TestSuite) SuiteResult {
	start := time.Now()

	exec := suite.Execution
	if exec == nil {
		exec = &schema.Execution{Mode: schema.ModeSequential, Repeat: 1}
	}

	// Normalize: if suite has Tests (not Interactions), wrap them in
	// a single unnamed sequential interaction.
	interactions := suite.Interactions
	if len(interactions) == 0 && len(suite.Tests) > 0 {
		interactions = []schema.Interaction{
			{
				Name:  suite.Suite,
				Mode:  schema.ModeSequential,
				Tests: suite.Tests,
			},
		}
	}

	// Count total tests for reporting.
	totalTests := 0
	for _, inter := range interactions {
		totalTests += len(inter.Tests)
	}

	info := reporter.SuiteInfo{
		Name:      suite.Suite,
		TestCount: totalTests,
	}
	e.reporter.OnSuiteStart(info)

	if e.telemetry != nil {
		_ = e.telemetry.Write(telemetry.Record{
			Timestamp: telemetry.NowNano(),
			EventType: telemetry.SuiteStart,
			Tags:      map[string]string{"suite": suite.Suite},
		})
	}

	// Set up fleet orchestrator if fleet config is present.
	var orch *fleet.Orchestrator
	if exec.Fleet != nil && e.fleetRegistry != nil {
		orch = fleet.NewOrchestrator(e.fleetRegistry, *exec.Fleet)
		if err := orch.Setup(ctx); err != nil {
			e.reporter.OnSuiteStart(info)
			e.reporter.OnSuiteEnd(info, reporter.SuiteSummary{
				Failed:  1,
				Elapsed: time.Since(start),
			})
			return SuiteResult{
				Suite:   suite.Suite,
				Failed:  1,
				Elapsed: time.Since(start),
				Results: []reporter.TestResult{{
					Passed: false,
					Steps: []reporter.StepResult{{
						Name:   "fleet-setup",
						Passed: false,
						Error:  fmt.Errorf("fleet setup failed: %w", err),
					}},
				}},
			}
		}
		defer orch.Teardown(context.Background())
	}

	runner := NewRunner(e.registry, e.reporter, e.telemetry, e.coverage)

	sr := SuiteResult{
		Suite:   suite.Suite,
		Results: make([]reporter.TestResult, 0),
	}
	var mu sync.Mutex

	// Parse duration and interval.
	duration := parseDuration(exec.Duration)
	interval := parseDuration(exec.Interval)
	repeat := exec.Repeat
	if repeat == 0 && duration == 0 {
		repeat = 1 // safety fallback
	}

	// Build interaction weights for weighted dispatcher.
	interWeights := make([]int, len(interactions))
	for i, inter := range interactions {
		w := inter.Weight
		if w <= 0 {
			w = 1
		}
		interWeights[i] = w
	}

	// Outer dispatcher: dispatches interactions.
	outerDispatcher := NewDispatcher(exec.Mode, exec.Concurrency, interWeights)

	// For weighted mode, each pass selects one interaction by probability.
	// For all other modes, each pass dispatches all interactions.
	dispatchCount := len(interactions)
	if exec.Mode == schema.ModeWeighted {
		dispatchCount = 1
	}

	lc := NewLoopController(repeat, duration, interval)
	lc.Run(ctx, func(passCtx context.Context, passNum int) {
		outerDispatcher.Dispatch(passCtx, dispatchCount, func(dCtx context.Context, interIdx int) {
			inter := interactions[interIdx]

			// Select provider for this interaction if fleet is configured.
			var providerName string
			if orch != nil {
				providerName = orch.SelectProvider()
			}

			e.runInteraction(dCtx, runner, inter, suite, passNum, providerName, &sr, &mu)
		})
	})

	sr.summarise()
	sr.Elapsed = time.Since(start)

	summary := reporter.SuiteSummary{
		Passed:  sr.Passed,
		Failed:  sr.Failed,
		Skipped: sr.Skipped,
		Elapsed: sr.Elapsed,
	}
	e.reporter.OnSuiteEnd(info, summary)

	if e.telemetry != nil {
		_ = e.telemetry.Write(telemetry.Record{
			Timestamp: telemetry.NowNano(),
			EventType: telemetry.SuiteEnd,
			Tags:      map[string]string{"suite": suite.Suite},
		})
	}

	return sr
}

// runInteraction executes all tests within an interaction using the
// interaction's own dispatcher mode.
func (e *Engine) runInteraction(
	ctx context.Context,
	runner *Runner,
	inter schema.Interaction,
	suite schema.TestSuite,
	passNum int,
	providerName string,
	sr *SuiteResult,
	mu *sync.Mutex,
) {
	// Each interaction gets its own TestContext so captures are isolated
	// between interactions but shared within one interaction.
	tctx := NewTestContext(suite.Suite, "", suite.Fixtures)
	tctx.Interaction = inter.Name
	tctx.PassNum = passNum
	if e.exprRegistry != nil {
		tctx.Store.SetExpressionRegistry(e.exprRegistry)
	}

	// Build test weights for weighted dispatcher.
	testWeights := make([]int, len(inter.Tests))
	for i, test := range inter.Tests {
		w := test.Weight
		if w <= 0 {
			w = 1
		}
		testWeights[i] = w
	}

	mode := inter.Mode
	if mode == "" {
		mode = schema.ModeSequential
	}

	// Emit interaction start telemetry.
	if e.telemetry != nil {
		tags := map[string]string{
			"suite":       suite.Suite,
			"interaction": inter.Name,
			"mode":        string(mode),
			"pass":        fmt.Sprintf("%d", passNum),
		}
		if providerName != "" {
			tags["provider"] = providerName
		}
		_ = e.telemetry.Write(telemetry.Record{
			Timestamp: telemetry.NowNano(),
			EventType: telemetry.InteractionStart,
			Tags:      tags,
		})
	}

	// For sequential mode, share the TestContext so captures flow between
	// tests within the interaction. For concurrent/random/weighted modes,
	// each test gets its own TestContext to avoid data races.
	shared := mode == schema.ModeSequential

	// For weighted interaction mode, dispatch count = len(tests) means each
	// dispatch makes N weighted selections from the test pool. Some tests may
	// run multiple times and others may not run at all — this is intentional
	// for simulating realistic workload distributions within an interaction.
	innerDispatcher := NewDispatcher(mode, 0, testWeights)
	innerDispatcher.Dispatch(ctx, len(inter.Tests), func(tCtx context.Context, testIdx int) {
		test := inter.Tests[testIdx]

		if test.Skip {
			mu.Lock()
			sr.Skipped++
			mu.Unlock()
			return
		}

		// Use per-test context for non-sequential modes to avoid races.
		testCtx := tctx
		if !shared {
			testCtx = NewTestContext(suite.Suite, test.Name, suite.Fixtures)
			testCtx.Interaction = inter.Name
			testCtx.PassNum = passNum
			if e.exprRegistry != nil {
				testCtx.Store.SetExpressionRegistry(e.exprRegistry)
			}
		} else {
			testCtx.Test = test.Name
		}

		testInfo := reporter.TestInfo{
			Name:  test.Name,
			Suite: suite.Suite,
			Tags:  test.Tags,
		}
		e.reporter.OnTestStart(testInfo)

		connName := test.Connector
		connectorConfig := make(map[string]any)
		if e.connectorConfigs != nil && connName != "" {
			if cc, ok := e.connectorConfigs[connName]; ok {
				for k, v := range cc {
					connectorConfig[k] = v
				}
			}
		}

		result := runner.RunTest(tCtx, testCtx, test, connectorConfig)

		mu.Lock()
		sr.Results = append(sr.Results, result)
		mu.Unlock()

		e.reporter.OnTestEnd(testInfo, result)
	})

	// Emit interaction end telemetry.
	if e.telemetry != nil {
		_ = e.telemetry.Write(telemetry.Record{
			Timestamp: telemetry.NowNano(),
			EventType: telemetry.InteractionEnd,
			Tags: map[string]string{
				"suite":       suite.Suite,
				"interaction": inter.Name,
				"pass":        fmt.Sprintf("%d", passNum),
			},
		})
	}
}

// parseDuration parses a duration string, returning 0 for empty strings.
func parseDuration(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, _ := time.ParseDuration(s)
	return d
}

// ExitCode returns the appropriate exit code based on results.
func (e *Engine) ExitCode(results []SuiteResult) int {
	for _, sr := range results {
		if sr.Failed > 0 {
			return exitcode.TestFailure
		}
	}
	return exitcode.OK
}
