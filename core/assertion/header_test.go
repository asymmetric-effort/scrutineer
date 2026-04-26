package assertion

import (
	"testing"
)

func TestHeaderEqualsAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &HeaderEqualsAssertion{Header: "X", Expected: "v"}
		assertEqual(t, "header_equals", a.Name())
	})

	t.Run("matching header map[string]string", func(t *testing.T) {
		a := &HeaderEqualsAssertion{Header: "Content-Type", Expected: "text/html"}
		assertNoError(t, a.Evaluate(map[string]string{"Content-Type": "text/html"}))
	})

	t.Run("case insensitive header name", func(t *testing.T) {
		a := &HeaderEqualsAssertion{Header: "content-type", Expected: "text/html"}
		assertNoError(t, a.Evaluate(map[string]string{"Content-Type": "text/html"}))
	})

	t.Run("non-matching value", func(t *testing.T) {
		a := &HeaderEqualsAssertion{Header: "Content-Type", Expected: "application/json"}
		ae := assertAssertionError(t, a.Evaluate(map[string]string{"Content-Type": "text/html"}), "header_equals")
		assertEqual(t, "application/json", ae.Expected)
	})

	t.Run("missing header", func(t *testing.T) {
		a := &HeaderEqualsAssertion{Header: "X-Missing", Expected: "v"}
		assertError(t, a.Evaluate(map[string]string{"Content-Type": "text/html"}))
	})

	t.Run("map[string][]string", func(t *testing.T) {
		a := &HeaderEqualsAssertion{Header: "Content-Type", Expected: "text/html"}
		assertNoError(t, a.Evaluate(map[string][]string{"Content-Type": {"text/html"}}))
	})

	t.Run("map[string][]string multiple values match second", func(t *testing.T) {
		a := &HeaderEqualsAssertion{Header: "Accept", Expected: "text/html"}
		assertNoError(t, a.Evaluate(map[string][]string{"Accept": {"application/json", "text/html"}}))
	})

	t.Run("invalid type", func(t *testing.T) {
		a := &HeaderEqualsAssertion{Header: "X", Expected: "v"}
		assertError(t, a.Evaluate(42))
	})

	t.Run("nil actual", func(t *testing.T) {
		a := &HeaderEqualsAssertion{Header: "X", Expected: "v"}
		assertError(t, a.Evaluate(nil))
	})

	t.Run("empty headers", func(t *testing.T) {
		a := &HeaderEqualsAssertion{Header: "X", Expected: "v"}
		assertError(t, a.Evaluate(map[string]string{}))
	})
}

func TestHeaderContainsAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &HeaderContainsAssertion{Header: "X", Substr: "v"}
		assertEqual(t, "header_contains", a.Name())
	})

	t.Run("contains substring", func(t *testing.T) {
		a := &HeaderContainsAssertion{Header: "Content-Type", Substr: "text"}
		assertNoError(t, a.Evaluate(map[string]string{"Content-Type": "text/html"}))
	})

	t.Run("case insensitive header name", func(t *testing.T) {
		a := &HeaderContainsAssertion{Header: "content-type", Substr: "text"}
		assertNoError(t, a.Evaluate(map[string]string{"Content-Type": "text/html"}))
	})

	t.Run("not contains", func(t *testing.T) {
		a := &HeaderContainsAssertion{Header: "Content-Type", Substr: "json"}
		ae := assertAssertionError(t, a.Evaluate(map[string]string{"Content-Type": "text/html"}), "header_contains")
		assertEqual(t, "json", ae.Expected)
	})

	t.Run("missing header", func(t *testing.T) {
		a := &HeaderContainsAssertion{Header: "X-Missing", Substr: "v"}
		assertError(t, a.Evaluate(map[string]string{}))
	})

	t.Run("map[string][]string", func(t *testing.T) {
		a := &HeaderContainsAssertion{Header: "Content-Type", Substr: "text"}
		assertNoError(t, a.Evaluate(map[string][]string{"Content-Type": {"text/html"}}))
	})

	t.Run("invalid type", func(t *testing.T) {
		a := &HeaderContainsAssertion{Header: "X", Substr: "v"}
		assertError(t, a.Evaluate("not a map"))
	})
}

func TestHeaderExistsAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &HeaderExistsAssertion{Header: "X"}
		assertEqual(t, "header_exists", a.Name())
	})

	t.Run("exists", func(t *testing.T) {
		a := &HeaderExistsAssertion{Header: "Content-Type"}
		assertNoError(t, a.Evaluate(map[string]string{"Content-Type": "text/html"}))
	})

	t.Run("case insensitive", func(t *testing.T) {
		a := &HeaderExistsAssertion{Header: "content-type"}
		assertNoError(t, a.Evaluate(map[string]string{"Content-Type": "text/html"}))
	})

	t.Run("not exists", func(t *testing.T) {
		a := &HeaderExistsAssertion{Header: "X-Missing"}
		ae := assertAssertionError(t, a.Evaluate(map[string]string{"Content-Type": "text/html"}), "header_exists")
		assertEqual(t, "X-Missing", ae.Expected)
	})

	t.Run("invalid type", func(t *testing.T) {
		a := &HeaderExistsAssertion{Header: "X"}
		assertError(t, a.Evaluate(42))
	})

	t.Run("map[string][]string", func(t *testing.T) {
		a := &HeaderExistsAssertion{Header: "Content-Type"}
		assertNoError(t, a.Evaluate(map[string][]string{"Content-Type": {"text/html"}}))
	})

	t.Run("empty map", func(t *testing.T) {
		a := &HeaderExistsAssertion{Header: "X"}
		assertError(t, a.Evaluate(map[string]string{}))
	})
}
