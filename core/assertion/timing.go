package assertion

import (
	"fmt"
	"time"
)

// toDuration converts various types to time.Duration.
func toDuration(v any) (time.Duration, error) {
	switch d := v.(type) {
	case time.Duration:
		return d, nil
	case int:
		return time.Duration(d), nil
	case int64:
		return time.Duration(d), nil
	case float64:
		return time.Duration(d), nil
	case string:
		return time.ParseDuration(d)
	default:
		return 0, fmt.Errorf("cannot convert %T to time.Duration", v)
	}
}

// ResponseTimeBelowAssertion checks that the actual elapsed time is below
// the specified maximum duration.
type ResponseTimeBelowAssertion struct {
	MaxDuration time.Duration
}

// Name returns the assertion name.
func (a *ResponseTimeBelowAssertion) Name() string { return "response_time_below" }

// Evaluate checks that the actual duration is below MaxDuration.
// The actual value must be a time.Duration.
func (a *ResponseTimeBelowAssertion) Evaluate(actual any) error {
	d, ok := actual.(time.Duration)
	if !ok {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.MaxDuration,
			Actual:    actual,
			Message:   fmt.Sprintf("expected time.Duration value, got %T", actual),
		}
	}
	if d < a.MaxDuration {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.MaxDuration,
		Actual:    d,
		Message:   fmt.Sprintf("response time %s is not below %s", d, a.MaxDuration),
	}
}
