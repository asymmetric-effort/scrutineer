package assertion

import "fmt"

// StatusCodeAssertion checks that an HTTP status code matches the expected value.
type StatusCodeAssertion struct {
	Expected int
}

// Name returns the assertion name.
func (a *StatusCodeAssertion) Name() string { return "status_code" }

// Evaluate checks that the actual value matches the expected status code.
func (a *StatusCodeAssertion) Evaluate(actual any) error {
	code, err := toInt(actual)
	if err != nil {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    actual,
			Message:   fmt.Sprintf("expected integer status code, got %T", actual),
		}
	}
	if code == a.Expected {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Expected,
		Actual:    code,
		Message:   fmt.Sprintf("expected status code %d, got %d", a.Expected, code),
	}
}

// StatusClassAssertion checks that an HTTP status code belongs to a class (2xx, 3xx, 4xx, 5xx).
type StatusClassAssertion struct {
	Class    string
	minCode  int
	maxCode  int
}

// NewStatusClassAssertion creates a StatusClassAssertion for the given class string.
func NewStatusClassAssertion(class string) (*StatusClassAssertion, error) {
	var min, max int
	switch class {
	case "1xx":
		min, max = 100, 199
	case "2xx":
		min, max = 200, 299
	case "3xx":
		min, max = 300, 399
	case "4xx":
		min, max = 400, 499
	case "5xx":
		min, max = 500, 599
	default:
		return nil, fmt.Errorf("unknown status class %q: expected 1xx, 2xx, 3xx, 4xx, or 5xx", class)
	}
	return &StatusClassAssertion{Class: class, minCode: min, maxCode: max}, nil
}

// Name returns the assertion name.
func (a *StatusClassAssertion) Name() string { return "status_class" }

// Evaluate checks that the actual status code belongs to the expected class.
func (a *StatusClassAssertion) Evaluate(actual any) error {
	code, err := toInt(actual)
	if err != nil {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Class,
			Actual:    actual,
			Message:   fmt.Sprintf("expected integer status code, got %T", actual),
		}
	}
	if code >= a.minCode && code <= a.maxCode {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Class,
		Actual:    code,
		Message:   fmt.Sprintf("expected status code in class %s, got %d", a.Class, code),
	}
}
