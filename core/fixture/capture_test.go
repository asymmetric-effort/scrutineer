package fixture

import (
	"strings"
	"testing"
)

func TestExtract_SimpleKey(t *testing.T) {
	data := map[string]any{"status": 200}

	val, err := Extract(data, "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 200 {
		t.Fatalf("expected 200, got %v", val)
	}
}

func TestExtract_NestedPath(t *testing.T) {
	data := map[string]any{
		"body": map[string]any{
			"user": map[string]any{
				"id": "abc",
			},
		},
	}

	val, err := Extract(data, "body.user.id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "abc" {
		t.Fatalf("expected 'abc', got %v", val)
	}
}

func TestExtract_DeepPath(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": map[string]any{
					"d": 42,
				},
			},
		},
	}

	val, err := Extract(data, "a.b.c.d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 42 {
		t.Fatalf("expected 42, got %v", val)
	}
}

func TestExtract_MissingKey(t *testing.T) {
	data := map[string]any{"status": 200}

	_, err := Extract(data, "missing")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' in error, got %v", err)
	}
}

func TestExtract_NilMap(t *testing.T) {
	_, err := Extract(nil, "key")
	if err == nil {
		t.Fatal("expected error for nil map")
	}
	if !strings.Contains(err.Error(), "nil map") {
		t.Fatalf("expected 'nil map' in error, got %v", err)
	}
}

func TestExtract_NonMapIntermediate(t *testing.T) {
	data := map[string]any{
		"body": "not a map",
	}

	_, err := Extract(data, "body.field")
	if err == nil {
		t.Fatal("expected error for non-map intermediate")
	}
	if !strings.Contains(err.Error(), "non-map") {
		t.Fatalf("expected 'non-map' in error, got %v", err)
	}
}

func TestExtract_MissingNestedKey(t *testing.T) {
	data := map[string]any{
		"body": map[string]any{
			"user": map[string]any{},
		},
	}

	_, err := Extract(data, "body.user.missing")
	if err == nil {
		t.Fatal("expected error for missing nested key")
	}
}

func TestExtract_ReturnsMapValue(t *testing.T) {
	data := map[string]any{
		"body": map[string]any{
			"user": map[string]any{"name": "alice"},
		},
	}

	val, err := Extract(data, "body.user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := val.(map[string]any)
	if !ok {
		t.Fatal("expected map result")
	}
	if m["name"] != "alice" {
		t.Fatalf("expected name=alice, got %v", m["name"])
	}
}
