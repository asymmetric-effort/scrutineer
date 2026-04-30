package static

import (
	"context"
	"fmt"
	"testing"

	"github.com/scrutineer/scrutineer/core/fleet"

	"golang.org/x/crypto/ssh"
)

func TestProviderName(t *testing.T) {
	p := New()
	if p.Name() != "static" {
		t.Errorf("name = %q", p.Name())
	}
}

func TestProviderSatisfiesInterface(t *testing.T) {
	var _ fleet.Provider = New()
}

func TestSetupValid(t *testing.T) {
	p := New()
	config := map[string]any{
		"ssh": map[string]any{
			"user":     "testuser",
			"key_file": "/tmp/test_key",
			"port":     2222,
		},
		"binary": "/usr/local/bin/scrutineer",
		"nodes":  []any{"10.0.1.1", "10.0.1.2"},
	}
	err := p.Setup(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.config.SSH.User != "testuser" {
		t.Errorf("user = %q", p.config.SSH.User)
	}
	if p.config.SSH.KeyFile != "/tmp/test_key" {
		t.Errorf("key_file = %q", p.config.SSH.KeyFile)
	}
	if p.config.SSH.Port != 2222 {
		t.Errorf("port = %d", p.config.SSH.Port)
	}
	if p.config.Binary != "/usr/local/bin/scrutineer" {
		t.Errorf("binary = %q", p.config.Binary)
	}
	if len(p.config.Nodes) != 2 {
		t.Errorf("nodes = %d", len(p.config.Nodes))
	}
}

func TestSetupDefaultPort(t *testing.T) {
	p := New()
	config := map[string]any{
		"ssh": map[string]any{
			"user":     "testuser",
			"key_file": "/tmp/test_key",
		},
		"nodes": []any{"10.0.1.1"},
	}
	err := p.Setup(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.config.SSH.Port != 22 {
		t.Errorf("default port = %d, want 22", p.config.SSH.Port)
	}
}

func TestSetupNilConfig(t *testing.T) {
	p := New()
	err := p.Setup(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestSetupMissingNodes(t *testing.T) {
	p := New()
	config := map[string]any{
		"ssh": map[string]any{
			"user":     "testuser",
			"key_file": "/tmp/test_key",
		},
	}
	err := p.Setup(context.Background(), config)
	if err == nil {
		t.Fatal("expected error for missing nodes")
	}
}

func TestSetupMissingKeyFile(t *testing.T) {
	p := New()
	config := map[string]any{
		"ssh": map[string]any{
			"user": "testuser",
		},
		"nodes": []any{"10.0.1.1"},
	}
	err := p.Setup(context.Background(), config)
	if err == nil {
		t.Fatal("expected error for missing key_file")
	}
}

func TestSetupMissingUser(t *testing.T) {
	p := New()
	config := map[string]any{
		"ssh": map[string]any{
			"key_file": "/tmp/test_key",
		},
		"nodes": []any{"10.0.1.1"},
	}
	err := p.Setup(context.Background(), config)
	if err == nil {
		t.Fatal("expected error for missing user")
	}
}

func TestAcquireAll(t *testing.T) {
	p := New()
	p.config.Nodes = []string{"10.0.1.1", "10.0.1.2", "10.0.1.3"}

	hosts, err := p.Acquire(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 3 {
		t.Errorf("expected 3 hosts, got %d", len(hosts))
	}
	if hosts[0].Address != "10.0.1.1" {
		t.Errorf("host[0] address = %q", hosts[0].Address)
	}
	if hosts[0].Provider != "static" {
		t.Errorf("host[0] provider = %q", hosts[0].Provider)
	}
}

func TestAcquireSubset(t *testing.T) {
	p := New()
	p.config.Nodes = []string{"10.0.1.1", "10.0.1.2", "10.0.1.3"}

	hosts, err := p.Acquire(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(hosts))
	}
}

func TestAcquireTooMany(t *testing.T) {
	p := New()
	p.config.Nodes = []string{"10.0.1.1"}

	_, err := p.Acquire(context.Background(), 5)
	if err == nil {
		t.Fatal("expected error for too many hosts")
	}
}

func TestReleaseNoOp(t *testing.T) {
	p := New()
	err := p.Release(context.Background(), []fleet.Host{{ID: "h1"}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecuteWithMock(t *testing.T) {
	// Replace the runCommand function for testing.
	origRunCmd := runCommand
	defer func() { runCommand = origRunCmd }()
	runCommand = func(_ context.Context, _ *ssh.Client, cmd string) (string, string, int, error) {
		return "hello", "", 0, nil
	}

	// Also replace dialSSH so getOrDial succeeds.
	origDial := dialSSH
	defer func() { dialSSH = origDial }()
	dialSSH = func(address string, cfg SSHConfig) (*ssh.Client, error) {
		// Return a nil client — our mock runCommand doesn't use it.
		return &ssh.Client{}, nil
	}

	p := New()
	p.config.SSH = SSHConfig{User: "test", KeyFile: "/tmp/key", Port: 22}

	host := fleet.Host{ID: "h1", Address: "10.0.1.1", Provider: "static"}
	result, err := p.Execute(context.Background(), host, "echo hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Stdout != "hello" {
		t.Errorf("stdout = %q", result.Stdout)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d", result.ExitCode)
	}
}

func TestExecuteDialError(t *testing.T) {
	origDial := dialSSH
	defer func() { dialSSH = origDial }()
	dialSSH = func(address string, cfg SSHConfig) (*ssh.Client, error) {
		return nil, fmt.Errorf("connection refused")
	}

	p := New()
	p.config.SSH = SSHConfig{User: "test", KeyFile: "/tmp/key", Port: 22}

	host := fleet.Host{ID: "h1", Address: "10.0.1.1", Provider: "static"}
	_, err := p.Execute(context.Background(), host, "echo hello")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExecuteCommandError(t *testing.T) {
	origRunCmd := runCommand
	defer func() { runCommand = origRunCmd }()
	runCommand = func(_ context.Context, _ *ssh.Client, cmd string) (string, string, int, error) {
		return "", "error", 1, nil
	}

	origDial := dialSSH
	defer func() { dialSSH = origDial }()
	dialSSH = func(address string, cfg SSHConfig) (*ssh.Client, error) {
		return &ssh.Client{}, nil
	}

	p := New()
	p.config.SSH = SSHConfig{User: "test", KeyFile: "/tmp/key", Port: 22}

	host := fleet.Host{ID: "h1", Address: "10.0.1.1", Provider: "static"}
	result, err := p.Execute(context.Background(), host, "fail")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", result.ExitCode)
	}
	if result.Stderr != "error" {
		t.Errorf("stderr = %q", result.Stderr)
	}
}

func TestTeardownEmptyNoError(t *testing.T) {
	p := New()
	err := p.Teardown(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetOrDialCachesConnection(t *testing.T) {
	dialCount := 0
	origDial := dialSSH
	defer func() { dialSSH = origDial }()
	dialSSH = func(address string, cfg SSHConfig) (*ssh.Client, error) {
		dialCount++
		return &ssh.Client{}, nil
	}

	p := New()
	p.config.SSH = SSHConfig{User: "test", KeyFile: "/tmp/key", Port: 22}

	_, _ = p.getOrDial("10.0.1.1")
	_, _ = p.getOrDial("10.0.1.1")

	if dialCount != 1 {
		t.Errorf("dial count = %d, want 1 (should cache)", dialCount)
	}
}

func TestPushWithMock(t *testing.T) {
	origDial := dialSSH
	defer func() { dialSSH = origDial }()
	dialSSH = func(address string, cfg SSHConfig) (*ssh.Client, error) {
		return &ssh.Client{}, nil
	}

	origScp := scpFile
	defer func() { scpFile = origScp }()
	scpCalls := 0
	scpFile = func(_ context.Context, _ *ssh.Client, local, remote string) error {
		scpCalls++
		return nil
	}

	p := New()
	p.config.SSH = SSHConfig{User: "test", KeyFile: "/tmp/key", Port: 22}

	host := fleet.Host{ID: "h1", Address: "10.0.1.1", Provider: "static"}
	err := p.Push(context.Background(), host, []string{"file1", "file2"})
	if err != nil {
		t.Fatal(err)
	}
	if scpCalls != 2 {
		t.Errorf("scp calls = %d, want 2", scpCalls)
	}
}

func TestPushDialError(t *testing.T) {
	origDial := dialSSH
	defer func() { dialSSH = origDial }()
	dialSSH = func(address string, cfg SSHConfig) (*ssh.Client, error) {
		return nil, fmt.Errorf("refused")
	}

	p := New()
	p.config.SSH = SSHConfig{User: "test", KeyFile: "/tmp/key", Port: 22}

	host := fleet.Host{ID: "h1", Address: "10.0.1.1", Provider: "static"}
	err := p.Push(context.Background(), host, []string{"file1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPushScpError(t *testing.T) {
	origDial := dialSSH
	defer func() { dialSSH = origDial }()
	dialSSH = func(address string, cfg SSHConfig) (*ssh.Client, error) {
		return &ssh.Client{}, nil
	}

	origScp := scpFile
	defer func() { scpFile = origScp }()
	scpFile = func(_ context.Context, _ *ssh.Client, local, remote string) error {
		return fmt.Errorf("disk full")
	}

	p := New()
	p.config.SSH = SSHConfig{User: "test", KeyFile: "/tmp/key", Port: 22}

	host := fleet.Host{ID: "h1", Address: "10.0.1.1", Provider: "static"}
	err := p.Push(context.Background(), host, []string{"file1"})
	if err == nil {
		t.Fatal("expected error")
	}
}
