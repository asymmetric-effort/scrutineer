package fuzz

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
)

// mockConnector implements connector.Connector for testing.
type mockConnector struct {
	name     string
	failEvery int // fail every N-th call; 0 = never fail
	calls    int
}

func (m *mockConnector) Name() string { return m.name }

func (m *mockConnector) Setup(_ context.Context, _ map[string]any) error { return nil }

func (m *mockConnector) Execute(_ context.Context, _ connector.Step) (*connector.Result, error) {
	m.calls++
	if m.failEvery > 0 && m.calls%m.failEvery == 0 {
		return nil, fmt.Errorf("simulated failure on call %d", m.calls)
	}
	return &connector.Result{
		Data:    map[string]any{"status": "ok"},
		Elapsed: time.Millisecond,
	}, nil
}

func (m *mockConnector) Teardown(_ context.Context) error { return nil }

func validTarget() *Target {
	return &Target{
		Name:       "test-target",
		Connector:  "mock",
		Action:     "do-something",
		Parameters: map[string]any{"field1": "hello", "field2": 42},
		FuzzFields: []string{"field1"},
	}
}

func validParams() map[string]any {
	return map[string]any{
		"name":        "test-target",
		"connector":   "mock",
		"action":      "do-something",
		"parameters":  map[string]any{"field1": "hello", "field2": 42},
		"fuzz_fields": []any{"field1"},
	}
}

