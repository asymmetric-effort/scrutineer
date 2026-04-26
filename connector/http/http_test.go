package http

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
	if c.headers == nil {
		t.Fatal("headers map not initialized")
	}
}

func TestName(t *testing.T) {
	c := New()
	if got := c.Name(); got != "http" {
		t.Fatalf("Name() = %q, want %q", got, "http")
	}
}

func TestSetupWithBaseURL(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"base_url": "https://example.com",
	})
	if err != nil {
		t.Fatalf("Setup() error: %v", err)
	}
	if c.baseURL != "https://example.com" {
		t.Fatalf("baseURL = %q, want %q", c.baseURL, "https://example.com")
	}
}

func TestSetupWithDefaultHeaders(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"base_url": "https://example.com",
		"default_headers": map[string]any{
			"X-Custom": "value",
		},
	})
	if err != nil {
		t.Fatalf("Setup() error: %v", err)
	}
	if c.headers["X-Custom"] != "value" {
		t.Fatalf("header X-Custom = %q, want %q", c.headers["X-Custom"], "value")
	}
}

func TestSetupWithDefaultHeadersStringMap(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"default_headers": map[string]string{
			"X-Custom": "value",
		},
	})
	if err != nil {
		t.Fatalf("Setup() error: %v", err)
	}
	if c.headers["X-Custom"] != "value" {
		t.Fatalf("header X-Custom = %q, want %q", c.headers["X-Custom"], "value")
	}
}

func TestSetupEmptyConfig(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Setup() error: %v", err)
	}
	if c.baseURL != "" {
		t.Fatalf("baseURL = %q, want empty", c.baseURL)
	}
}

func TestSetupInvalidBaseURL(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"base_url": 123,
	})
	if err == nil {
		t.Fatal("expected error for non-string base_url")
	}
}

func TestSetupInvalidHeaders(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"default_headers": "not-a-map",
	})
	if err == nil {
		t.Fatal("expected error for non-map default_headers")
	}
}

func TestSetupInvalidTimeout(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"timeout": "not-a-duration",
	})
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

func TestSetupInvalidTimeoutType(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"timeout": 123,
	})
	if err == nil {
		t.Fatal("expected error for non-string timeout")
	}
}

func TestSetupWithTimeout(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"timeout": "30s",
	})
	if err != nil {
		t.Fatalf("Setup() error: %v", err)
	}
}

func TestTeardown(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"base_url": "https://example.com",
	})
	if err != nil {
		t.Fatalf("Setup() error: %v", err)
	}
	err = c.Teardown(context.Background())
	if err != nil {
		t.Fatalf("Teardown() error: %v", err)
	}
}

func TestTeardownNilClient(t *testing.T) {
	c := New()
	err := c.Teardown(context.Background())
	if err != nil {
		t.Fatalf("Teardown() error: %v", err)
	}
}

func TestExecuteUnsupportedAction(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{})
	_, err := c.Execute(context.Background(), connectorStep("unknown", nil))
	if err == nil {
		t.Fatal("expected error for unsupported action")
	}
}
