package assertion

import (
	"testing"
)

func TestMatchesAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a, _ := NewMatchesAssertion(`\d+`)
		assertEqual(t, "matches", a.Name())
	})

	t.Run("matches pattern", func(t *testing.T) {
		a, err := NewMatchesAssertion(`^\d+$`)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate("12345"))
	})

	t.Run("does not match", func(t *testing.T) {
		a, _ := NewMatchesAssertion(`^\d+$`)
		ae := assertAssertionError(t, a.Evaluate("abc"), "matches")
		assertEqual(t, `^\d+$`, ae.Expected)
	})

	t.Run("invalid regex", func(t *testing.T) {
		_, err := NewMatchesAssertion("[invalid")
		assertError(t, err)
	})

	t.Run("non-string actual", func(t *testing.T) {
		a, _ := NewMatchesAssertion(`\d+`)
		assertError(t, a.Evaluate(42))
	})

	t.Run("nil actual", func(t *testing.T) {
		a, _ := NewMatchesAssertion(`\d+`)
		assertError(t, a.Evaluate(nil))
	})

	t.Run("empty string against empty pattern", func(t *testing.T) {
		a, err := NewMatchesAssertion(`^$`)
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(""))
	})

	t.Run("partial match", func(t *testing.T) {
		a, _ := NewMatchesAssertion(`\d+`)
		assertNoError(t, a.Evaluate("abc123def"))
	})

	t.Run("complex pattern", func(t *testing.T) {
		a, _ := NewMatchesAssertion(`^[a-z]+@[a-z]+\.[a-z]{2,}$`)
		assertNoError(t, a.Evaluate("test@example.com"))
		assertError(t, a.Evaluate("not-an-email"))
	})
}
