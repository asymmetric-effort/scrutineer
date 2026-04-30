// Package fleet defines the provider interface and orchestration logic
// for distributed test execution across fleet providers.
package fleet

import (
	"context"
	"time"
)

// Provider is the interface that fleet providers implement to manage
// remote test execution hosts.
type Provider interface {
	// Name returns the provider identifier (e.g., "static", "aws_ec2").
	Name() string

	// Setup initializes the provider with provider-specific configuration.
	Setup(ctx context.Context, config map[string]any) error

	// Acquire provisions or reserves n hosts and returns their handles.
	Acquire(ctx context.Context, n int) ([]Host, error)

	// Push distributes artifacts (binary, test files) to a host.
	Push(ctx context.Context, host Host, artifacts []string) error

	// Execute runs a command on the specified host.
	Execute(ctx context.Context, host Host, cmd string) (*ExecResult, error)

	// Release releases hosts back to the provider.
	Release(ctx context.Context, hosts []Host) error

	// Teardown cleans up all provider resources.
	Teardown(ctx context.Context) error
}

// Host represents a remote execution host.
type Host struct {
	ID       string
	Address  string
	Provider string
	Meta     map[string]string
	BornAt   time.Time
}

// ExecResult holds the result of a remote command execution.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Elapsed  time.Duration
}
