package assertion

import (
	"testing"
)

func TestEqualAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &EqualAssertion{Expected: 1}
		assertEqual(t, "equal", a.Name())
	})

	t.Run("equal integers", func(t *testing.T) {
		a := &EqualAssertion{Expected: 42}
		assertNoError(t, a.Evaluate(42))
	})

	t.Run("equal strings", func(t *testing.T) {
		a := &EqualAssertion{Expected: "hello"}
		assertNoError(t, a.Evaluate("hello"))
	})

	t.Run("unequal integers", func(t *testing.T) {
		a := &EqualAssertion{Expected: 42}
		ae := assertAssertionError(t, a.Evaluate(99), "equal")
		assertEqual(t, 42, ae.Expected)
		assertEqual(t, 99, ae.Actual)
	})

	t.Run("nil equals nil", func(t *testing.T) {
		a := &EqualAssertion{Expected: nil}
		assertNoError(t, a.Evaluate(nil))
	})

	t.Run("nil not equal to value", func(t *testing.T) {
		a := &EqualAssertion{Expected: nil}
		assertError(t, a.Evaluate(42))
	})

	t.Run("value not equal to nil", func(t *testing.T) {
		a := &EqualAssertion{Expected: 42}
		assertError(t, a.Evaluate(nil))
	})

	t.Run("numeric cross-type int float", func(t *testing.T) {
		a := &EqualAssertion{Expected: 42}
		assertNoError(t, a.Evaluate(42.0))
	})

	t.Run("numeric cross-type float int", func(t *testing.T) {
		a := &EqualAssertion{Expected: 42.0}
		assertNoError(t, a.Evaluate(42))
	})

	t.Run("empty string", func(t *testing.T) {
		a := &EqualAssertion{Expected: ""}
		assertNoError(t, a.Evaluate(""))
	})

	t.Run("zero value", func(t *testing.T) {
		a := &EqualAssertion{Expected: 0}
		assertNoError(t, a.Evaluate(0))
	})

	t.Run("bool true", func(t *testing.T) {
		a := &EqualAssertion{Expected: true}
		assertNoError(t, a.Evaluate(true))
	})

	t.Run("bool false not true", func(t *testing.T) {
		a := &EqualAssertion{Expected: false}
		assertError(t, a.Evaluate(true))
	})
}

func TestNotEqualAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &NotEqualAssertion{Expected: 1}
		assertEqual(t, "not_equal", a.Name())
	})

	t.Run("different values pass", func(t *testing.T) {
		a := &NotEqualAssertion{Expected: 42}
		assertNoError(t, a.Evaluate(99))
	})

	t.Run("same values fail", func(t *testing.T) {
		a := &NotEqualAssertion{Expected: 42}
		ae := assertAssertionError(t, a.Evaluate(42), "not_equal")
		assertEqual(t, 42, ae.Expected)
		assertEqual(t, 42, ae.Actual)
	})

	t.Run("nil not equal to value", func(t *testing.T) {
		a := &NotEqualAssertion{Expected: nil}
		assertNoError(t, a.Evaluate(42))
	})

	t.Run("nil equal to nil fails", func(t *testing.T) {
		a := &NotEqualAssertion{Expected: nil}
		assertError(t, a.Evaluate(nil))
	})

	t.Run("numeric cross type equality fails", func(t *testing.T) {
		a := &NotEqualAssertion{Expected: 42}
		assertError(t, a.Evaluate(42.0))
	})

	t.Run("different types pass", func(t *testing.T) {
		a := &NotEqualAssertion{Expected: "42"}
		assertNoError(t, a.Evaluate(42))
	})
}

func TestDeepEqualAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &DeepEqualAssertion{Expected: nil}
		assertEqual(t, "deep_equal", a.Name())
	})

	t.Run("equal slices", func(t *testing.T) {
		a := &DeepEqualAssertion{Expected: []int{1, 2, 3}}
		assertNoError(t, a.Evaluate([]int{1, 2, 3}))
	})

	t.Run("unequal slices", func(t *testing.T) {
		a := &DeepEqualAssertion{Expected: []int{1, 2, 3}}
		ae := assertAssertionError(t, a.Evaluate([]int{1, 2}), "deep_equal")
		if ae.Message == "" {
			t.Error("expected non-empty message")
		}
	})

	t.Run("equal maps", func(t *testing.T) {
		a := &DeepEqualAssertion{Expected: map[string]int{"a": 1}}
		assertNoError(t, a.Evaluate(map[string]int{"a": 1}))
	})

	t.Run("unequal maps", func(t *testing.T) {
		a := &DeepEqualAssertion{Expected: map[string]int{"a": 1}}
		assertError(t, a.Evaluate(map[string]int{"a": 2}))
	})

	t.Run("nil deep equal", func(t *testing.T) {
		a := &DeepEqualAssertion{Expected: nil}
		assertNoError(t, a.Evaluate(nil))
	})

	t.Run("empty slice vs nil", func(t *testing.T) {
		a := &DeepEqualAssertion{Expected: []int{}}
		assertError(t, a.Evaluate(nil))
	})

	t.Run("nested structs", func(t *testing.T) {
		type inner struct {
			X int
		}
		type outer struct {
			I inner
		}
		a := &DeepEqualAssertion{Expected: outer{I: inner{X: 1}}}
		assertNoError(t, a.Evaluate(outer{I: inner{X: 1}}))
	})
}