func TestParseTarget_Valid(t *testing.T) {
	target, err := ParseTarget(validParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Name != "test-target" {
		t.Errorf("expected name test-target, got %s", target.Name)
	}
	if target.Connector != "mock" {
		t.Errorf("expected connector mock, got %s", target.Connector)
	}
	if target.Action != "do-something" {
		t.Errorf("expected action do-something, got %s", target.Action)
	}
	if len(target.FuzzFields) != 1 || target.FuzzFields[0] != "field1" {
		t.Errorf("unexpected fuzz fields: %v", target.FuzzFields)
	}
}

func TestParseTarget_WithSeedAndAssert(t *testing.T) {
	params := validParams()
	params["seed"] = []any{map[string]any{"field1": "seed1"}}
	params["assert"] = []any{map[string]any{"type": "not_nil"}}

	target, err := ParseTarget(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(target.Seed) != 1 {
		t.Errorf("expected 1 seed entry, got %d", len(target.Seed))
	}
	if len(target.Assert) != 1 {
		t.Errorf("expected 1 assert entry, got %d", len(target.Assert))
	}
}

func TestParseTarget_MissingName(t *testing.T) {
	params := validParams()
	delete(params, "name")
	_, err := ParseTarget(params)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseTarget_MissingConnector(t *testing.T) {
	params := validParams()
	delete(params, "connector")
	_, err := ParseTarget(params)
	if err == nil {
		t.Fatal("expected error for missing connector")
	}
}

func TestParseTarget_MissingAction(t *testing.T) {
	params := validParams()
	delete(params, "action")
	_, err := ParseTarget(params)
	if err == nil {
		t.Fatal("expected error for missing action")
	}
}

func TestParseTarget_MissingFuzzFields(t *testing.T) {
	params := validParams()
	delete(params, "fuzz_fields")
	_, err := ParseTarget(params)
	if err == nil {
		t.Fatal("expected error for missing fuzz_fields")
	}
}

func TestParseTarget_InvalidNameType(t *testing.T) {
	params := validParams()
	params["name"] = 123
	_, err := ParseTarget(params)
	if err == nil {
		t.Fatal("expected error for non-string name")
	}
}

func TestParseTarget_InvalidConnectorType(t *testing.T) {
	params := validParams()
	params["connector"] = 123
	_, err := ParseTarget(params)
	if err == nil {
		t.Fatal("expected error for non-string connector")
	}
}

func TestParseTarget_InvalidActionType(t *testing.T) {
	params := validParams()
	params["action"] = 123
	_, err := ParseTarget(params)
	if err == nil {
		t.Fatal("expected error for non-string action")
	}
}

func TestParseTarget_InvalidParametersType(t *testing.T) {
	params := validParams()
	params["parameters"] = "not-a-map"
	_, err := ParseTarget(params)
	if err == nil {
		t.Fatal("expected error for non-map parameters")
	}
}

func TestParseTarget_InvalidFuzzFieldsType(t *testing.T) {
	params := validParams()
	params["fuzz_fields"] = "not-a-list"
	_, err := ParseTarget(params)
	if err == nil {
		t.Fatal("expected error for non-list fuzz_fields")
	}
}

func TestParseTarget_InvalidFuzzFieldEntry(t *testing.T) {
	params := validParams()
	params["fuzz_fields"] = []any{123}
	_, err := ParseTarget(params)
	if err == nil {
		t.Fatal("expected error for non-string fuzz field entry")
	}
}

func TestParseTarget_InvalidSeedType(t *testing.T) {
	params := validParams()
	params["seed"] = "not-a-list"
	_, err := ParseTarget(params)
	if err == nil {
		t.Fatal("expected error for non-list seed")
	}
}

func TestParseTarget_InvalidSeedEntry(t *testing.T) {
	params := validParams()
	params["seed"] = []any{"not-a-map"}
	_, err := ParseTarget(params)
	if err == nil {
		t.Fatal("expected error for non-map seed entry")
	}
}

func TestParseTarget_InvalidAssertType(t *testing.T) {
	params := validParams()
	params["assert"] = "not-a-list"
	_, err := ParseTarget(params)
	if err == nil {
		t.Fatal("expected error for non-list assert")
	}
}

func TestParseTarget_InvalidAssertEntry(t *testing.T) {
	params := validParams()
	params["assert"] = []any{"not-a-map"}
	_, err := ParseTarget(params)
	if err == nil {
		t.Fatal("expected error for non-map assert entry")
	}
}

func TestNewRunner(t *testing.T) {
	mc := &mockConnector{name: "mock"}
	r := NewRunner(mc)
	if r == nil {
		t.Fatal("expected non-nil runner")
	}
	if r.connector != mc {
		t.Error("runner connector mismatch")
	}
}

func TestFuzz_AlwaysSucceeds(t *testing.T) {
	mc := &mockConnector{name: "mock"}
	r := NewRunner(mc)
	target := validTarget()

	result, err := r.Fuzz(context.Background(), target, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Iterations != 10 {
		t.Errorf("expected 10 iterations, got %d", result.Iterations)
	}
	if len(result.Failures) != 0 {
		t.Errorf("expected 0 failures, got %d", len(result.Failures))
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestFuzz_WithSeedCorpus(t *testing.T) {
	mc := &mockConnector{name: "mock"}
	r := NewRunner(mc)
	target := validTarget()
	target.Seed = []map[string]any{
		{"field1": "seed-value-1"},
		{"field1": "seed-value-2"},
	}

	result, err := r.Fuzz(context.Background(), target, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 2 seed + 5 generated = 7
	if result.Iterations != 7 {
		t.Errorf("expected 7 iterations, got %d", result.Iterations)
	}
}

func TestFuzz_SometimesFails(t *testing.T) {
	mc := &mockConnector{name: "mock", failEvery: 3}
	r := NewRunner(mc)
	target := validTarget()

	result, err := r.Fuzz(context.Background(), target, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Iterations != 9 {
		t.Errorf("expected 9 iterations, got %d", result.Iterations)
	}
	if len(result.Failures) != 3 {
		t.Errorf("expected 3 failures, got %d", len(result.Failures))
	}
	for _, f := range result.Failures {
		if f.Error == "" {
			t.Error("expected non-empty error string")
		}
		if f.Input == nil {
			t.Error("expected non-nil input")
		}
	}
}

func TestFuzz_ContextCancellation(t *testing.T) {
	mc := &mockConnector{name: "mock"}
	r := NewRunner(mc)
	target := validTarget()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	result, err := r.Fuzz(ctx, target, 0)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result even on cancellation")
	}
}

func TestFuzz_ContextCancellationDuringSeed(t *testing.T) {
	mc := &mockConnector{name: "mock"}
	r := NewRunner(mc)
	target := validTarget()
	target.Seed = []map[string]any{
		{"field1": "s1"},
		{"field1": "s2"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := r.Fuzz(ctx, target, 10)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestFuzz_InvalidTarget(t *testing.T) {
	mc := &mockConnector{name: "mock"}
	r := NewRunner(mc)
	target := &Target{} // invalid

	_, err := r.Fuzz(context.Background(), target, 10)
	if err == nil {
		t.Fatal("expected error for invalid target")
	}
}

func TestFuzzResult_Fields(t *testing.T) {
	r := &FuzzResult{
		Iterations: 42,
		Failures: []FuzzFailure{
			{Input: map[string]any{"x": 1}, Error: "boom"},
		},
		Duration: 5 * time.Second,
	}
	if r.Iterations != 42 {
		t.Errorf("expected 42 iterations, got %d", r.Iterations)
	}
	if len(r.Failures) != 1 {
		t.Errorf("expected 1 failure, got %d", len(r.Failures))
	}
	if r.Failures[0].Error != "boom" {
		t.Errorf("expected error boom, got %s", r.Failures[0].Error)
	}
	if r.Duration != 5*time.Second {
		t.Errorf("expected 5s duration, got %v", r.Duration)
	}
}

func TestCopyMap(t *testing.T) {
	orig := map[string]any{"a": 1, "b": "two"}
	cp := copyMap(orig)
	cp["a"] = 99
	if orig["a"] != 1 {
		t.Error("copyMap should not modify original")
	}
}
