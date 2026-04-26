package cli

import (
	"fmt"
	"io"
	"os/exec"
)

// writeStdin writes the given data to the process stdin pipe and closes it.
// If stdinData is empty, this is a no-op. The pipe must be obtained before
// the process is started.
func writeStdin(cmd *exec.Cmd, stdinData string) (io.WriteCloser, error) {
	if stdinData == "" {
		return nil, nil
	}

	pipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("cli: failed to create stdin pipe: %w", err)
	}

	return pipe, nil
}

// feedStdin writes data to the pipe and closes it. Must be called after
// the process has started.
func feedStdin(pipe io.WriteCloser, data string) error {
	if pipe == nil {
		return nil
	}
	_, err := io.WriteString(pipe, data)
	if err != nil {
		pipe.Close()
		return fmt.Errorf("cli: failed to write to stdin: %w", err)
	}
	return pipe.Close()
}
