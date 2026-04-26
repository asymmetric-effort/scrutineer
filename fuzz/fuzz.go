// Package fuzz provides a declarative fuzz testing layer on top of Go's
// built-in testing.F. Users define fuzz targets in YAML, and scrutineer
// generates and manages fuzz test execution.
package fuzz

import (
	"context"
	"fmt"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
)

// Target defines a declarative fuzz target from YAML.
type Target struct {
	Name       string            // target name
	Connector  string            // which connector to use
	Action     string            // the action to fuzz
	Parameters map[string]any    // base parameters (some fields will be fuzzed)
	FuzzFields []string          // which parameter fields to fuzz
	Seed       []map[string]any  // seed corpus entries
	Assert     []map[string]any  // assertions that must hold for any input
}

// ParseTarget creates a Target from YAML parameters.
func ParseTarget(params map[string]any) (*Target, error) {
	t := &Target{
		Parameters: make(map[string]any),
	}

	if v, ok := params["name"]; ok {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("name must be a string")
		}
		t.Name = s
	}

	if v, ok := params["connector"]; ok {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("connector must be a string")
		}
		t.Connector = s
	}

	if v, ok := params["action"]; ok {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("action must be a string")
		}
		t.Action = s
	}

	if v, ok := params["parameters"]; ok {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("parameters must be a map")
		}
		t.Parameters = m
	}

	if v, ok := params["fuzz_fields"]; ok {
		sl, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("fuzz_fields must be a list")
		}
		for _, item := range sl {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("fuzz_fields entries must be strings")
			}
			t.FuzzFields = append(t.FuzzFields, s)
		}
	}

	if v, ok := params["seed"]; ok {
		sl, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("seed must be a list")
		}
		for _, item := range sl {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("seed entries must be maps")
			}
			t.Seed = append(t.Seed, m)
		}
	}

	if v, ok := params["assert"]; ok {
		sl, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("assert must be a list")
		}
		for _, item := range sl {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("assert entries must be maps")
			}
			t.Assert = append(t.Assert, m)
		}
	}

	if err := t.Validate(); err != nil {
		return nil, fmt.Errorf("invalid target: %w", err)
	}

	return t, nil
}

// Runner executes fuzz targets.
type Runner struct {
	connector connector.Connector
}

// NewRunner creates a new Runner with the given connector.
func NewRunner(c connector.Connector) *Runner {
	return &Runner{connector: c}
}

// FuzzResult holds the outcome of a fuzz run.
type FuzzResult struct {
	Iterations int
	Failures   []FuzzFailure
	Duration   time.Duration
}

// FuzzFailure records a single failing fuzz input.
type FuzzFailure struct {
	Input map[string]any
	Error string
}

// Fuzz runs the fuzz target, generating inputs and checking assertions.
// iterations controls how many random inputs to try (0 = unlimited until
// context is cancelled).
func (r *Runner) Fuzz(ctx context.Context, target *Target, iterations int) (*FuzzResult, error) {
	if err := target.Validate(); err != nil {
		return nil, fmt.Errorf("invalid target: %w", err)
	}

	gen := NewGenerator(target, time.Now().UnixNano())
	result := &FuzzResult{}
	start := time.Now()

	// Run seed corpus first.
	for _, seed := range target.Seed {
		select {
		case <-ctx.Done():
			result.Duration = time.Since(start)
			return result, ctx.Err()
		default:
		}

		input := copyMap(target.Parameters)
		for k, v := range seed {
			input[k] = v
		}

		failure := r.executeAndCheck(ctx, target, input)
		result.Iterations++
		if failure != nil {
			result.Failures = append(result.Failures, *failure)
		}
	}

	// Run generated inputs.
	i := 0
	for iterations == 0 || i < iterations {
		select {
		case <-ctx.Done():
			result.Duration = time.Since(start)
			return result, ctx.Err()
		default:
		}

		input := gen.Next()
		failure := r.executeAndCheck(ctx, target, input)
		result.Iterations++
		if failure != nil {
			result.Failures = append(result.Failures, *failure)
		}
		i++
	}

	result.Duration = time.Since(start)
	return result, nil
}

// executeAndCheck runs a single input against the connector and returns a
// failure if the execution errors.
func (r *Runner) executeAndCheck(ctx context.Context, target *Target, input map[string]any) *FuzzFailure {
	step := connector.Step{
		Action:     target.Action,
		Parameters: input,
	}

	_, err := r.connector.Execute(ctx, step)
	if err != nil {
		return &FuzzFailure{
			Input: input,
			Error: err.Error(),
		}
	}
	return nil
}

// copyMap creates a shallow copy of a map.
func copyMap(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
