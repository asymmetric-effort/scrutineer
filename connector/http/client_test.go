package http

import (
	"testing"
	"time"
)

func TestBuildClientWithDefaults(t *testing.T) {
	client, err := buildClient(map[string]any{}, 0)
	if err != nil {
		t.Fatalf("buildClient() error: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil")
	}
	if client.Timeout != 0 {
		t.Fatalf("timeout = %v, want 0", client.Timeout)
	}
}

func TestBuildClientWithTimeout(t *testing.T) {
	client, err := buildClient(map[string]any{}, 30*time.Second)
	if err != nil {
		t.Fatalf("buildClient() error: %v", err)
	}
	if client.Timeout != 30*time.Second {
		t.Fatalf("timeout = %v, want 30s", client.Timeout)
	}
}

func TestBuildClientWithTLSSettings(t *testing.T) {
	client, err := buildClient(map[string]any{
		"tls_skip_verify": true,
	}, 10*time.Second)
	if err != nil {
		t.Fatalf("buildClient() error: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil")
	}
	if client.Timeout != 10*time.Second {
		t.Fatalf("timeout = %v, want 10s", client.Timeout)
	}
}

func TestBuildClientWithInvalidTLS(t *testing.T) {
	_, err := buildClient(map[string]any{
		"tls_skip_verify": "invalid",
	}, 0)
	if err == nil {
		t.Fatal("expected error for invalid TLS config")
	}
}
