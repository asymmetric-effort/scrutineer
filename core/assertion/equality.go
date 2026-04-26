package assertion

import (
	"fmt"
	"reflect"
)

// EqualAssertion checks that actual == expected using == comparison.
type EqualAssertion struct {
	Expected any
}

// Name returns the assertion name.
func (a *EqualAssertion) Name() string { return "equal" }

// Evaluate checks that actual equals expected.
func (a *EqualAssertion) Evaluate(actual any) error {
	if actual == a.Expected {
		return nil
	}
	// Try numeric comparison for mixed int/float
	if numericEqual(actual, a.Expected) {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Expected,
		Actual:    actual,
		Message:   fmt.Sprintf("expected %v to equal %v", actual, a.Expected),
	}
}

// NotEqualAssertion checks that actual != expected.
type NotEqualAssertion struct {
	Expected any
}

// Name returns the assertion name.
func (a *NotEqualAssertion) Name() string { return "not_equal" }

// Evaluate checks that actual does not equal expected.
func (a *NotEqualAssertion) Evaluate(actual any) error {
	if actual != a.Expected && !numericEqual(actual, a.Expected) {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Expected,
		Actual:    actual,
		Message:   fmt.Sprintf("expected %v to not equal %v", actual, a.Expected),
	}
}

// DeepEqualAssertion checks that actual and expected are deeply equal
// using reflect.DeepEqual.
type DeepEqualAssertion struct {
	Expected any
}

// Name returns the assertion name.
func (a *DeepEqualAssertion) Name() string { return "deep_equal" }

// Evaluate checks deep equality between actual and expected.
func (a *DeepEqualAssertion) Evaluate(actual any) error {
	if reflect.DeepEqual(actual, a.Expected) {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Expected,
		Actual:    actual,
		Message:   fmt.Sprintf("expected %v to deeply equal %v", actual, a.Expected),
	}
}

// numericEqual compares two values as float64 if both are numeric.
func numericEqual(a, b any) bool {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if aOk && bOk {
		return af == bf
	}
	return false
}
