package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/scrutineer/scrutineer/core/assertion"
	"github.com/scrutineer/scrutineer/core/connector"
	"github.com/scrutineer/scrutineer/core/coverage"
	"github.com/scrutineer/scrutineer/core/fixture"
	"github.com/scrutineer/scrutineer/core/reporter"
	"github.com/scrutineer/scrutineer/core/schema"
	"github.com/scrutineer/scrutineer/core/telemetry"
)

// Runner executes individual tests against connectors.
type Runner struct {
	registry      *connector.Registry
	reporter      reporter.Reporter
	telemetry     telemetry.RecordWriter // may be nil
	coverage      *coverage.Tracker      // may be nil
	assertBuilder *assertion.DefaultBuilder
}

// NewRunner creates a Runner with the provided dependencies.
func NewRunner(reg *connector.Registry, rep reporter.Reporter, tw telemetry.RecordWriter, cov *coverage.Tracker) *Runner {
	return &Runner{
		registry:      reg,
		reporter:      rep,
		telemetry:     tw,
		coverage:      cov,
		assertBuilder: &assertion.DefaultBuilder{},
	}
}

// RunTest executes a single test (its steps in order).
// Returns a TestResult summarising the outcome.
func (r *Runner) RunTest(ctx context.Context, tctx *TestContext, test schema.Test, connectorConfig map[string]any) reporter.TestResult {
	start := time.Now()

	r.emitTelemetry(telemetry.TestStart, tctx, nil)

	var stepResults []reporter.StepResult
	passed := true

	for i, step := range test.Steps {
		if ctx.Err() != nil {
			stepResults = append(stepResults, reporter.StepResult{
				Name:   stepName(step, i),
				Passed: false,
				Error:  ctx.Err(),
			})
			passed = false
			break
		}

		// Use connector from step, falling back to test-level connector.
		if step.Connector == "" {
			step.Connector = test.Connector
		}

		sr := r.runStep(ctx, tctx, step, i, connectorConfig)
		stepResults = append(stepResults, sr)

		info := reporter.TestInfo{Name: tctx.Test, Suite: tctx.Suite}
		r.reporter.OnStepResult(info, sr)

		if !sr.Passed {
			passed = false
			break
		}
	}

	elapsed := time.Since(start)
	result := reporter.TestResult{
		Passed:  passed,
		Steps:   stepResults,
		Elapsed: elapsed,
	}

	if passed {
		r.emitTelemetry(telemetry.TestPass, tctx, nil)
	} else {
		r.emitTelemetry(telemetry.TestFail, tctx, nil)
	}

	if r.coverage != nil {
		r.coverage.RecordTestRun(tctx.Suite, tctx.Test)
	}

	return result
}

