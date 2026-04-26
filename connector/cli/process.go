package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
)

// executeExec handles the "exec" action. It runs a command, captures stdout
// and stderr, and returns the exit code.
func (c *CLIConnector) executeExec(ctx context.Context, step connector.Step) (*connector.Result, error) {
	cmdStr, ok, err := paramString(step.Parameters, "command")
	if err != nil {
		return nil, err
	}
	if !ok || cmdStr == "" {
		return nil, fmt.Errorf("cli: exec requires a \"command\" parameter")
	}

	stdinData, _, err := paramString(step.Parameters, "stdin")
	if err != nil {
		return nil, err
	}

	// Apply timeout from step if set.
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, step.Timeout)
		defer cancel()
	}

	// Parse command into args.
	args, err := buildCommand(cmdStr)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	if c.workDir != "" {
		cmd.Dir = c.workDir
	}
	if len(c.env) > 0 {
		cmd.Env = append(cmd.Environ(), c.env...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set up stdin pipe if needed.
	pipe, err := writeStdin(cmd, stdinData)
	if err != nil {
		return nil, err
	}

	start := time.Now()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cli: failed to start command: %w", err)
	}

	// Feed stdin data after process start.
	if err := feedStdin(pipe, stdinData); err != nil {
		return nil, err
	}

	exitCode := 0
	runErr := cmd.Wait()
	elapsed := time.Since(start)

	if runErr != nil {
		// Check context first: when the context expires, CommandContext
		// kills the process, producing an ExitError. We want to report
		// the timeout/cancellation, not a normal non-zero exit.
		if ctx.Err() != nil {
			return nil, fmt.Errorf("cli: command timed out or was cancelled: %w", ctx.Err())
		}
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("cli: command error: %w", runErr)
		}
	}

	result := &connector.Result{
		Data: map[string]any{
			"stdout":    stdout.String(),
			"stderr":    stderr.String(),
			"exit_code": exitCode,
			"command":   cmdStr,
		},
		Elapsed: elapsed,
		Meta: map[string]string{
			"connector": "cli",
			"action":    "exec",
		},
	}

	return result, nil
}

// goos is the current operating system, exposed as a variable for testing.
var goos = runtime.GOOS

// buildCommand parses a command string into arguments. On Windows, if the
// command appears to use shell features, it wraps with cmd.exe.
func buildCommand(cmdStr string) ([]string, error) {
	if goos == "windows" {
		// On Windows, use cmd.exe /C to handle shell features.
		return []string{"cmd.exe", "/C", cmdStr}, nil
	}

	args, err := parseShellArgs(cmdStr)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("cli: empty command")
	}
	return args, nil
}
