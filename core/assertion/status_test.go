package assertion

import (
	"testing"
)

func TestStatusCodeAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &StatusCodeAssertion{Expected: 200}
		assertEqual(t, "status_code", a.Name())
	})

	t.Run("matching code", func(t *testing.T) {
		a := &StatusCodeAssertion{Expected: 200}
		assertNoError(t, a.Evaluate(200))
	})

	t.Run("non-matching code", func(t *testing.T) {
		a := &StatusCodeAssertion{Expected: 200}
		ae := assertAssertionError(t, a.Evaluate(404), "status_code")
		assertEqual(t, 200, ae.Expected)
		assertEqual(t, 404, ae.Actual)
	})

	t.Run("non-integer actual", func(t *testing.T) {
		a := &StatusCodeAssertion{Expected: 200}
		assertError(t, a.Evaluate("200"))
	})

	t.Run("nil actual", func(t *testing.T) {
		a := &StatusCodeAssertion{Expected: 200}
		assertError(t, a.Evaluate(nil))
	})

	t.Run("int64 actual", func(t *testing.T) {
		a := &StatusCodeAssertion{Expected: 200}
		assertNoError(t, a.Evaluate(int64(200)))
	})

	t.Run("float64 actual", func(t *testing.T) {
		a := &StatusCodeAssertion{Expected: 200}
		assertNoError(t, a.Evaluate(float64(200)))
	})
}

func TestStatusClassAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a, _ := NewStatusClassAssertion("2xx")
		assertEqual(t, "status_class", a.Name())
	})

	t.Run("1xx", func(t *testing.T) {
		a, err := NewStatusClassAssertion("1xx")
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(100))
		assertNoError(t, a.Evaluate(199))
		assertError(t, a.Evaluate(200))
	})

	t.Run("2xx", func(t *testing.T) {
		a, err := NewStatusClassAssertion("2xx")
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(200))
		assertNoError(t, a.Evaluate(201))
		assertNoError(t, a.Evaluate(299))
		assertError(t, a.Evaluate(300))
		assertError(t, a.Evaluate(199))
	})

	t.Run("3xx", func(t *testing.T) {
		a, err := NewStatusClassAssertion("3xx")
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(301))
		assertError(t, a.Evaluate(200))
	})

	t.Run("4xx", func(t *testing.T) {
		a, err := NewStatusClassAssertion("4xx")
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(404))
		assertError(t, a.Evaluate(500))
	})

	t.Run("5xx", func(t *testing.T) {
		a, err := NewStatusClassAssertion("5xx")
		assertNoError(t, err)
		assertNoError(t, a.Evaluate(500))
		assertNoError(t, a.Evaluate(503))
		assertError(t, a.Evaluate(404))
	})

	t.Run("invalid class", func(t *testing.T) {
		_, err := NewStatusClassAssertion("6xx")
		assertError(t, err)
	})

	t.Run("non-integer actual", func(t *testing.T) {
		a, _ := NewStatusClassAssertion("2xx")
		assertError(t, a.Evaluate("200"))
	})

	t.Run("nil actual", func(t *testing.T) {
		a, _ := NewStatusClassAssertion("2xx")
		assertError(t, a.Evaluate(nil))
	})

	t.Run("error fields", func(t *testing.T) {
		a, _ := NewStatusClassAssertion("2xx")
		ae := assertAssertionError(t, a.Evaluate(404), "status_class")
		assertEqual(t, "2xx", ae.Expected)
		assertEqual(t, 404, ae.Actual)
	})
}
