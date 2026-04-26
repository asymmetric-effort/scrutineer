package assertion

import (
	"fmt"
	"time"
)

// toFloat64 converts numeric types to float64 for comparison.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	case time.Duration:
		return float64(n), true
	default:
		return 0, false
	}
}

// toInt converts numeric types to int.
func toInt(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case int8:
		return int(n), nil
	case int16:
		return int(n), nil
	case int32:
		return int(n), nil
	case int64:
		return int(n), nil
	case uint:
		return int(n), nil
	case uint8:
		return int(n), nil
	case uint16:
		return int(n), nil
	case uint32:
		return int(n), nil
	case uint64:
		return int(n), nil
	case float32:
		return int(n), nil
	case float64:
		return int(n), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}

// GreaterThanAssertion checks that actual > expected.
type GreaterThanAssertion struct {
	Expected any
}

// Name returns the assertion name.
func (a *GreaterThanAssertion) Name() string { return "greater_than" }

// Evaluate checks that actual is greater than expected.
func (a *GreaterThanAssertion) Evaluate(actual any) error {
	af, aOk := toFloat64(actual)
	ef, eOk := toFloat64(a.Expected)
	if !aOk {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    actual,
			Message:   fmt.Sprintf("actual value %v (%T) is not numeric", actual, actual),
		}
	}
	if !eOk {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    actual,
			Message:   fmt.Sprintf("expected value %v (%T) is not numeric", a.Expected, a.Expected),
		}
	}
	if af > ef {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Expected,
		Actual:    actual,
		Message:   fmt.Sprintf("expected %v to be greater than %v", actual, a.Expected),
	}
}

// LessThanAssertion checks that actual < expected.
type LessThanAssertion struct {
	Expected any
}

// Name returns the assertion name.
func (a *LessThanAssertion) Name() string { return "less_than" }

// Evaluate checks that actual is less than expected.
func (a *LessThanAssertion) Evaluate(actual any) error {
	af, aOk := toFloat64(actual)
	ef, eOk := toFloat64(a.Expected)
	if !aOk {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    actual,
			Message:   fmt.Sprintf("actual value %v (%T) is not numeric", actual, actual),
		}
	}
	if !eOk {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    actual,
			Message:   fmt.Sprintf("expected value %v (%T) is not numeric", a.Expected, a.Expected),
		}
	}
	if af < ef {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Expected,
		Actual:    actual,
		Message:   fmt.Sprintf("expected %v to be less than %v", actual, a.Expected),
	}
}

// GreaterOrEqualAssertion checks that actual >= expected.
type GreaterOrEqualAssertion struct {
	Expected any
}

// Name returns the assertion name.
func (a *GreaterOrEqualAssertion) Name() string { return "greater_or_equal" }

// Evaluate checks that actual is greater than or equal to expected.
func (a *GreaterOrEqualAssertion) Evaluate(actual any) error {
	af, aOk := toFloat64(actual)
	ef, eOk := toFloat64(a.Expected)
	if !aOk {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    actual,
			Message:   fmt.Sprintf("actual value %v (%T) is not numeric", actual, actual),
		}
	}
	if !eOk {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    actual,
			Message:   fmt.Sprintf("expected value %v (%T) is not numeric", a.Expected, a.Expected),
		}
	}
	if af >= ef {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Expected,
		Actual:    actual,
		Message:   fmt.Sprintf("expected %v to be greater than or equal to %v", actual, a.Expected),
	}
}

// LessOrEqualAssertion checks that actual <= expected.
type LessOrEqualAssertion struct {
	Expected any
}

// Name returns the assertion name.
func (a *LessOrEqualAssertion) Name() string { return "less_or_equal" }

// Evaluate checks that actual is less than or equal to expected.
func (a *LessOrEqualAssertion) Evaluate(actual any) error {
	af, aOk := toFloat64(actual)
	ef, eOk := toFloat64(a.Expected)
	if !aOk {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    actual,
			Message:   fmt.Sprintf("actual value %v (%T) is not numeric", actual, actual),
		}
	}
	if !eOk {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    actual,
			Message:   fmt.Sprintf("expected value %v (%T) is not numeric", a.Expected, a.Expected),
		}
	}
	if af <= ef {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Expected,
		Actual:    actual,
		Message:   fmt.Sprintf("expected %v to be less than or equal to %v", actual, a.Expected),
	}
}

// InRangeAssertion checks that actual is within [Min, Max] inclusive.
type InRangeAssertion struct {
	Min any
	Max any
}

// Name returns the assertion name.
func (a *InRangeAssertion) Name() string { return "in_range" }

// Evaluate checks that actual is within the range [Min, Max].
func (a *InRangeAssertion) Evaluate(actual any) error {
	af, aOk := toFloat64(actual)
	minF, minOk := toFloat64(a.Min)
	maxF, maxOk := toFloat64(a.Max)
	if !aOk {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  fmt.Sprintf("[%v, %v]", a.Min, a.Max),
			Actual:    actual,
			Message:   fmt.Sprintf("actual value %v (%T) is not numeric", actual, actual),
		}
	}
	if !minOk || !maxOk {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  fmt.Sprintf("[%v, %v]", a.Min, a.Max),
			Actual:    actual,
			Message:   "min or max value is not numeric",
		}
	}
	if af >= minF && af <= maxF {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  fmt.Sprintf("[%v, %v]", a.Min, a.Max),
		Actual:    actual,
		Message:   fmt.Sprintf("expected %v to be in range [%v, %v]", actual, a.Min, a.Max),
	}
}
