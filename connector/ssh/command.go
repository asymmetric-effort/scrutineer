package ssh

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
	cryptossh "golang.org/x/crypto/ssh"
)

// executeCommand runs a remote command via SSH.
//
// Parameters:
//   - "command" (string, required): the command to execute
//   - "stdin" (string, optional): data to send to the command's stdin
//
// Result data:
//   - "stdout" (string): captured standard output
//   - "stderr" (string): captured standard error
//   - "exit_code" (int): the command's exit code
func (c *SSHConnector) executeCommand(ctx context.Context, step connector.Step) (*connector.Result, error) {
	cmd, err := requireParam(step.Parameters, "command")
	if err != nil {
		return nil, err
	}

	start := time.Now()

	session, err := c.newSession()
	if err != nil {
		return nil, err
	}
	defer closeSession(session)

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Provide stdin if specified.
	if stdinStr, ok := step.Parameters["stdin"]; ok {
		s, isStr := stdinStr.(string)
		if !isStr {
			return nil, fmt.Errorf("ssh: stdin parameter must be a string")
		}
		session.Stdin = strings.NewReader(s)
	}

	// Handle context cancellation / timeout.
	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	var runErr error
	select {
	case <-ctx.Done():
		// Send signal to remote process.
		session.Signal(cryptossh.SIGKILL)
		return nil, ctx.Err()
	case runErr = <-done:
	}

	exitCode := 0
	if runErr != nil {
		exitCode = extractExitCode(runErr)
	}

	elapsed := time.Since(start)

	return &connector.Result{
		Data: map[string]any{
			"stdout":    stdout.String(),
			"stderr":    stderr.String(),
			"exit_code": exitCode,
		},
		Elapsed: elapsed,
		Meta: map[string]string{
			"connector": "ssh",
			"action":    "exec",
			"command":   cmd,
		},
	}, nil
}

// requireParam extracts a required string parameter from the step parameters.
func requireParam(params map[string]any, key string) (string, error) {
	v, ok := params[key]
	if !ok {
		return "", fmt.Errorf("ssh: missing required parameter %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("ssh: parameter %q must be a string", key)
	}
	return s, nil
}

// extractExitCode extracts the exit code from an SSH session error.
func extractExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*cryptossh.ExitError); ok {
		return exitErr.ExitStatus()
	}
	// If it's not an ExitError, treat it as a generic failure.
	return -1
}
