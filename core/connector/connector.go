// Package connector defines the interface and types for protocol connectors.
//
// Every protocol connector (HTTP, CLI, SSH, etc.) implements the Connector
// interface, which provides a uniform lifecycle: Setup, Execute, Teardown.
// Connectors are registered by name in a Registry so the engine can
// instantiate them from YAML test definitions.
package connector

import (
	"context"
	"time"
)

// Connector is the primary interface every protocol connector implements.
type Connector interface {
	// Name returns the connector identifier used in YAML (e.g. "http", "cli", "ssh").
	Name() string

	// Setup initializes the connector with the given configuration.
	Setup(ctx context.Context, config map[string]any) error

	// Execute runs a single test step and returns the result.
	Execute(ctx context.Context, step Step) (*Result, error)

	// Teardown cleans up resources. Always called, even after failures.
	Teardown(ctx context.Context) error
}

// Step represents a single action within a test, as parsed from YAML.
type Step struct {
	Action     string
	Parameters map[string]any
	Timeout    time.Duration
}

// Result holds the output of a single step execution.
type Result struct {
	Data    map[string]any    // keyed output (e.g. "body", "stdout", "status_code")
	Elapsed time.Duration     // time taken for this step
	Meta    map[string]string // metadata for telemetry
}

// Factory creates a new Connector instance. Used by the registry.
type Factory func() Connector
