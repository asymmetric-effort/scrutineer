package cli

import (
	"os/exec"
	"testing"
)

func TestWriteStdinEmpty(t *testing.T) {
	cmd := exec.Command("true")
	pipe, err := writeStdin(cmd, "")
	if err != nil {
		t.Fatalf("writeStdin() error = %v", err)
	}
	if pipe != nil {
		t.Error("pipe should be nil for empty stdin")
	}
}

func TestWriteStdinNonEmpty(t *testing.T) {
	cmd := exec.Command("cat")
	pipe, err := writeStdin(cmd, "hello")
	if err != nil {
		t.Fatalf("writeStdin() error = %v", err)
	}
	if pipe == nil {
		t.Error("pipe should not be nil for non-empty stdin")
	}
}

func TestFeedStdinNil(t *testing.T) {
	err := feedStdin(nil, "")
	if err != nil {
		t.Fatalf("feedStdin(nil) error = %v", err)
	}
}

func TestFeedStdinWithData(t *testing.T) {
	cmd := exec.Command("cat")
	pipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	err = feedStdin(pipe, "hello")
	if err != nil {
		t.Fatalf("feedStdin() error = %v", err)
	}
	_ = cmd.Wait()
}

func TestWriteStdinAfterStart(t *testing.T) {
	// writeStdin on an already-started command should fail.
	cmd := exec.Command("cat")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_, err := writeStdin(cmd, "hello")
	if err == nil {
		t.Error("expected error calling writeStdin after Start")
	}
	_ = cmd.Wait()
}

func TestFeedStdinWriteError(t *testing.T) {
	// Create a pipe and close it, then try to write -- triggers write error.
	cmd := exec.Command("true")
	pipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// Close the pipe first to cause a write error.
	pipe.Close()
	err = feedStdin(pipe, "data")
	if err == nil {
		t.Error("expected error writing to closed pipe")
	}
	_ = cmd.Wait()
}
