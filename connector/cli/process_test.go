package cli

import (
	"context"
	"testing"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
)

func TestExecEchoCommand(t *testing.T) {
	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "echo hello",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	stdout := result.Data["stdout"].(string)
	if stdout != "hello\n" {
		t.Errorf("stdout = %q, want %q", stdout, "hello\n")
	}
	exitCode := result.Data["exit_code"].(int)
	if exitCode != 0 {
		t.Errorf("exit_code = %d, want 0", exitCode)
	}
	cmd := result.Data["command"].(string)
	if cmd != "echo hello" {
		t.Errorf("command = %q, want %q", cmd, "echo hello")
	}
}

func TestExecStdoutCaptured(t *testing.T) {
	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "echo test output",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	stdout := result.Data["stdout"].(string)
	if stdout != "test output\n" {
		t.Errorf("stdout = %q, want %q", stdout, "test output\n")
	}
	stderr := result.Data["stderr"].(string)
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestExecExitCodeSuccess(t *testing.T) {
	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "true",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	exitCode := result.Data["exit_code"].(int)
	if exitCode != 0 {
		t.Errorf("exit_code = %d, want 0", exitCode)
	}
}

func TestExecCommandFails(t *testing.T) {
	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "false",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	exitCode := result.Data["exit_code"].(int)
	if exitCode == 0 {
		t.Error("exit_code = 0, want non-zero")
	}
}

func TestExecWithStdin(t *testing.T) {
	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "cat",
			"stdin":   "hello from stdin",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	stdout := result.Data["stdout"].(string)
	if stdout != "hello from stdin" {
		t.Errorf("stdout = %q, want %q", stdout, "hello from stdin")
	}
}

func TestExecWithTimeout(t *testing.T) {
	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "sleep 60",
		},
		Timeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("Execute() expected timeout error")
	}
}

func TestExecWithCancelledContext(t *testing.T) {
	c := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.
	_, err := c.Execute(ctx, connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "sleep 60",
		},
	})
	if err == nil {
		t.Fatal("Execute() expected error for cancelled context")
	}
}

func TestExecUnknownAction(t *testing.T) {
	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action:     "unknown",
		Parameters: map[string]any{},
	})
	if err == nil {
		t.Fatal("Execute() expected error for unknown action")
	}
}

func TestExecMissingCommand(t *testing.T) {
	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action:     "exec",
		Parameters: map[string]any{},
	})
	if err == nil {
		t.Fatal("Execute() expected error for missing command")
	}
}

func TestExecEmptyCommand(t *testing.T) {
	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "",
		},
	})
	if err == nil {
		t.Fatal("Execute() expected error for empty command")
	}
}

func TestExecInvalidCommandType(t *testing.T) {
	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": 123,
		},
	})
	if err == nil {
		t.Fatal("Execute() expected error for non-string command")
	}
}

func TestExecInvalidStdinType(t *testing.T) {
	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "cat",
			"stdin":   123,
		},
	})
	if err == nil {
		t.Fatal("Execute() expected error for non-string stdin")
	}
}

func TestExecQuotedArguments(t *testing.T) {
	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": `echo "hello world"`,
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	stdout := result.Data["stdout"].(string)
	if stdout != "hello world\n" {
		t.Errorf("stdout = %q, want %q", stdout, "hello world\n")
	}
}

func TestExecSingleQuotedArguments(t *testing.T) {
	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "echo 'hello world'",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	stdout := result.Data["stdout"].(string)
	if stdout != "hello world\n" {
		t.Errorf("stdout = %q, want %q", stdout, "hello world\n")
	}
}

func TestExecWithWorkDir(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{
		"work_dir": "/tmp",
	})
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "pwd",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	stdout := result.Data["stdout"].(string)
	if stdout != "/tmp\n" {
		t.Errorf("stdout = %q, want %q", stdout, "/tmp\n")
	}
}

func TestExecWithEnv(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{
		"env": map[string]any{
			"MY_TEST_VAR": "hello123",
		},
	})
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "/bin/sh -c 'echo $MY_TEST_VAR'",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	stdout := result.Data["stdout"].(string)
	if stdout != "hello123\n" {
		t.Errorf("stdout = %q, want %q", stdout, "hello123\n")
	}
}

func TestExecResultMeta(t *testing.T) {
	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "true",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Meta["connector"] != "cli" {
		t.Errorf("Meta[connector] = %q, want %q", result.Meta["connector"], "cli")
	}
	if result.Meta["action"] != "exec" {
		t.Errorf("Meta[action] = %q, want %q", result.Meta["action"], "exec")
	}
	if result.Elapsed <= 0 {
		t.Error("Elapsed should be positive")
	}
}

func TestExecStderrCaptured(t *testing.T) {
	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "/bin/sh -c 'echo error >&2'",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	stderr := result.Data["stderr"].(string)
	if stderr != "error\n" {
		t.Errorf("stderr = %q, want %q", stderr, "error\n")
	}
}

func TestExecNonexistentCommand(t *testing.T) {
	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": "/nonexistent/command/xyz",
		},
	})
	if err == nil {
		t.Fatal("Execute() expected error for nonexistent command")
	}
}

func TestExecUnterminatedQuote(t *testing.T) {
	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action: "exec",
		Parameters: map[string]any{
			"command": `echo "unterminated`,
		},
	})
	if err == nil {
		t.Fatal("Execute() expected error for unterminated quote")
	}
}

func TestBuildCommandWindows(t *testing.T) {
	orig := goos
	goos = "windows"
	defer func() { goos = orig }()

	args, err := buildCommand("echo hello")
	if err != nil {
		t.Fatalf("buildCommand() error = %v", err)
	}
	if len(args) != 3 || args[0] != "cmd.exe" || args[1] != "/C" || args[2] != "echo hello" {
		t.Errorf("args = %v, want [cmd.exe /C echo hello]", args)
	}
}

func TestBuildCommandLinux(t *testing.T) {
	orig := goos
	goos = "linux"
	defer func() { goos = orig }()

	args, err := buildCommand("echo hello")
	if err != nil {
		t.Fatalf("buildCommand() error = %v", err)
	}
	if len(args) != 2 || args[0] != "echo" || args[1] != "hello" {
		t.Errorf("args = %v, want [echo hello]", args)
	}
}

func TestBuildCommandEmpty(t *testing.T) {
	orig := goos
	goos = "linux"
	defer func() { goos = orig }()

	_, err := buildCommand("   ")
	if err == nil {
		t.Fatal("expected error for empty command")
	}
}
