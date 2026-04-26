package assertion

import (
	"testing"
)

func TestContainsAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &ContainsAssertion{Substr: "x"}
		assertEqual(t, "contains", a.Name())
	})

	t.Run("contains substring", func(t *testing.T) {
		a := &ContainsAssertion{Substr: "world"}
		assertNoError(t, a.Evaluate("hello world"))
	})

	t.Run("does not contain", func(t *testing.T) {
		a := &ContainsAssertion{Substr: "xyz"}
		ae := assertAssertionError(t, a.Evaluate("hello"), "contains")
		assertEqual(t, "xyz", ae.Expected)
	})

	t.Run("empty substring always matches", func(t *testing.T) {
		a := &ContainsAssertion{Substr: ""}
		assertNoError(t, a.Evaluate("hello"))
	})

	t.Run("empty string contains empty", func(t *testing.T) {
		a := &ContainsAssertion{Substr: ""}
		assertNoError(t, a.Evaluate(""))
	})

	t.Run("non-string actual", func(t *testing.T) {
		a := &ContainsAssertion{Substr: "x"}
		ae := assertAssertionError(t, a.Evaluate(42), "contains")
		if ae.Message == "" {
			t.Error("expected non-empty message for type error")
		}
	})

	t.Run("nil actual", func(t *testing.T) {
		a := &ContainsAssertion{Substr: "x"}
		assertError(t, a.Evaluate(nil))
	})
}

func TestNotContainsAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &NotContainsAssertion{Substr: "x"}
		assertEqual(t, "not_contains", a.Name())
	})

	t.Run("does not contain", func(t *testing.T) {
		a := &NotContainsAssertion{Substr: "xyz"}
		assertNoError(t, a.Evaluate("hello"))
	})

	t.Run("contains fails", func(t *testing.T) {
		a := &NotContainsAssertion{Substr: "hello"}
		ae := assertAssertionError(t, a.Evaluate("hello world"), "not_contains")
		assertEqual(t, "hello", ae.Expected)
	})

	t.Run("non-string actual", func(t *testing.T) {
		a := &NotContainsAssertion{Substr: "x"}
		assertError(t, a.Evaluate(42))
	})

	t.Run("empty substring always matches so fails", func(t *testing.T) {
		a := &NotContainsAssertion{Substr: ""}
		assertError(t, a.Evaluate("hello"))
	})
}

func TestHasPrefixAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &HasPrefixAssertion{Prefix: "x"}
		assertEqual(t, "has_prefix", a.Name())
	})

	t.Run("has prefix", func(t *testing.T) {
		a := &HasPrefixAssertion{Prefix: "hello"}
		assertNoError(t, a.Evaluate("hello world"))
	})

	t.Run("no prefix", func(t *testing.T) {
		a := &HasPrefixAssertion{Prefix: "world"}
		ae := assertAssertionError(t, a.Evaluate("hello world"), "has_prefix")
		assertEqual(t, "world", ae.Expected)
	})

	t.Run("empty prefix", func(t *testing.T) {
		a := &HasPrefixAssertion{Prefix: ""}
		assertNoError(t, a.Evaluate("hello"))
	})

	t.Run("non-string actual", func(t *testing.T) {
		a := &HasPrefixAssertion{Prefix: "x"}
		assertError(t, a.Evaluate(42))
	})

	t.Run("exact match", func(t *testing.T) {
		a := &HasPrefixAssertion{Prefix: "hello"}
		assertNoError(t, a.Evaluate("hello"))
	})
}

func TestHasSuffixAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &HasSuffixAssertion{Suffix: "x"}
		assertEqual(t, "has_suffix", a.Name())
	})

	t.Run("has suffix", func(t *testing.T) {
		a := &HasSuffixAssertion{Suffix: "world"}
		assertNoError(t, a.Evaluate("hello world"))
	})

	t.Run("no suffix", func(t *testing.T) {
		a := &HasSuffixAssertion{Suffix: "hello"}
		ae := assertAssertionError(t, a.Evaluate("hello world"), "has_suffix")
		assertEqual(t, "hello", ae.Expected)
	})

	t.Run("empty suffix", func(t *testing.T) {
		a := &HasSuffixAssertion{Suffix: ""}
		assertNoError(t, a.Evaluate("hello"))
	})

	t.Run("non-string actual", func(t *testing.T) {
		a := &HasSuffixAssertion{Suffix: "x"}
		assertError(t, a.Evaluate(42))
	})
}
