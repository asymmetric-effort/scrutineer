package assertion

import (
	"testing"
	"time"
)

func TestToDuration(t *testing.T) {
	t.Run("time.Duration", func(t *testing.T) {
		d, err := toDuration(500 * time.Millisecond)
		assertNoError(t, err)
		assertEqual(t, 500*time.Millisecond, d)
	})

	t.Run("int", func(t *testing.T) {
		d, err := toDuration(int(1000))
		assertNoError(t, err)
		assertEqual(t, time.Duration(1000), d)
	})

	t.Run("int64", func(t *testing.T) {
		d, err := toDuration(int64(1000))
		assertNoError(t, err)
		assertEqual(t, time.Duration(1000), d)
	})

	t.Run("float64", func(t *testing.T) {
		d, err := toDuration(float64(1000))
		assertNoError(t, err)
		assertEqual(t, time.Duration(1000), d)
	})

	t.Run("string", func(t *testing.T) {
		d, err := toDuration("500ms")
		assertNoError(t, err)
		assertEqual(t, 500*time.Millisecond, d)
	})

	t.Run("invalid string", func(t *testing.T) {
		_, err := toDuration("not-a-duration")
		assertError(t, err)
	})

	t.Run("unsupported type", func(t *testing.T) {
		_, err := toDuration([]int{1})
		assertError(t, err)
	})
}

func TestResponseTimeBelowAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &ResponseTimeBelowAssertion{MaxDuration: time.Second}
		assertEqual(t, "response_time_below", a.Name())
	})

	t.Run("below threshold", func(t *testing.T) {
		a := &ResponseTimeBelowAssertion{MaxDuration: 500 * time.Millisecond}
		assertNoError(t, a.Evaluate(100*time.Millisecond))
	})

	t.Run("above threshold", func(t *testing.T) {
		a := &ResponseTimeBelowAssertion{MaxDuration: 500 * time.Millisecond}
		ae := assertAssertionError(t, a.Evaluate(600*time.Millisecond), "response_time_below")
		assertEqual(t, 500*time.Millisecond, ae.Expected)
		assertEqual(t, 600*time.Millisecond, ae.Actual)
	})

	t.Run("equal to threshold fails", func(t *testing.T) {
		a := &ResponseTimeBelowAssertion{MaxDuration: 500 * time.Millisecond}
		assertError(t, a.Evaluate(500*time.Millisecond))
	})

	t.Run("zero duration", func(t *testing.T) {
		a := &ResponseTimeBelowAssertion{MaxDuration: time.Second}
		assertNoError(t, a.Evaluate(time.Duration(0)))
	})

	t.Run("non-duration actual", func(t *testing.T) {
		a := &ResponseTimeBelowAssertion{MaxDuration: time.Second}
		assertError(t, a.Evaluate(42))
	})

	t.Run("nil actual", func(t *testing.T) {
		a := &ResponseTimeBelowAssertion{MaxDuration: time.Second}
		assertError(t, a.Evaluate(nil))
	})

	t.Run("string actual", func(t *testing.T) {
		a := &ResponseTimeBelowAssertion{MaxDuration: time.Second}
		assertError(t, a.Evaluate("500ms"))
	})
}
