package assertion

import (
	"fmt"
	"strings"
)

// headersFromActual extracts headers from actual, which must be a map[string]string
// or map[string][]string. Returns a normalized map with lowercase keys.
func headersFromActual(actual any) (map[string][]string, error) {
	switch h := actual.(type) {
	case map[string]string:
		result := make(map[string][]string, len(h))
		for k, v := range h {
			result[strings.ToLower(k)] = []string{v}
		}
		return result, nil
	case map[string][]string:
		result := make(map[string][]string, len(h))
		for k, v := range h {
			result[strings.ToLower(k)] = v
		}
		return result, nil
	default:
		return nil, fmt.Errorf("expected map[string]string or map[string][]string, got %T", actual)
	}
}

// HeaderEqualsAssertion checks that a header value equals the expected value.
// Header names are compared case-insensitively.
type HeaderEqualsAssertion struct {
	Header   string
	Expected any
}

// Name returns the assertion name.
func (a *HeaderEqualsAssertion) Name() string { return "header_equals" }

// Evaluate checks that the named header equals the expected value.
func (a *HeaderEqualsAssertion) Evaluate(actual any) error {
	headers, err := headersFromActual(actual)
	if err != nil {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    actual,
			Message:   err.Error(),
		}
	}

	key := strings.ToLower(a.Header)
	vals, exists := headers[key]
	if !exists || len(vals) == 0 {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    nil,
			Message:   fmt.Sprintf("header %q not found", a.Header),
		}
	}

	expectedStr := fmt.Sprintf("%v", a.Expected)
	for _, v := range vals {
		if v == expectedStr {
			return nil
		}
	}

	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Expected,
		Actual:    vals[0],
		Message:   fmt.Sprintf("header %q value %q does not equal %q", a.Header, vals[0], expectedStr),
	}
}

// HeaderContainsAssertion checks that a header value contains the given substring.
// Header names are compared case-insensitively.
type HeaderContainsAssertion struct {
	Header string
	Substr string
}

// Name returns the assertion name.
func (a *HeaderContainsAssertion) Name() string { return "header_contains" }

// Evaluate checks that the named header contains the substring.
func (a *HeaderContainsAssertion) Evaluate(actual any) error {
	headers, err := headersFromActual(actual)
	if err != nil {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Substr,
			Actual:    actual,
			Message:   err.Error(),
		}
	}

	key := strings.ToLower(a.Header)
	vals, exists := headers[key]
	if !exists || len(vals) == 0 {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Substr,
			Actual:    nil,
			Message:   fmt.Sprintf("header %q not found", a.Header),
		}
	}

	for _, v := range vals {
		if strings.Contains(v, a.Substr) {
			return nil
		}
	}

	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Substr,
		Actual:    vals[0],
		Message:   fmt.Sprintf("header %q value %q does not contain %q", a.Header, vals[0], a.Substr),
	}
}

// HeaderExistsAssertion checks that a header exists (case-insensitive name).
type HeaderExistsAssertion struct {
	Header string
}

// Name returns the assertion name.
func (a *HeaderExistsAssertion) Name() string { return "header_exists" }

// Evaluate checks that the named header exists.
func (a *HeaderExistsAssertion) Evaluate(actual any) error {
	headers, err := headersFromActual(actual)
	if err != nil {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Header,
			Actual:    actual,
			Message:   err.Error(),
		}
	}

	key := strings.ToLower(a.Header)
	if _, exists := headers[key]; exists {
		return nil
	}

	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Header,
		Actual:    nil,
		Message:   fmt.Sprintf("header %q does not exist", a.Header),
	}
}
