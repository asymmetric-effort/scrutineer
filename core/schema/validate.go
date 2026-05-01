package schema

import (
	"fmt"
	"strings"
	"time"
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

// validateTests validates a slice of tests (used by both suite-level tests
// and interaction-level tests).
func validateTests(ve *ValidationError, tests []Test, prefix string) {
	for i, t := range tests {
		p := fmt.Sprintf("%stest[%d]", prefix, i)
		if t.Name == "" {
			ve.add(fmt.Sprintf("%s: name is required", p))
		}
		if len(t.Steps) == 0 {
			ve.add(fmt.Sprintf("%s: at least one step is required", p))
		}
		if t.Weight < 0 {
			ve.add(fmt.Sprintf("%s: weight must be >= 0", p))
		}
		for j, s := range t.Steps {
			if s.Action == "" {
				ve.add(fmt.Sprintf("%s.step[%d]: action is required", p, j))
			}
		}
	}
}

// ValidateSuite validates that a TestSuite is well-formed.
func ValidateSuite(suite *TestSuite) error {
	ve := &ValidationError{}

	if suite.Suite == "" {
		ve.add("suite name is required")
	}

	hasTests := len(suite.Tests) > 0
	hasInteractions := len(suite.Interactions) > 0

	if hasTests && hasInteractions {
		ve.add("suite must have either tests or interactions, not both")
	}

	if !hasTests && !hasInteractions {
		ve.add("at least one test or interaction is required")
	}

	if hasTests {
		validateTests(ve, suite.Tests, "")
	}

	if hasInteractions {
		for i, inter := range suite.Interactions {
			validateInteraction(ve, inter, i)
		}
	}

	if suite.Execution != nil {
		validateExecution(ve, suite.Execution)
	}

	if ve.hasErrors() {
		return ve
	}
	return nil
}

// validateExecution validates an Execution block.
func validateExecution(ve *ValidationError, exec *Execution) {
	if exec.Mode != "" && !validExecutionModes[exec.Mode] {
		ve.add(fmt.Sprintf("execution: invalid mode %q", exec.Mode))
	}

	if exec.Concurrency < 0 {
		ve.add("execution: concurrency must be >= 0")
	}

	if exec.Mode == ModeConcurrent && exec.Concurrency == 0 {
		ve.add("execution: concurrent mode requires concurrency > 0")
	}

	if exec.Duration != "" {
		if _, err := time.ParseDuration(exec.Duration); err != nil {
			ve.add(fmt.Sprintf("execution: invalid duration %q: %v", exec.Duration, err))
		}
	}

	if exec.Interval != "" {
		if _, err := time.ParseDuration(exec.Interval); err != nil {
			ve.add(fmt.Sprintf("execution: invalid interval %q: %v", exec.Interval, err))
		}
	}

	if exec.Repeat < 0 {
		ve.add("execution: repeat must be >= 0")
	}

	// repeat=0 means unlimited — requires duration to prevent infinite runs.
	if exec.Repeat == 0 && exec.Duration == "" {
		ve.add("execution: repeat=0 (unlimited) requires a duration")
	}

	// weighted mode needs a termination condition beyond single pass.
	if exec.Mode == ModeWeighted {
		hasDuration := exec.Duration != ""
		hasRepeat := exec.Repeat > 0
		if !hasDuration && !hasRepeat {
			ve.add("execution: weighted mode requires duration or repeat > 0")
		}
	}

	if exec.Fleet != nil {
		validateFleetConfig(ve, exec.Fleet)
	}
}

// validateInteraction validates a single Interaction.
func validateInteraction(ve *ValidationError, inter Interaction, index int) {
	prefix := fmt.Sprintf("interaction[%d]", index)

	if inter.Name == "" {
		ve.add(fmt.Sprintf("%s: name is required", prefix))
	}

	if inter.Weight < 0 {
		ve.add(fmt.Sprintf("%s: weight must be >= 0", prefix))
	}

	if inter.Mode != "" && !validExecutionModes[inter.Mode] {
		ve.add(fmt.Sprintf("%s: invalid mode %q", prefix, inter.Mode))
	}

	if len(inter.Tests) == 0 {
		ve.add(fmt.Sprintf("%s: at least one test is required", prefix))
	}

	validateTests(ve, inter.Tests, prefix+".")
}

// validateFleetConfig validates a FleetConfig.
func validateFleetConfig(ve *ValidationError, fleet *FleetConfig) {
	if len(fleet.Providers) == 0 {
		ve.add("fleet: at least one provider is required")
		return
	}

	totalWeight := 0
	for i, p := range fleet.Providers {
		prefix := fmt.Sprintf("fleet.provider[%d]", i)
		if p.Provider == "" {
			ve.add(fmt.Sprintf("%s: provider name is required", prefix))
		}
		if p.Weight < 0 {
			ve.add(fmt.Sprintf("%s: weight must be >= 0", prefix))
		}
		if p.TTL < 0 {
			ve.add(fmt.Sprintf("%s: ttl must be >= 0", prefix))
		}
		totalWeight += p.Weight
	}

	if totalWeight != 100 {
		ve.add(fmt.Sprintf("fleet: provider weights must sum to 100, got %d", totalWeight))
	}
}

// ValidateConfig validates that a Config is well-formed.
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
