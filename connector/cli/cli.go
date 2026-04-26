// Package cli implements the CLI connector for scrutineer.
//
// The CLI connector executes local commands, captures their output (stdout,
// stderr, exit code), and validates filesystem side-effects. It implements
// the connector.Connector interface from the core module.
package cli

import (
	"context"
	"fmt"

	"github.com/scrutineer/scrutineer/core/connector"
)

// CLIConnector implements connector.Connector for local command execution.
type CLIConnector struct {
	workDir string
	env     []string
}

// Compile-time check that CLIConnector implements connector.Connector.
var _ connector.Connector = (*CLIConnector)(nil)

// New creates a new CLIConnector with default settings.
func New() *CLIConnector {
	return &CLIConnector{}
}

// Name returns the connector identifier used in YAML definitions.
func (c *CLIConnector) Name() string {
	return "cli"
}

// Setup initializes the connector with the given configuration.
//
// Accepted config keys:
//   - "work_dir" (string): working directory for command execution
//   - "env" (map[string]string): environment variables to set
func (c *CLIConnector) Setup(_ context.Context, config map[string]any) error {
	if config == nil {
		return nil
	}

	if wd, ok := config["work_dir"]; ok {
		s, ok := wd.(string)
		if !ok {
			return fmt.Errorf("cli: work_dir must be a string, got %T", wd)
		}
		c.workDir = s
	}

	if envVal, ok := config["env"]; ok {
		envMap, ok := envVal.(map[string]string)
		if !ok {
			// Also accept map[string]any for flexibility.
			anyMap, ok2 := envVal.(map[string]any)
			if !ok2 {
				return fmt.Errorf("cli: env must be a map[string]string, got %T", envVal)
			}
			for k, v := range anyMap {
				s, ok := v.(string)
				if !ok {
					return fmt.Errorf("cli: env value for key %q must be a string, got %T", k, v)
				}
				c.env = append(c.env, k+"="+s)
			}
		} else {
			for k, v := range envMap {
				c.env = append(c.env, k+"="+v)
			}
		}
	}

	return nil
}

// Execute dispatches a step to the appropriate handler based on the action.
func (c *CLIConnector) Execute(ctx context.Context, step connector.Step) (*connector.Result, error) {
	switch step.Action {
	case "exec":
		return c.executeExec(ctx, step)
	case "filesystem":
		return c.executeFilesystem(ctx, step)
	default:
		return nil, fmt.Errorf("cli: unknown action %q", step.Action)
	}
}

// Teardown cleans up resources. For the CLI connector this is a no-op.
func (c *CLIConnector) Teardown(_ context.Context) error {
	return nil
}
