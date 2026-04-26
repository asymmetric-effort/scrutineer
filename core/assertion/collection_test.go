package assertion

import (
	"testing"
)

func TestLengthAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &LengthAssertion{Expected: 3}
		assertEqual(t, "length", a.Name())
	})

	t.Run("matching slice length", func(t *testing.T) {
		a := &LengthAssertion{Expected: 3}
		assertNoError(t, a.Evaluate([]int{1, 2, 3}))
	})

	t.Run("non-matching slice length", func(t *testing.T) {
		a := &LengthAssertion{Expected: 3}
		ae := assertAssertionError(t, a.Evaluate([]int{1, 2}), "length")
		assertEqual(t, 3, ae.Expected)
		assertEqual(t, 2, ae.Actual)
	})

	t.Run("string length", func(t *testing.T) {
		a := &LengthAssertion{Expected: 5}
		assertNoError(t, a.Evaluate("hello"))
	})

	t.Run("map length", func(t *testing.T) {
		a := &LengthAssertion{Expected: 2}
		assertNoError(t, a.Evaluate(map[string]int{"a": 1, "b": 2}))
	})

	t.Run("array length", func(t *testing.T) {
		a := &LengthAssertion{Expected: 3}
		assertNoError(t, a.Evaluate([3]int{1, 2, 3}))
	})

	t.Run("nil with expected 0", func(t *testing.T) {
		a := &LengthAssertion{Expected: 0}
		assertNoError(t, a.Evaluate(nil))
	})

	t.Run("nil with expected non-zero", func(t *testing.T) {
		a := &LengthAssertion{Expected: 3}
		assertError(t, a.Evaluate(nil))
	})

	t.Run("non-measurable type", func(t *testing.T) {
		a := &LengthAssertion{Expected: 1}
		assertError(t, a.Evaluate(42))
	})

	t.Run("empty slice", func(t *testing.T) {
		a := &LengthAssertion{Expected: 0}
		assertNoError(t, a.Evaluate([]int{}))
	})
}

func TestEmptyAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &EmptyAssertion{}
		assertEqual(t, "empty", a.Name())
	})

	t.Run("empty slice", func(t *testing.T) {
		a := &EmptyAssertion{}
		assertNoError(t, a.Evaluate([]int{}))
	})

	t.Run("non-empty slice", func(t *testing.T) {
		a := &EmptyAssertion{}
		ae := assertAssertionError(t, a.Evaluate([]int{1}), "empty")
		assertEqual(t, "empty", ae.Expected)
	})

	t.Run("empty string", func(t *testing.T) {
		a := &EmptyAssertion{}
		assertNoError(t, a.Evaluate(""))
	})

	t.Run("non-empty string", func(t *testing.T) {
		a := &EmptyAssertion{}
		assertError(t, a.Evaluate("hello"))
	})

	t.Run("empty map", func(t *testing.T) {
		a := &EmptyAssertion{}
		assertNoError(t, a.Evaluate(map[string]int{}))
	})

	t.Run("non-empty map", func(t *testing.T) {
		a := &EmptyAssertion{}
		assertError(t, a.Evaluate(map[string]int{"a": 1}))
	})

	t.Run("nil", func(t *testing.T) {
		a := &EmptyAssertion{}
		assertNoError(t, a.Evaluate(nil))
	})

	t.Run("non-measurable type", func(t *testing.T) {
		a := &EmptyAssertion{}
		assertError(t, a.Evaluate(42))
	})

	t.Run("empty array", func(t *testing.T) {
		a := &EmptyAssertion{}
		assertNoError(t, a.Evaluate([0]int{}))
	})
}

func TestCollectionNotEmptyAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &CollectionNotEmptyAssertion{}
		assertEqual(t, "collection_not_empty", a.Name())
	})

	t.Run("non-empty slice", func(t *testing.T) {
		a := &CollectionNotEmptyAssertion{}
		assertNoError(t, a.Evaluate([]int{1}))
	})

	t.Run("empty slice", func(t *testing.T) {
		a := &CollectionNotEmptyAssertion{}
		assertError(t, a.Evaluate([]int{}))
	})

	t.Run("non-empty string", func(t *testing.T) {
		a := &CollectionNotEmptyAssertion{}
		assertNoError(t, a.Evaluate("hello"))
	})

	t.Run("empty string", func(t *testing.T) {
		a := &CollectionNotEmptyAssertion{}
		assertError(t, a.Evaluate(""))
	})

	t.Run("nil", func(t *testing.T) {
		a := &CollectionNotEmptyAssertion{}
		assertError(t, a.Evaluate(nil))
	})

	t.Run("non-measurable type", func(t *testing.T) {
		a := &CollectionNotEmptyAssertion{}
		assertError(t, a.Evaluate(42))
	})

	t.Run("non-empty map", func(t *testing.T) {
		a := &CollectionNotEmptyAssertion{}
		assertNoError(t, a.Evaluate(map[string]int{"a": 1}))
	})
}

func TestEachAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &EachAssertion{Inner: &GreaterThanAssertion{Expected: 0}}
		assertEqual(t, "each", a.Name())
	})

	t.Run("all pass", func(t *testing.T) {
		a := &EachAssertion{Inner: &GreaterThanAssertion{Expected: 0}}
		assertNoError(t, a.Evaluate([]int{1, 2, 3}))
	})

	t.Run("one fails", func(t *testing.T) {
		a := &EachAssertion{Inner: &GreaterThanAssertion{Expected: 0}}
		ae := assertAssertionError(t, a.Evaluate([]int{1, 0, 3}), "each")
		if ae.Message == "" {
			t.Error("expected non-empty message")
		}
	})

	t.Run("empty slice passes", func(t *testing.T) {
		a := &EachAssertion{Inner: &GreaterThanAssertion{Expected: 0}}
		assertNoError(t, a.Evaluate([]int{}))
	})

	t.Run("non-slice actual", func(t *testing.T) {
		a := &EachAssertion{Inner: &GreaterThanAssertion{Expected: 0}}
		assertError(t, a.Evaluate(42))
	})

	t.Run("nil actual", func(t *testing.T) {
		a := &EachAssertion{Inner: &GreaterThanAssertion{Expected: 0}}
		assertError(t, a.Evaluate(nil))
	})

	t.Run("any typed slice", func(t *testing.T) {
		a := &EachAssertion{Inner: &EqualAssertion{Expected: "hello"}}
		assertNoError(t, a.Evaluate([]any{"hello", "hello"}))
	})
}

func TestAnyAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &AnyAssertion{Inner: &EqualAssertion{Expected: 2}}
		assertEqual(t, "any", a.Name())
	})

	t.Run("one matches", func(t *testing.T) {
		a := &AnyAssertion{Inner: &EqualAssertion{Expected: 2}}
		assertNoError(t, a.Evaluate([]int{1, 2, 3}))
	})

	t.Run("none match", func(t *testing.T) {
		a := &AnyAssertion{Inner: &EqualAssertion{Expected: 99}}
		ae := assertAssertionError(t, a.Evaluate([]int{1, 2, 3}), "any")
		if ae.Message == "" {
			t.Error("expected non-empty message")
		}
	})

	t.Run("empty slice fails", func(t *testing.T) {
		a := &AnyAssertion{Inner: &EqualAssertion{Expected: 1}}
		assertError(t, a.Evaluate([]int{}))
	})

	t.Run("non-slice actual", func(t *testing.T) {
		a := &AnyAssertion{Inner: &EqualAssertion{Expected: 1}}
		assertError(t, a.Evaluate(42))
	})

	t.Run("nil actual", func(t *testing.T) {
		a := &AnyAssertion{Inner: &EqualAssertion{Expected: 1}}
		assertError(t, a.Evaluate(nil))
	})
}

func TestAllAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &AllAssertion{Inner: &GreaterThanAssertion{Expected: 0}}
		assertEqual(t, "all", a.Name())
	})

	t.Run("all pass", func(t *testing.T) {
		a := &AllAssertion{Inner: &GreaterThanAssertion{Expected: 0}}
		assertNoError(t, a.Evaluate([]int{1, 2, 3}))
	})

	t.Run("one fails", func(t *testing.T) {
		a := &AllAssertion{Inner: &GreaterThanAssertion{Expected: 0}}
		ae := assertAssertionError(t, a.Evaluate([]int{1, 0, 3}), "all")
		if ae.Message == "" {
			t.Error("expected non-empty message")
		}
	})

	t.Run("empty slice passes", func(t *testing.T) {
		a := &AllAssertion{Inner: &GreaterThanAssertion{Expected: 0}}
		assertNoError(t, a.Evaluate([]int{}))
	})

	t.Run("non-slice actual", func(t *testing.T) {
		a := &AllAssertion{Inner: &GreaterThanAssertion{Expected: 0}}
		assertError(t, a.Evaluate("not a slice"))
	})

	t.Run("nil actual", func(t *testing.T) {
		a := &AllAssertion{Inner: &GreaterThanAssertion{Expected: 0}}
		assertError(t, a.Evaluate(nil))
	})
}

func TestToSlice(t *testing.T) {
	t.Run("int slice", func(t *testing.T) {
		s, err := toSlice([]int{1, 2, 3})
		assertNoError(t, err)
		assertEqual(t, 3, len(s))
	})

	t.Run("string slice", func(t *testing.T) {
		s, err := toSlice([]string{"a", "b"})
		assertNoError(t, err)
		assertEqual(t, 2, len(s))
	})

	t.Run("array", func(t *testing.T) {
		s, err := toSlice([3]int{1, 2, 3})
		assertNoError(t, err)
		assertEqual(t, 3, len(s))
	})

	t.Run("any slice", func(t *testing.T) {
		s, err := toSlice([]any{1, "two", 3.0})
		assertNoError(t, err)
		assertEqual(t, 3, len(s))
	})

	t.Run("nil", func(t *testing.T) {
		_, err := toSlice(nil)
		assertError(t, err)
	})

	t.Run("not a slice", func(t *testing.T) {
		_, err := toSlice(42)
		assertError(t, err)
	})
}
