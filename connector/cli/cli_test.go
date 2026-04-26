package cli

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
}

func TestName(t *testing.T) {
	c := New()
	if got := c.Name(); got != "cli" {
		t.Errorf("Name() = %q, want %q", got, "cli")
	}
}

func TestSetupWithWorkDir(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"work_dir": "/tmp",
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if c.workDir != "/tmp" {
		t.Errorf("workDir = %q, want %q", c.workDir, "/tmp")
	}
}

func TestSetupWithEnv(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"env": map[string]string{
			"FOO": "bar",
		},
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if len(c.env) != 1 || c.env[0] != "FOO=bar" {
		t.Errorf("env = %v, want [FOO=bar]", c.env)
	}
}

func TestSetupWithEnvAnyMap(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"env": map[string]any{
			"KEY": "value",
		},
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if len(c.env) != 1 || c.env[0] != "KEY=value" {
		t.Errorf("env = %v, want [KEY=value]", c.env)
	}
}

func TestSetupWithEmptyConfig(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
}

func TestSetupWithNilConfig(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), nil)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
}

func TestSetupInvalidWorkDir(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"work_dir": 123,
	})
	if err == nil {
		t.Fatal("Setup() expected error for non-string work_dir")
	}
}

func TestSetupInvalidEnvType(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"env": "not-a-map",
	})
	if err == nil {
		t.Fatal("Setup() expected error for non-map env")
	}
}

func TestSetupInvalidEnvValueType(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"env": map[string]any{
			"KEY": 123,
		},
	})
	if err == nil {
		t.Fatal("Setup() expected error for non-string env value")
	}
}

func TestTeardown(t *testing.T) {
	c := New()
	err := c.Teardown(context.Background())
	if err != nil {
		t.Fatalf("Teardown() error = %v", err)
	}
}
