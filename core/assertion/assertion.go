// Package assertion provides a comprehensive assertion library for the
// scrutineer test framework. Assertions evaluate actual values against
// expected conditions and produce clear, human-friendly error messages.
package assertion

import "fmt"

// Assertion defines the contract for all assertion implementations.
// Each assertion has a name and can evaluate an actual value, returning
// an error if the assertion fails.
type Assertion interface {
	// Name returns a human-readable name for this assertion.
	Name() string
	// Evaluate checks the actual value against the assertion's condition.
	// Returns nil if the assertion passes, or an error (typically *AssertionError) if it fails.
	Evaluate(actual any) error
}

// AssertionError describes a failed assertion with structured fields
// for programmatic inspection and a human-friendly error message.
type AssertionError struct {
	// Assertion is the name of the assertion that failed.
	Assertion string
	// Expected is the value the assertion expected.
	Expected any
	// Actual is the value that was provided.
	Actual any
	// Message is a human-friendly description of the failure.
	Message string
	// Path is an optional dot-notation path for nested value assertions.
	Path string
}

// Error implements the error interface with a clear human-friendly message.
func (e *AssertionError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("assertion %q failed at path %q: %s (expected: %v, actual: %v)",
			e.Assertion, e.Path, e.Message, e.Expected, e.Actual)
	}
	return fmt.Sprintf("assertion %q failed: %s (expected: %v, actual: %v)",
		e.Assertion, e.Message, e.Expected, e.Actual)
}

// Builder creates Assertion instances from operator strings and expected values.
type Builder interface {
	// Build creates an Assertion from an operator string, an expected value,
	// and optional configuration options.
	Build(operator string, expected any, options map[string]any) (Assertion, error)
}

// DefaultBuilder implements Builder and supports all built-in operator strings.
type DefaultBuilder struct{}

// Build creates an assertion based on the operator string.
func (b *DefaultBuilder) Build(operator string, expected any, options map[string]any) (Assertion, error) {
	switch operator {
	case "equal", "eq":
		return &EqualAssertion{Expected: expected}, nil
	case "not_equal", "neq":
		return &NotEqualAssertion{Expected: expected}, nil
	case "deep_equal":
		return &DeepEqualAssertion{Expected: expected}, nil
	case "contains":
		s, ok := expected.(string)
		if !ok {
			return nil, fmt.Errorf("contains operator requires string expected value, got %T", expected)
		}
		return &ContainsAssertion{Substr: s}, nil
	case "not_contains":
		s, ok := expected.(string)
		if !ok {
			return nil, fmt.Errorf("not_contains operator requires string expected value, got %T", expected)
		}
		return &NotContainsAssertion{Substr: s}, nil
	case "has_prefix":
		s, ok := expected.(string)
		if !ok {
			return nil, fmt.Errorf("has_prefix operator requires string expected value, got %T", expected)
		}
		return &HasPrefixAssertion{Prefix: s}, nil
	case "has_suffix":
		s, ok := expected.(string)
		if !ok {
			return nil, fmt.Errorf("has_suffix operator requires string expected value, got %T", expected)
		}
		return &HasSuffixAssertion{Suffix: s}, nil
	case "matches":
		s, ok := expected.(string)
		if !ok {
			return nil, fmt.Errorf("matches operator requires string expected value, got %T", expected)
		}
		return NewMatchesAssertion(s)
	case "greater_than", "gt":
		return &GreaterThanAssertion{Expected: expected}, nil
	case "less_than", "lt":
		return &LessThanAssertion{Expected: expected}, nil
	case "greater_or_equal", "gte":
		return &GreaterOrEqualAssertion{Expected: expected}, nil
	case "less_or_equal", "lte":
		return &LessOrEqualAssertion{Expected: expected}, nil
	case "in_range":
		min, minOk := options["min"]
		max, maxOk := options["max"]
		if !minOk || !maxOk {
			return nil, fmt.Errorf("in_range operator requires 'min' and 'max' in options")
		}
		return &InRangeAssertion{Min: min, Max: max}, nil
	case "json_path":
		path, ok := expected.(string)
		if !ok {
			return nil, fmt.Errorf("json_path operator requires string path as expected value, got %T", expected)
		}
		expectedVal, hasExpected := options["expected"]
		if !hasExpected {
			return nil, fmt.Errorf("json_path operator requires 'expected' in options")
		}
		return &JSONPathAssertion{Path: path, Expected: expectedVal}, nil
	case "not_empty":
		return &NotEmptyAssertion{}, nil
	case "status_code":
		code, err := toInt(expected)
		if err != nil {
			return nil, fmt.Errorf("status_code operator requires integer expected value: %w", err)
		}
		return &StatusCodeAssertion{Expected: code}, nil
	case "status_class":
		s, ok := expected.(string)
		if !ok {
			return nil, fmt.Errorf("status_class operator requires string expected value, got %T", expected)
		}
		return NewStatusClassAssertion(s)
	case "header_equals":
		headerName, ok := options["header"].(string)
		if !ok {
			return nil, fmt.Errorf("header_equals operator requires 'header' string in options")
		}
		return &HeaderEqualsAssertion{Header: headerName, Expected: expected}, nil
	case "header_contains":
		headerName, ok := options["header"].(string)
		if !ok {
			return nil, fmt.Errorf("header_contains operator requires 'header' string in options")
		}
		substr, ok := expected.(string)
		if !ok {
			return nil, fmt.Errorf("header_contains operator requires string expected value, got %T", expected)
		}
		return &HeaderContainsAssertion{Header: headerName, Substr: substr}, nil
	case "header_exists":
		headerName, ok := expected.(string)
		if !ok {
			return nil, fmt.Errorf("header_exists operator requires string expected value (header name), got %T", expected)
		}
		return &HeaderExistsAssertion{Header: headerName}, nil
	case "response_time_below":
		d, err := toDuration(expected)
		if err != nil {
			return nil, fmt.Errorf("response_time_below operator: %w", err)
		}
		return &ResponseTimeBelowAssertion{MaxDuration: d}, nil
	case "length":
		l, err := toInt(expected)
		if err != nil {
			return nil, fmt.Errorf("length operator requires integer expected value: %w", err)
		}
		return &LengthAssertion{Expected: l}, nil
	case "empty":
		return &EmptyAssertion{}, nil
	case "collection_not_empty":
		return &CollectionNotEmptyAssertion{}, nil
	case "each":
		inner, ok := options["assertion"].(Assertion)
		if !ok {
			return nil, fmt.Errorf("each operator requires 'assertion' of type Assertion in options")
		}
		return &EachAssertion{Inner: inner}, nil
	case "any":
		inner, ok := options["assertion"].(Assertion)
		if !ok {
			return nil, fmt.Errorf("any operator requires 'assertion' of type Assertion in options")
		}
		return &AnyAssertion{Inner: inner}, nil
	case "all":
		inner, ok := options["assertion"].(Assertion)
		if !ok {
			return nil, fmt.Errorf("all operator requires 'assertion' of type Assertion in options")
		}
		return &AllAssertion{Inner: inner}, nil
	default:
		return nil, fmt.Errorf("unknown operator: %q", operator)
	}
}
