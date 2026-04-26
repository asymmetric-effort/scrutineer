package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	nethttp "net/http"
)

func TestGetRequest(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"message":"hello"}`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	result, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/test",
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	status := result.Data["status"].(int)
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}

	body := result.Data["body"].(map[string]any)
	if body["message"] != "hello" {
		t.Fatalf("body.message = %v, want hello", body["message"])
	}

	if result.Data["body_raw"] != `{"message":"hello"}` {
		t.Fatalf("body_raw unexpected: %v", result.Data["body_raw"])
	}

	if result.Data["elapsed_ms"] == nil {
		t.Fatal("elapsed_ms missing")
	}

	if result.Data["status_text"] == nil {
		t.Fatal("status_text missing")
	}
}

func TestPostWithJSONBody(t *testing.T) {
	var receivedBody map[string]any
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != "POST" {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &receivedBody)
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	result, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "POST",
		"path":   "/items",
		"body":   map[string]any{"name": "test"},
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if result.Data["status"].(int) != 201 {
		t.Fatalf("status = %v, want 201", result.Data["status"])
	}
	if receivedBody["name"] != "test" {
		t.Fatalf("received body name = %v, want test", receivedBody["name"])
	}
}

func TestPostWithStringBody(t *testing.T) {
	var receivedBody string
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		data, _ := io.ReadAll(r.Body)
		receivedBody = string(data)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`ok`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "POST",
		"path":   "/raw",
		"body":   "raw body content",
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if receivedBody != "raw body content" {
		t.Fatalf("received body = %q, want %q", receivedBody, "raw body content")
	}
}

func TestPutRequest(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != "PUT" {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"updated":true}`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	result, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "PUT",
		"path":   "/items/1",
		"body":   map[string]any{"name": "updated"},
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if result.Data["status"].(int) != 200 {
		t.Fatalf("status = %v, want 200", result.Data["status"])
	}
}

func TestDeleteRequest(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	result, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "DELETE",
		"path":   "/items/1",
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if result.Data["status"].(int) != 204 {
		t.Fatalf("status = %v, want 204", result.Data["status"])
	}
}

func TestCustomHeaders(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Header.Get("X-Custom") != "custom-value" {
			t.Fatalf("X-Custom = %q, want %q", r.Header.Get("X-Custom"), "custom-value")
		}
		if r.Header.Get("X-Default") != "default-value" {
			t.Fatalf("X-Default = %q, want %q", r.Header.Get("X-Default"), "default-value")
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{
		"base_url": srv.URL,
		"default_headers": map[string]any{
			"X-Default": "default-value",
		},
	})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/test",
		"headers": map[string]any{
			"X-Custom": "custom-value",
		},
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestQueryParameters(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Query().Get("page") != "1" {
			t.Fatalf("page = %q, want %q", r.URL.Query().Get("page"), "1")
		}
		if r.URL.Query().Get("limit") != "10" {
			t.Fatalf("limit = %q, want %q", r.URL.Query().Get("limit"), "10")
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/items",
		"query": map[string]any{
			"page":  "1",
			"limit": "10",
		},
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestResponseBodyParsedAsJSON(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[1,2,3],"total":3}`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	result, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	body, ok := result.Data["body"].(map[string]any)
	if !ok {
		t.Fatalf("body is not map[string]any: %T", result.Data["body"])
	}
	if body["total"].(float64) != 3 {
		t.Fatalf("body.total = %v, want 3", body["total"])
	}
}

func TestResponseNonJSONBody(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("plain text response"))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	result, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	body, ok := result.Data["body"].(string)
	if !ok {
		t.Fatalf("body is not string: %T", result.Data["body"])
	}
	if body != "plain text response" {
		t.Fatalf("body = %q, want %q", body, "plain text response")
	}
}

func TestMissingMethod(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": "http://localhost"})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"path": "/test",
	}))
	if err == nil {
		t.Fatal("expected error for missing method")
	}
	if !strings.Contains(err.Error(), "method") {
		t.Fatalf("error should mention method: %v", err)
	}
}

func TestMissingPath(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": "http://localhost"})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
	}))
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	if !strings.Contains(err.Error(), "path") {
		t.Fatalf("error should mention path: %v", err)
	}
}

func TestInvalidQueryType(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": "http://localhost"})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/test",
		"query":  "not-a-map",
	}))
	if err == nil {
		t.Fatal("expected error for invalid query type")
	}
}

func TestInvalidHeadersType(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method":  "GET",
		"path":    "/",
		"headers": "not-a-map",
	}))
	if err == nil {
		t.Fatal("expected error for invalid headers type")
	}
}

func TestRequestWithTimeout(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	result, err := c.Execute(context.Background(), connectorStepWithTimeout("request", map[string]any{
		"method": "GET",
		"path":   "/",
	}, 5*time.Second))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if result.Data["status"].(int) != 200 {
		t.Fatalf("status = %v, want 200", result.Data["status"])
	}
}

func TestResponseHeaders(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("X-Response-Custom", "resp-value")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	result, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/",
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	headers := result.Data["headers"].(map[string][]string)
	if v, ok := headers["X-Response-Custom"]; !ok || v[0] != "resp-value" {
		t.Fatalf("X-Response-Custom header = %v, want [resp-value]", v)
	}
}

func TestBodyWithSlice(t *testing.T) {
	var receivedBody string
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		data, _ := io.ReadAll(r.Body)
		receivedBody = string(data)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "POST",
		"path":   "/",
		"body":   []any{"a", "b", "c"},
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if receivedBody != `["a","b","c"]` {
		t.Fatalf("received body = %q, want %q", receivedBody, `["a","b","c"]`)
	}
}

func TestBodyMarshalError(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	// A channel cannot be marshalled to JSON.
	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "POST",
		"path":   "/",
		"body":   make(chan int),
	}))
	if err == nil {
		t.Fatal("expected error for unmarshalable body")
	}
}

func TestBodyMapMarshalError(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	// A map with a channel value cannot be marshalled to JSON.
	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "POST",
		"path":   "/",
		"body":   map[string]any{"bad": make(chan int)},
	}))
	if err == nil {
		t.Fatal("expected error for unmarshalable map body")
	}
}

func TestSetupWithInvalidTLSPropagatesError(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"tls_skip_verify": "not-a-bool",
	})
	if err == nil {
		t.Fatal("expected error from invalid TLS config")
	}
}

func TestInvalidURLParse(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": "://bad"})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/test",
		"query":  map[string]any{"k": "v"},
	}))
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestInvalidMethod(t *testing.T) {
	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": "http://localhost"})

	_, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "INVALID METHOD WITH SPACES",
		"path":   "/test",
	}))
	if err == nil {
		t.Fatal("expected error for invalid method")
	}
}

func TestResultMeta(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New()
	_ = c.Setup(context.Background(), map[string]any{"base_url": srv.URL})

	result, err := c.Execute(context.Background(), connectorStep("request", map[string]any{
		"method": "GET",
		"path":   "/test",
	}))
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if result.Meta["method"] != "GET" {
		t.Fatalf("meta.method = %q, want GET", result.Meta["method"])
	}
	if !strings.Contains(result.Meta["url"], "/test") {
		t.Fatalf("meta.url should contain /test: %v", result.Meta["url"])
	}
}