// runStep executes a single test step: connector setup, execute, assertions,
// captures, teardown.
func (r *Runner) runStep(ctx context.Context, tctx *TestContext, step schema.TestStep, index int, connectorConfig map[string]any) reporter.StepResult {
	name := stepName(step, index)
	stepStart := time.Now()

	r.emitTelemetry(telemetry.StepStart, tctx, map[string]string{"step": name})

	connName := step.Connector
	if connName == "" {
		return reporter.StepResult{
			Name:    name,
			Passed:  false,
			Elapsed: time.Since(stepStart),
			Error:   fmt.Errorf("no connector specified for step %q", name),
		}
	}

	// Get connector from registry.
	conn, err := r.registry.Get(connName)
	if err != nil {
		r.emitTelemetry(telemetry.Error, tctx, map[string]string{"step": name, "error": err.Error()})
		return reporter.StepResult{
			Name:    name,
			Passed:  false,
			Elapsed: time.Since(stepStart),
			Error:   fmt.Errorf("connector %q: %w", connName, err),
		}
	}

	// Setup connector.
	r.emitTelemetry(telemetry.ConnectorSetup, tctx, map[string]string{"connector": connName})
	if err := conn.Setup(ctx, connectorConfig); err != nil {
		r.emitTelemetry(telemetry.Error, tctx, map[string]string{"step": name, "error": err.Error()})
		return reporter.StepResult{
			Name:    name,
			Passed:  false,
			Elapsed: time.Since(stepStart),
			Error:   fmt.Errorf("connector setup: %w", err),
		}
	}

	// Always teardown, even on failure.
	defer func() {
		r.emitTelemetry(telemetry.ConnectorTeardown, tctx, map[string]string{"connector": connName})
		_ = conn.Teardown(ctx)
	}()

	// Interpolate parameters.
	var params map[string]any
	if step.Parameters != nil {
		params, err = tctx.Store.InterpolateMap(step.Parameters)
		if err != nil {
			r.emitTelemetry(telemetry.Error, tctx, map[string]string{"step": name, "error": err.Error()})
			return reporter.StepResult{
				Name:    name,
				Passed:  false,
				Elapsed: time.Since(stepStart),
				Error:   fmt.Errorf("variable interpolation: %w", err),
			}
		}
	}

	// Build connector step.
	cStep := connector.Step{
		Action:     step.Action,
		Parameters: params,
	}
	if step.Timeout != "" {
		d, parseErr := time.ParseDuration(step.Timeout)
		if parseErr == nil {
			cStep.Timeout = d
		}
	}

	// Execute.
	result, err := conn.Execute(ctx, cStep)
	if err != nil {
		r.emitTelemetry(telemetry.Error, tctx, map[string]string{"step": name, "error": err.Error()})
		return reporter.StepResult{
			Name:    name,
			Passed:  false,
			Elapsed: time.Since(stepStart),
			Error:   fmt.Errorf("execute: %w", err),
		}
	}

	if r.coverage != nil {
		r.coverage.RecordStep(tctx.Suite, tctx.Test)
	}

	// Process captures.
	for varName, path := range step.Capture {
		val, extractErr := fixture.Extract(result.Data, path)
		if extractErr != nil {
			r.emitTelemetry(telemetry.Error, tctx, map[string]string{"step": name, "error": extractErr.Error()})
			return reporter.StepResult{
				Name:    name,
				Passed:  false,
				Elapsed: time.Since(stepStart),
				Error:   fmt.Errorf("capture %q from %q: %w", varName, path, extractErr),
			}
		}
		tctx.Store.SetCapture(varName, val)
	}

	// Evaluate assertions.
	for _, assertMap := range step.Assert {
		if assertErr := r.evaluateAssertion(tctx, result.Data, assertMap); assertErr != nil {
			r.emitTelemetry(telemetry.Assertion, tctx, map[string]string{"step": name, "passed": "false"})
			if r.coverage != nil {
				r.coverage.RecordAssertion(tctx.Suite, tctx.Test, false)
			}
			return reporter.StepResult{
				Name:    name,
				Passed:  false,
				Elapsed: time.Since(stepStart),
				Error:   assertErr,
			}
		}
		r.emitTelemetry(telemetry.Assertion, tctx, map[string]string{"step": name, "passed": "true"})
		if r.coverage != nil {
			r.coverage.RecordAssertion(tctx.Suite, tctx.Test, true)
		}
	}

	r.emitTelemetry(telemetry.StepEnd, tctx, map[string]string{"step": name})

	return reporter.StepResult{
		Name:    name,
		Passed:  true,
		Elapsed: time.Since(stepStart),
	}
}

// evaluateAssertion builds and evaluates a single assertion from an assert map.
// The map is expected to contain keys: "field" (optional), "operator", "expected",
// plus any extra keys passed as options.
func (r *Runner) evaluateAssertion(tctx *TestContext, data map[string]any, assertMap map[string]any) error {
	field, _ := assertMap["field"].(string)
	operator, _ := assertMap["operator"].(string)
	expected := assertMap["expected"]

	if operator == "" {
		return fmt.Errorf("assertion missing 'operator' field")
	}

	// Build options map from remaining keys.
	options := make(map[string]any)
	for k, v := range assertMap {
		switch k {
		case "field", "operator", "expected":
			continue
		default:
			options[k] = v
		}
	}

	a, err := r.assertBuilder.Build(operator, expected, options)
	if err != nil {
		return fmt.Errorf("build assertion %q: %w", operator, err)
	}

	// Extract the value at the field path from data.
	var actual any
	if field != "" {
		val, extractErr := fixture.Extract(data, field)
		if extractErr != nil {
			return fmt.Errorf("extract field %q for assertion: %w", field, extractErr)
		}
		actual = val
	} else {
		actual = data
	}

	return a.Evaluate(actual)
}

// emitTelemetry writes a telemetry record if the writer is configured.
func (r *Runner) emitTelemetry(event telemetry.EventType, tctx *TestContext, extra map[string]string) {
	if r.telemetry == nil {
		return
	}

	tags := map[string]string{
		"suite": tctx.Suite,
		"test":  tctx.Test,
	}
	for k, v := range extra {
		tags[k] = v
	}

	_ = r.telemetry.Write(telemetry.Record{
		Timestamp: telemetry.NowNano(),
		EventType: event,
		Tags:      tags,
	})
}

// stepName returns a display name for a step.
func stepName(step schema.TestStep, index int) string {
	if step.Action != "" {
		return fmt.Sprintf("step[%d]:%s", index, step.Action)
	}
	return fmt.Sprintf("step[%d]", index)
}
