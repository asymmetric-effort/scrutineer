package assertion

import (
	"fmt"
	"strings"
)

// ContainsAssertion checks that a string actual value contains the given substring.
type ContainsAssertion struct {
	Substr string
}

// Name returns the assertion name.
func (a *ContainsAssertion) Name() string { return "contains" }

// Evaluate checks that the actual string contains the substring.
func (a *ContainsAssertion) Evaluate(actual any) error {
	s, ok := actual.(string)
	if !ok {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Substr,
			Actual:    actual,
			Message:   fmt.Sprintf("expected string value, got %T", actual),
		}
	}
	if strings.Contains(s, a.Substr) {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Substr,
		Actual:    actual,
		Message:   fmt.Sprintf("expected %q to contain %q", s, a.Substr),
	}
}

// NotContainsAssertion checks that a string actual value does not contain the given substring.
type NotContainsAssertion struct {
	Substr string
}

// Name returns the assertion name.
func (a *NotContainsAssertion) Name() string { return "not_contains" }

// Evaluate checks that the actual string does not contain the substring.
func (a *NotContainsAssertion) Evaluate(actual any) error {
	s, ok := actual.(string)
	if !ok {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Substr,
			Actual:    actual,
			Message:   fmt.Sprintf("expected string value, got %T", actual),
		}
	}
	if !strings.Contains(s, a.Substr) {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Substr,
		Actual:    actual,
		Message:   fmt.Sprintf("expected %q to not contain %q", s, a.Substr),
	}
}

// HasPrefixAssertion checks that a string actual value starts with the given prefix.
type HasPrefixAssertion struct {
	Prefix string
}

// Name returns the assertion name.
func (a *HasPrefixAssertion) Name() string { return "has_prefix" }

// Evaluate checks that the actual string starts with the prefix.
func (a *HasPrefixAssertion) Evaluate(actual any) error {
	s, ok := actual.(string)
	if !ok {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Prefix,
			Actual:    actual,
			Message:   fmt.Sprintf("expected string value, got %T", actual),
		}
	}
	if strings.HasPrefix(s, a.Prefix) {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Prefix,
		Actual:    actual,
		Message:   fmt.Sprintf("expected %q to have prefix %q", s, a.Prefix),
	}
}

// HasSuffixAssertion checks that a string actual value ends with the given suffix.
type HasSuffixAssertion struct {
	Suffix string
}

// Name returns the assertion name.
func (a *HasSuffixAssertion) Name() string { return "has_suffix" }

// Evaluate checks that the actual string ends with the suffix.
func (a *HasSuffixAssertion) Evaluate(actual any) error {
	s, ok := actual.(string)
	if !ok {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Suffix,
			Actual:    actual,
			Message:   fmt.Sprintf("expected string value, got %T", actual),
		}
	}
	if strings.HasSuffix(s, a.Suffix) {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Suffix,
		Actual:    actual,
		Message:   fmt.Sprintf("expected %q to have suffix %q", s, a.Suffix),
	}
}
