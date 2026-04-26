package assertion

import (
	"fmt"
	"regexp"
)

// MatchesAssertion checks that a string actual value matches the compiled regex pattern.
type MatchesAssertion struct {
	Pattern string
	re      *regexp.Regexp
}

// NewMatchesAssertion creates a new MatchesAssertion, compiling the regex pattern.
func NewMatchesAssertion(pattern string) (*MatchesAssertion, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}
	return &MatchesAssertion{Pattern: pattern, re: re}, nil
}

// Name returns the assertion name.
func (a *MatchesAssertion) Name() string { return "matches" }

// Evaluate checks that the actual string matches the regex pattern.
func (a *MatchesAssertion) Evaluate(actual any) error {
	s, ok := actual.(string)
	if !ok {
		return &AssertionError{
			Assertion: a.Name(),
			Expected:  a.Pattern,
			Actual:    actual,
			Message:   fmt.Sprintf("expected string value, got %T", actual),
		}
	}
	if a.re.MatchString(s) {
		return nil
	}
	return &AssertionError{
		Assertion: a.Name(),
		Expected:  a.Pattern,
		Actual:    actual,
		Message:   fmt.Sprintf("expected %q to match pattern %q", s, a.Pattern),
	}
}
