package assertion

import (
	"fmt"
	"reflect"
	"strings"
)

// JSONPathAssertion extracts a value from a nested map[string]any using
// dot-notation paths (e.g. "body.user.name") and compares it to an expected value.
type JSONPathAssertion struct {
	Path     string
	Expected any
}

// Name returns the assertion name.
func (a *JSONPathAssertion) Name() string { return "json_path" }

// Evaluate extracts the value at the dot-notation path and compares it to expected.
func (a *JSONPathAssertion) Evaluate(actual any) error {
	m, ok := actual.(map[string]any)
	if !ok {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    actual,
			Path:      a.Path,
			Message:   fmt.Sprintf("expected map[string]any value, got %T", actual),
		}
	}

	val, err := extractPath(m, a.Path)
	if err != nil {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Expected,
			Actual:    actual,
			Path:      a.Path,
			Message:   err.Error(),
		}
	}

	if reflect.DeepEqual(val, a.Expected) {
		return nil
	}
	// Try numeric comparison
	if numericEqual(val, a.Expected) {
		return nil
	}

	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Expected,
		Actual:    val,
		Path:      a.Path,
		Message:   fmt.Sprintf("value at path %q is %v, expected %v", a.Path, val, a.Expected),
	}
}

// extractPath walks a nested map using dot-notation path segments.
func extractPath(m map[string]any, path string) (any, error) {
	parts := strings.Split(path, ".")
	var current any = m
	for i, part := range parts {
		cm, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("value at %q is not a map (got %T)", strings.Join(parts[:i], "."), current)
		}
		val, exists := cm[part]
		if !exists {
			return nil, fmt.Errorf("key %q not found at path %q", part, strings.Join(parts[:i+1], "."))
		}
		current = val
	}
	return current, nil
}

// NotEmptyAssertion checks that a value is not empty. It handles strings,
// slices, maps, and nil values.
type NotEmptyAssertion struct{}

// Name returns the assertion name.
func (a *NotEmptyAssertion) Name() string { return "not_empty" }

// Evaluate checks that the actual value is not empty.
func (a *NotEmptyAssertion) Evaluate(actual any) error {
	if actual == nil {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  "non-empty value",
			Actual:    actual,
			Message:   "expected non-empty value, got nil",
		}
	}

	v := reflect.ValueOf(actual)
	switch v.Kind() {
	case reflect.String:
		if v.Len() == 0 {
			return &AssertionError{
				Assertion: a.Name(),
				Expected:  "non-empty value",
				Actual:    actual,
				Message:   "expected non-empty string",
			}
		}
	case reflect.Slice, reflect.Array, reflect.Map:
		if v.Len() == 0 {
			return &AssertionError{
				Assertion: a.Name(),
				Expected:  "non-empty value",
				Actual:    actual,
				Message:   fmt.Sprintf("expected non-empty %s", v.Kind()),
			}
		}
	}

	return nil
}
