package http

import (
	"context"
	"encoding/base64"
	"net/http/httptest"
	"testing"

	nethttp "net/http"
)

func TestBearerAuth(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my-token-123" {
			t.Fatalf("Authorization = %q, want %q", auth, "Bearer my-token-123")
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
		"auth": map[string]any{
			"type":  "bearer",
			"token": "my-token-123",
		},
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestBasicAuth(t *testing.T) {
	expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		auth := r.Header.Get("Authorization")
		if auth != expected {
			t.Fatalf("Authorization = %q, want %q", auth, expected)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
		"auth": map[string]any{
			"type":     "basic",
			"username": "user",
			"password": "pass",
		},
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestAPIKeyAuth(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		key := r.Header.Get("X-API-Key")
		if key != "secret-key" {
			t.Fatalf("X-API-Key = %q, want %q", key, "secret-key")
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
		"auth": map[string]any{
			"type":   "api_key",
			"header": "X-API-Key",
			"key":    "secret-key",
		},
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestNoAuth(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Fatalf("Authorization should be empty, got %q", auth)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestInvalidAuthType(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": "http://localhost"})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
		"auth": map[string]any{
			"type": "oauth",
		},
	}))
	if err == nil {
		t.Fatal("expected error for unsupported auth type")
	}
}

func TestInvalidAuthNotMap(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": "http://localhost"})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
		"auth":   "not-a-map",
	}))
	if err == nil {
		t.Fatal("expected error for non-map auth")
	}
}

func TestInvalidAuthTypeMissing(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": "http://localhost"})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
		"auth":   map[string]any{},
	}))
	if err == nil {
		t.Fatal("expected error for missing auth.type")
	}
}

func TestBearerAuthMissingToken(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": "http://localhost"})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
		"auth": map[string]any{
			"type": "bearer",
		},
	}))
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestBasicAuthMissingUsername(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": "http://localhost"})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
		"auth": map[string]any{
			"type":     "basic",
			"password": "pass",
		},
	}))
	if err == nil {
		t.Fatal("expected error for missing username")
	}
}

func TestBasicAuthMissingPassword(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": "http://localhost"})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
		"auth": map[string]any{
			"type":     "basic",
			"username": "user",
		},
	}))
	if err == nil {
		t.Fatal("expected error for missing password")
	}
}

func TestAPIKeyAuthMissingHeader(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": "http://localhost"})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
		"auth": map[string]any{
			"type": "api_key",
			"key":  "value",
		},
	}))
	if err == nil {
		t.Fatal("expected error for missing header")
	}
}

func TestAPIKeyAuthMissingKey(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": "http://localhost"})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
		"auth": map[string]any{
			"type":   "api_key",
			"header": "X-Key",
		},
	}))
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}
