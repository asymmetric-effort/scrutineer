package assertion

import (
	"testing"
	"time"
)

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want float64
		ok   bool
	}{
		{"int", int(42), 42.0, true},
		{"int8", int8(42), 42.0, true},
		{"int16", int16(42), 42.0, true},
		{"int32", int32(42), 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"uint", uint(42), 42.0, true},
		{"uint8", uint8(42), 42.0, true},
		{"uint16", uint16(42), 42.0, true},
		{"uint32", uint32(42), 42.0, true},
		{"uint64", uint64(42), 42.0, true},
		{"float32", float32(42.5), 42.5, true},
		{"float64", float64(42.5), 42.5, true},
		{"duration", time.Second, float64(time.Second), true},
		{"string", "42", 0, false},
		{"nil", nil, 0, false},
		{"bool", true, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toFloat64(tt.val)
			if ok != tt.ok {
				t.Errorf("toFloat64(%v) ok = %v, want %v", tt.val, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("toFloat64(%v) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name    string
		val     any
		want    int
		wantErr bool
	}{
		{"int", int(42), 42, false},
		{"int8", int8(42), 42, false},
		{"int16", int16(42), 42, false},
		{"int32", int32(42), 42, false},
		{"int64", int64(42), 42, false},
		{"uint", uint(42), 42, false},
		{"uint8", uint8(42), 42, false},
		{"uint16", uint16(42), 42, false},
		{"uint32", uint32(42), 42, false},
		{"uint64", uint64(42), 42, false},
		{"float32", float32(42), 42, false},
		{"float64", float64(42), 42, false},
		{"string", "42", 0, true},
		{"nil", nil, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toInt(tt.val)
			if (err != nil) != tt.wantErr {
				t.Errorf("toInt(%v) error = %v, wantErr %v", tt.val, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("toInt(%v) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestGreaterThanAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &GreaterThanAssertion{Expected: 10}
		assertEqual(t, "greater_than", a.Name())
	})

	t.Run("greater", func(t *testing.T) {
		a := &GreaterThanAssertion{Expected: 10}
		assertNoError(t, a.Evaluate(20))
	})

	t.Run("equal fails", func(t *testing.T) {
		a := &GreaterThanAssertion{Expected: 10}
		assertError(t, a.Evaluate(10))
	})

	t.Run("less fails", func(t *testing.T) {
		a := &GreaterThanAssertion{Expected: 10}
		ae := assertAssertionError(t, a.Evaluate(5), "greater_than")
		assertEqual(t, 10, ae.Expected)
	})

	t.Run("float comparison", func(t *testing.T) {
		a := &GreaterThanAssertion{Expected: 10.5}
		assertNoError(t, a.Evaluate(10.6))
		assertError(t, a.Evaluate(10.4))
	})

	t.Run("cross type int float", func(t *testing.T) {
		a := &GreaterThanAssertion{Expected: 10}
		assertNoError(t, a.Evaluate(10.5))
	})

	t.Run("non-numeric actual", func(t *testing.T) {
		a := &GreaterThanAssertion{Expected: 10}
		assertError(t, a.Evaluate("not a number"))
	})

	t.Run("non-numeric expected", func(t *testing.T) {
		a := &GreaterThanAssertion{Expected: "not a number"}
		assertError(t, a.Evaluate(10))
	})

	t.Run("nil actual", func(t *testing.T) {
		a := &GreaterThanAssertion{Expected: 10}
		assertError(t, a.Evaluate(nil))
	})
}

func TestLessThanAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &LessThanAssertion{Expected: 10}
		assertEqual(t, "less_than", a.Name())
	})

	t.Run("less", func(t *testing.T) {
		a := &LessThanAssertion{Expected: 10}
		assertNoError(t, a.Evaluate(5))
	})

	t.Run("equal fails", func(t *testing.T) {
		a := &LessThanAssertion{Expected: 10}
		assertError(t, a.Evaluate(10))
	})

	t.Run("greater fails", func(t *testing.T) {
		a := &LessThanAssertion{Expected: 10}
		ae := assertAssertionError(t, a.Evaluate(20), "less_than")
		assertEqual(t, 10, ae.Expected)
	})

	t.Run("non-numeric actual", func(t *testing.T) {
		a := &LessThanAssertion{Expected: 10}
		assertError(t, a.Evaluate("x"))
	})

	t.Run("non-numeric expected", func(t *testing.T) {
		a := &LessThanAssertion{Expected: "x"}
		assertError(t, a.Evaluate(10))
	})
}

func TestGreaterOrEqualAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &GreaterOrEqualAssertion{Expected: 10}
		assertEqual(t, "greater_or_equal", a.Name())
	})

	t.Run("greater", func(t *testing.T) {
		a := &GreaterOrEqualAssertion{Expected: 10}
		assertNoError(t, a.Evaluate(20))
	})

	t.Run("equal", func(t *testing.T) {
		a := &GreaterOrEqualAssertion{Expected: 10}
		assertNoError(t, a.Evaluate(10))
	})

	t.Run("less fails", func(t *testing.T) {
		a := &GreaterOrEqualAssertion{Expected: 10}
		ae := assertAssertionError(t, a.Evaluate(5), "greater_or_equal")
		assertEqual(t, 10, ae.Expected)
	})

	t.Run("non-numeric actual", func(t *testing.T) {
		a := &GreaterOrEqualAssertion{Expected: 10}
		assertError(t, a.Evaluate("x"))
	})

	t.Run("non-numeric expected", func(t *testing.T) {
		a := &GreaterOrEqualAssertion{Expected: "x"}
		assertError(t, a.Evaluate(10))
	})
}

func TestLessOrEqualAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &LessOrEqualAssertion{Expected: 10}
		assertEqual(t, "less_or_equal", a.Name())
	})

	t.Run("less", func(t *testing.T) {
		a := &LessOrEqualAssertion{Expected: 10}
		assertNoError(t, a.Evaluate(5))
	})

	t.Run("equal", func(t *testing.T) {
		a := &LessOrEqualAssertion{Expected: 10}
		assertNoError(t, a.Evaluate(10))
	})

	t.Run("greater fails", func(t *testing.T) {
		a := &LessOrEqualAssertion{Expected: 10}
		ae := assertAssertionError(t, a.Evaluate(20), "less_or_equal")
		assertEqual(t, 10, ae.Expected)
	})

	t.Run("non-numeric actual", func(t *testing.T) {
		a := &LessOrEqualAssertion{Expected: 10}
		assertError(t, a.Evaluate("x"))
	})

	t.Run("non-numeric expected", func(t *testing.T) {
		a := &LessOrEqualAssertion{Expected: "x"}
		assertError(t, a.Evaluate(10))
	})
}

func TestInRangeAssertion(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		a := &InRangeAssertion{Min: 1, Max: 10}
		assertEqual(t, "in_range", a.Name())
	})

	t.Run("in range", func(t *testing.T) {
		a := &InRangeAssertion{Min: 1, Max: 10}
		assertNoError(t, a.Evaluate(5))
	})

	t.Run("at min boundary", func(t *testing.T) {
		a := &InRangeAssertion{Min: 1, Max: 10}
		assertNoError(t, a.Evaluate(1))
	})

	t.Run("at max boundary", func(t *testing.T) {
		a := &InRangeAssertion{Min: 1, Max: 10}
		assertNoError(t, a.Evaluate(10))
	})

	t.Run("below range", func(t *testing.T) {
		a := &InRangeAssertion{Min: 1, Max: 10}
		assertError(t, a.Evaluate(0))
	})

	t.Run("above range", func(t *testing.T) {
		a := &InRangeAssertion{Min: 1, Max: 10}
		ae := assertAssertionError(t, a.Evaluate(11), "in_range")
		if ae.Message == "" {
			t.Error("expected non-empty message")
		}
	})

	t.Run("non-numeric actual", func(t *testing.T) {
		a := &InRangeAssertion{Min: 1, Max: 10}
		assertError(t, a.Evaluate("x"))
	})

	t.Run("non-numeric min", func(t *testing.T) {
		a := &InRangeAssertion{Min: "x", Max: 10}
		assertError(t, a.Evaluate(5))
	})

	t.Run("float range", func(t *testing.T) {
		a := &InRangeAssertion{Min: 1.5, Max: 3.5}
		assertNoError(t, a.Evaluate(2.5))
		assertError(t, a.Evaluate(4.0))
	})
}
