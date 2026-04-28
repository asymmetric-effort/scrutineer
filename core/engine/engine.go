package engine

import (
	"context"
	"sync"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
	"github.com/scrutineer/scrutineer/core/coverage"
	"github.com/scrutineer/scrutineer/core/exitcode"
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

// Engine orchestrates test execution across suites.
type Engine struct {
	registry         *connector.Registry
	reporter         reporter.Reporter
	telemetry        telemetry.RecordWriter
	coverage         *coverage.Tracker
	parallelism      int
	connectorConfigs map[string]map[string]any
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

// runSuite executes a single test suite.
func (e *Engine) runSuite(ctx context.Context, suite schema.TestSuite) SuiteResult {
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

// ExitCode returns the appropriate exit code based on results.
func (e *Engine) ExitCode(results []SuiteResult) int {
	for _, sr := range results {
		if sr.Failed > 0 {
			return exitcode.TestFailure
		}
	}
	return exitcode.OK
}
