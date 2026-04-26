package connector

import (
	"context"
	"testing"
	"time"
)

// mockConnector implements the Connector interface for testing.
type mockConnector struct {
	name string
}

func (m *mockConnector) Name() string { return m.name }

func (m *mockConnector) Setup(_ context.Context, _ map[string]any) error { return nil }

func (m *mockConnector) Execute(_ context.Context, _ Step) (*Result, error) {
	return &Result{
		Data:    map[string]any{"status": "ok"},
		Elapsed: 42 * time.Millisecond,
		Meta:    map[string]string{"connector": m.name},
	}, nil
}

func (m *mockConnector) Teardown(_ context.Context) error { return nil }

func TestStepConstruction(t *testing.T) {
	s := Step{
		Action:     "get",
		Parameters: map[string]any{"url": "https://example.com"},
		Timeout:    5 * time.Second,
	}
	if s.Action != "get" {
		t.Fatalf("expected action 'get', got %q", s.Action)
	}
	if s.Timeout != 5*time.Second {
		t.Fatalf("expected timeout 5s, got %v", s.Timeout)
	}
	if s.Parameters["url"] != "https://example.com" {
		t.Fatalf("expected url parameter, got %v", s.Parameters["url"])
	}
}

func TestResultConstruction(t *testing.T) {
	r := Result{
		Data:    map[string]any{"body": "hello"},
		Elapsed: 100 * time.Millisecond,
		Meta:    map[string]string{"trace_id": "abc123"},
	}
	if r.Data["body"] != "hello" {
		t.Fatalf("expected body 'hello', got %v", r.Data["body"])
	}
	if r.Elapsed != 100*time.Millisecond {
		t.Fatalf("expected elapsed 100ms, got %v", r.Elapsed)
	}
	if r.Meta["trace_id"] != "abc123" {
		t.Fatalf("expected trace_id 'abc123', got %v", r.Meta["trace_id"])
	}
}

func TestConnectorInterface(t *testing.T) {
	var c Connector = &mockConnector{name: "test"}

	if c.Name() != "test" {
		t.Fatalf("expected name 'test', got %q", c.Name())
	}

	ctx := context.Background()

	if err := c.Setup(ctx, nil); err != nil {
		t.Fatalf("unexpected Setup error: %v", err)
	}

	result, err := c.Execute(ctx, Step{Action: "ping"})
	if err != nil {
		t.Fatalf("unexpected Execute error: %v", err)
	}
	if result.Data["status"] != "ok" {
		t.Fatalf("expected status 'ok', got %v", result.Data["status"])
	}
	if result.Elapsed != 42*time.Millisecond {
		t.Fatalf("expected elapsed 42ms, got %v", result.Elapsed)
	}
	if result.Meta["connector"] != "test" {
		t.Fatalf("expected meta connector 'test', got %v", result.Meta["connector"])
	}

	if err := c.Teardown(ctx); err != nil {
		t.Fatalf("unexpected Teardown error: %v", err)
	}
}

func TestFactoryType(t *testing.T) {
	var f Factory = func() Connector {
		return &mockConnector{name: "from-factory"}
	}
	c := f()
	if c.Name() != "from-factory" {
		t.Fatalf("expected name 'from-factory', got %q", c.Name())
	}
}
