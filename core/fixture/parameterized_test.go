package fixture

import (
	"testing"
)

func TestExpand_SingleParamSet(t *testing.T) {
	steps := []map[string]any{
		{"action": "get", "url": "/users"},
	}
	params := []ParameterSet{
		{Name: "admin", Values: map[string]any{"role": "admin"}},
	}

	result := Expand("GetUser", steps, params)

	if len(result) != 1 {
		t.Fatalf("expected 1 expanded test, got %d", len(result))
	}
	if result[0].Name != "GetUser [admin]" {
		t.Fatalf("expected 'GetUser [admin]', got %q", result[0].Name)
	}
	if result[0].Params["role"] != "admin" {
		t.Fatalf("expected role=admin, got %v", result[0].Params["role"])
	}
	if len(result[0].Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(result[0].Steps))
	}
}

func TestExpand_MultipleParamSets(t *testing.T) {
	steps := []map[string]any{
		{"action": "login"},
		{"action": "verify"},
	}
	params := []ParameterSet{
		{Name: "admin", Values: map[string]any{"role": "admin"}},
		{Name: "viewer", Values: map[string]any{"role": "viewer"}},
		{Name: "editor", Values: map[string]any{"role": "editor"}},
	}

	result := Expand("LoginTest", steps, params)

	if len(result) != 3 {
		t.Fatalf("expected 3 expanded tests, got %d", len(result))
	}

	expectedNames := []string{
		"LoginTest [admin]",
		"LoginTest [viewer]",
		"LoginTest [editor]",
	}
	for i, name := range expectedNames {
		if result[i].Name != name {
			t.Fatalf("expected name %q, got %q", name, result[i].Name)
		}
	}

	// Each expanded test should have 2 steps.
	for i, et := range result {
		if len(et.Steps) != 2 {
			t.Fatalf("expanded test %d: expected 2 steps, got %d", i, len(et.Steps))
		}
	}
}

func TestExpand_EmptyParams(t *testing.T) {
	steps := []map[string]any{
		{"action": "test"},
	}

	result := Expand("Test", steps, nil)
	if result != nil {
		t.Fatalf("expected nil for empty params, got %v", result)
	}

	result = Expand("Test", steps, []ParameterSet{})
	if result != nil {
		t.Fatalf("expected nil for empty slice, got %v", result)
	}
}

func TestExpand_StepsAreDeepCopied(t *testing.T) {
	steps := []map[string]any{
		{"action": "test", "nested": map[string]any{"key": "value"}},
	}
	params := []ParameterSet{
		{Name: "a", Values: map[string]any{"x": 1}},
		{Name: "b", Values: map[string]any{"x": 2}},
	}

	result := Expand("Test", steps, params)

	// Modify steps in the first expanded test.
	result[0].Steps[0]["action"] = "modified"
	nested := result[0].Steps[0]["nested"].(map[string]any)
	nested["key"] = "changed"

	// Original and second expanded test should be unaffected.
	if steps[0]["action"] != "test" {
		t.Fatal("original steps were modified")
	}
	if result[1].Steps[0]["action"] != "test" {
		t.Fatal("second expanded test was affected by modification of first")
	}
	nested2 := result[1].Steps[0]["nested"].(map[string]any)
	if nested2["key"] != "value" {
		t.Fatal("nested value in second test was affected")
	}
}

func TestExpand_NilSteps(t *testing.T) {
	params := []ParameterSet{
		{Name: "a", Values: map[string]any{"x": 1}},
	}

	result := Expand("Test", nil, params)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Steps != nil {
		t.Fatal("expected nil steps to remain nil")
	}
}

func TestExpand_VerifyNames(t *testing.T) {
	steps := []map[string]any{{"action": "x"}}
	params := []ParameterSet{
		{Name: "case-1", Values: map[string]any{}},
		{Name: "case-2", Values: map[string]any{}},
	}

	result := Expand("MyTest", steps, params)

	if result[0].Name != "MyTest [case-1]" {
		t.Fatalf("unexpected name: %q", result[0].Name)
	}
	if result[1].Name != "MyTest [case-2]" {
		t.Fatalf("unexpected name: %q", result[1].Name)
	}
}

func TestDeepCopySteps_WithSliceValues(t *testing.T) {
	steps := []map[string]any{
		{"tags": []any{"a", "b"}},
	}

	copied := deepCopySteps(steps)
	// Modify original slice.
	original := steps[0]["tags"].([]any)
	original[0] = "modified"

	copiedTags := copied[0]["tags"].([]any)
	if copiedTags[0] != "a" {
		t.Fatal("deep copy did not isolate slice values")
	}
}

func TestDeepCopyMap_Nil(t *testing.T) {
	result := deepCopyMap(nil)
	if result != nil {
		t.Fatal("expected nil for nil input")
	}
}
