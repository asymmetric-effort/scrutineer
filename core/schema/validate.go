package schema

import (
	"fmt"
	"strings"
)

// ValidationError collects multiple validation errors.
type ValidationError struct {
	Errors []string
}

// Error returns a formatted string of all validation errors.
func (ve *ValidationError) Error() string {
	return fmt.Sprintf("validation failed: %s", strings.Join(ve.Errors, "; "))
}

// add appends a message to the error list.
func (ve *ValidationError) add(msg string) {
	ve.Errors = append(ve.Errors, msg)
}

// hasErrors reports whether any errors have been recorded.
func (ve *ValidationError) hasErrors() bool {
	return len(ve.Errors) > 0
}

// ValidateSuite validates that a TestSuite is well-formed.
// It checks that the suite has a name, at least one test, each test has
// a name and at least one step, and each step has an action.
func ValidateSuite(suite *TestSuite) error {
	ve := &ValidationError{}

	if suite.Suite == "" {
		ve.add("suite name is required")
	}

	if len(suite.Tests) == 0 {
		ve.add("at least one test is required")
	}

	for i, t := range suite.Tests {
		if t.Name == "" {
			ve.add(fmt.Sprintf("test[%d]: name is required", i))
		}
		if len(t.Steps) == 0 {
			ve.add(fmt.Sprintf("test[%d]: at least one step is required", i))
		}
		for j, s := range t.Steps {
			if s.Action == "" {
				ve.add(fmt.Sprintf("test[%d].step[%d]: action is required", i, j))
			}
		}
	}

	if ve.hasErrors() {
		return ve
	}
	return nil
}

// ValidateConfig validates that a Config is well-formed.
// It checks that test paths are defined, parallelism is positive if set,
// and reporter types are valid.
func ValidateConfig(config *Config) error {
	ve := &ValidationError{}

	if len(config.Tests) == 0 {
		ve.add("at least one test path is required")
	}

	if config.Parallelism < 0 {
		ve.add("parallelism must be greater than 0")
	}

	for i, r := range config.Reporters {
		if !validReporterTypes[r.Type] {
			ve.add(fmt.Sprintf("reporters[%d]: invalid type %q", i, r.Type))
		}
	}

	if ve.hasErrors() {
		return ve
	}
	return nil
}
