package assertion

import (
	"testing"
)

func TestJSONPathAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &JSONPathAssertion{Path: "x", Expected: 1}
		assertEqual(t, "json_path", a.Name())
	})

	t.Run("simple path", func(t *testing.T) {
		a := &JSONPathAssertion{Path: "name", Expected: "alice"}
		assertNoError(t, a.Evaluate(map[string]any{"name": "alice"}))
	})

	t.Run("nested path", func(t *testing.T) {
		a := &JSONPathAssertion{Path: "user.name", Expected: "alice"}
		data := map[string]any{
			"user": map[string]any{
				"name": "alice",
			},
		}
		assertNoError(t, a.Evaluate(data))
	})

	t.Run("deeply nested path", func(t *testing.T) {
		a := &JSONPathAssertion{Path: "body.user.name", Expected: "alice"}
		data := map[string]any{
			"body": map[string]any{
				"user": map[string]any{
					"name": "alice",
				},
			},
		}
		assertNoError(t, a.Evaluate(data))
	})

	t.Run("wrong value", func(t *testing.T) {
		a := &JSONPathAssertion{Path: "name", Expected: "alice"}
		ae := assertAssertionError(t, a.Evaluate(map[string]any{"name": "bob"}), "json_path")
		assertEqual(t, "alice", ae.Expected)
		assertEqual(t, "bob", ae.Actual)
		assertEqual(t, "name", ae.Path)
	})

	t.Run("missing key", func(t *testing.T) {
		a := &JSONPathAssertion{Path: "missing", Expected: "x"}
		ae := assertAssertionError(t, a.Evaluate(map[string]any{"name": "alice"}), "json_path")
		assertEqual(t, "missing", ae.Path)
	})

	t.Run("non-map actual", func(t *testing.T) {
		a := &JSONPathAssertion{Path: "name", Expected: "x"}
		ae := assertAssertionError(t, a.Evaluate("not a map"), "json_path")
		assertEqual(t, "name", ae.Path)
	})

	t.Run("nil actual", func(t *testing.T) {
		a := &JSONPathAssertion{Path: "name", Expected: "x"}
		assertError(t, a.Evaluate(nil))
	})

	t.Run("intermediate not a map", func(t *testing.T) {
		a := &JSONPathAssertion{Path: "user.name", Expected: "x"}
		data := map[string]any{"user": "not a map"}
		assertError(t, a.Evaluate(data))
	})

	t.Run("numeric cross-type comparison", func(t *testing.T) {
		a := &JSONPathAssertion{Path: "count", Expected: 42}
		assertNoError(t, a.Evaluate(map[string]any{"count": 42.0}))
	})

	t.Run("nil value at path", func(t *testing.T) {
		a := &JSONPathAssertion{Path: "value", Expected: nil}
		assertNoError(t, a.Evaluate(map[string]any{"value": nil}))
	})

	t.Run("nil value expected non-nil", func(t *testing.T) {
		a := &JSONPathAssertion{Path: "value", Expected: "something"}
		assertError(t, a.Evaluate(map[string]any{"value": nil}))
	})
}

func TestNotEmptyAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &NotEmptyAssertion{}
		assertEqual(t, "not_empty", a.Name())
	})

	t.Run("non-empty string", func(t *testing.T) {
		a := &NotEmptyAssertion{}
		assertNoError(t, a.Evaluate("hello"))
	})

	t.Run("empty string", func(t *testing.T) {
		a := &NotEmptyAssertion{}
		ae := assertAssertionError(t, a.Evaluate(""), "not_empty")
		if ae.Message == "" {
			t.Error("expected non-empty message")
		}
	})

	t.Run("nil", func(t *testing.T) {
		a := &NotEmptyAssertion{}
		assertError(t, a.Evaluate(nil))
	})

	t.Run("non-empty slice", func(t *testing.T) {
		a := &NotEmptyAssertion{}
		assertNoError(t, a.Evaluate([]int{1}))
	})

	t.Run("empty slice", func(t *testing.T) {
		a := &NotEmptyAssertion{}
		assertError(t, a.Evaluate([]int{}))
	})

	t.Run("non-empty map", func(t *testing.T) {
		a := &NotEmptyAssertion{}
		assertNoError(t, a.Evaluate(map[string]int{"a": 1}))
	})

	t.Run("empty map", func(t *testing.T) {
		a := &NotEmptyAssertion{}
		assertError(t, a.Evaluate(map[string]int{}))
	})

	t.Run("empty array", func(t *testing.T) {
		a := &NotEmptyAssertion{}
		assertError(t, a.Evaluate([0]int{}))
	})

	t.Run("non-empty array", func(t *testing.T) {
		a := &NotEmptyAssertion{}
		assertNoError(t, a.Evaluate([1]int{42}))
	})

	t.Run("integer passes (not a container)", func(t *testing.T) {
		a := &NotEmptyAssertion{}
		assertNoError(t, a.Evaluate(42))
	})

	t.Run("zero integer passes", func(t *testing.T) {
		a := &NotEmptyAssertion{}
		assertNoError(t, a.Evaluate(0))
	})
}
