package schema

import (
	"testing"
)

func TestValidateSuiteValid(t *testing.T) {
	suite := &TestSuite{
		Suite: "Valid Suite",
		Tests: []Test{
			{
				Name: "test1",
				Steps: []TestStep{
					{Action: "do_something"},
				},
			},
		},
	}
	if err := ValidateSuite(suite); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSuiteMissingSuiteName(t *testing.T) {
	suite := &TestSuite{
		Tests: []Test{
			{
				Name:  "t",
				Steps: []TestStep{{Action: "a"}},
			},
		},
	}
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	ve := err.(*ValidationError)
	if len(ve.Errors) != 1 {
		t.Errorf("expected 1 error, got %d: %v", len(ve.Errors), ve.Errors)
	}
}

func TestValidateSuiteNoTests(t *testing.T) {
	suite := &TestSuite{Suite: "S"}
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	ve := err.(*ValidationError)
	found := false
	for _, e := range ve.Errors {
		if contains(e, "at least one test") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'at least one test' error, got %v", ve.Errors)
	}
}

func TestValidateSuiteTestNoName(t *testing.T) {
	suite := &TestSuite{
		Suite: "S",
		Tests: []Test{
			{Steps: []TestStep{{Action: "a"}}},
		},
	}
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	ve := err.(*ValidationError)
	found := false
	for _, e := range ve.Errors {
		if contains(e, "name is required") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'name is required' error, got %v", ve.Errors)
	}
}

func TestValidateSuiteStepNoAction(t *testing.T) {
	suite := &TestSuite{
		Suite: "S",
		Tests: []Test{
			{
				Name:  "t",
				Steps: []TestStep{{Connector: "http"}},
			},
		},
	}
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	ve := err.(*ValidationError)
	found := false
	for _, e := range ve.Errors {
		if contains(e, "action is required") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'action is required' error, got %v", ve.Errors)
	}
}

func TestValidateSuiteMultipleErrors(t *testing.T) {
	suite := &TestSuite{
		// Missing suite name
		Tests: []Test{
			{
				// Missing test name
				Steps: []TestStep{
					{}, // Missing action
				},
			},
		},
	}
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	ve := err.(*ValidationError)
	if len(ve.Errors) != 3 {
		t.Errorf("expected 3 errors, got %d: %v", len(ve.Errors), ve.Errors)
	}
}

func TestValidateSuiteTestNoSteps(t *testing.T) {
	suite := &TestSuite{
		Suite: "S",
		Tests: []Test{
			{Name: "t"},
		},
	}
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	ve := err.(*ValidationError)
	found := false
	for _, e := range ve.Errors {
		if contains(e, "at least one step") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'at least one step' error, got %v", ve.Errors)
	}
}

func TestValidateConfigValid(t *testing.T) {
	config := &Config{
		Tests: []string{"tests/*.yaml"},
	}
	if err := ValidateConfig(config); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateConfigNoTests(t *testing.T) {
	config := &Config{}
	err := ValidateConfig(config)
	if err == nil {
		t.Fatal("expected error")
	}
	ve := err.(*ValidationError)
	found := false
	for _, e := range ve.Errors {
		if contains(e, "at least one test path") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'at least one test path' error, got %v", ve.Errors)
	}
}

func TestValidateConfigNegativeParallelism(t *testing.T) {
	config := &Config{
		Tests:       []string{"t.yaml"},
		Parallelism: -1,
	}
	err := ValidateConfig(config)
	if err == nil {
		t.Fatal("expected error")
	}
	ve := err.(*ValidationError)
	found := false
	for _, e := range ve.Errors {
		if contains(e, "parallelism") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected parallelism error, got %v", ve.Errors)
	}
}

func TestValidateConfigZeroParallelism(t *testing.T) {
	config := &Config{
		Tests:       []string{"t.yaml"},
		Parallelism: 0,
	}
	if err := ValidateConfig(config); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateConfigPositiveParallelism(t *testing.T) {
	config := &Config{
		Tests:       []string{"t.yaml"},
		Parallelism: 4,
	}
	if err := ValidateConfig(config); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateConfigInvalidReporter(t *testing.T) {
	config := &Config{
		Tests: []string{"t.yaml"},
		Reporters: []ReporterConfig{
			{Type: "html"},
		},
	}
	err := ValidateConfig(config)
	if err == nil {
		t.Fatal("expected error")
	}
	ve := err.(*ValidationError)
	found := false
	for _, e := range ve.Errors {
		if contains(e, "invalid type") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'invalid type' error, got %v", ve.Errors)
	}
}

func TestValidateConfigValidReporterTypes(t *testing.T) {
	config := &Config{
		Tests: []string{"t.yaml"},
		Reporters: []ReporterConfig{
			{Type: "ansi"},
			{Type: "json", Output: "out.json"},
		},
	}
	if err := ValidateConfig(config); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateConfigMultipleErrors(t *testing.T) {
	config := &Config{
		Parallelism: -2,
		Reporters: []ReporterConfig{
			{Type: "xml"},
		},
	}
	err := ValidateConfig(config)
	if err == nil {
		t.Fatal("expected error")
	}
	ve := err.(*ValidationError)
	// Should have: no tests, negative parallelism, invalid reporter
	if len(ve.Errors) != 3 {
		t.Errorf("expected 3 errors, got %d: %v", len(ve.Errors), ve.Errors)
	}
}

func TestValidationErrorString(t *testing.T) {
	ve := &ValidationError{Errors: []string{"a", "b"}}
	got := ve.Error()
	if got != "validation failed: a; b" {
		t.Errorf("Error() = %q", got)
	}
}

// --- Execution block validation tests ---

func validSuiteWithExecution(exec *Execution) *TestSuite {
	return &TestSuite{
		Suite: "S",
		Tests: []Test{
			{Name: "t", Steps: []TestStep{{Action: "a"}}},
		},
		Execution: exec,
	}
}

func validSuiteWithInteractions(interactions []Interaction) *TestSuite {
	return &TestSuite{
		Suite:        "S",
		Interactions: interactions,
	}
}

func TestValidateSuiteWithExecution(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:        ModeSequential,
		Concurrency: 10,
		Duration:    "5m",
		Repeat:      3,
		Interval:    "1s",
	})
	if err := ValidateSuite(suite); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSuiteExecutionDefaultRepeat(t *testing.T) {
	// repeat defaults to 0 in Go zero value, which requires duration.
	suite := validSuiteWithExecution(&Execution{
		Mode:     ModeSequential,
		Duration: "10s",
	})
	if err := ValidateSuite(suite); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSuiteInvalidMode(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:   "invalid",
		Repeat: 1,
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "invalid mode") {
		t.Errorf("expected 'invalid mode' error, got: %v", err)
	}
}

func TestValidateSuiteNegativeConcurrency(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:        ModeSequential,
		Concurrency: -1,
		Repeat:      1,
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "concurrency must be >= 0") {
		t.Errorf("expected concurrency error, got: %v", err)
	}
}

func TestValidateSuiteInvalidDuration(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:     ModeSequential,
		Duration: "not-a-duration",
		Repeat:   1,
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "invalid duration") {
		t.Errorf("expected 'invalid duration' error, got: %v", err)
	}
}

func TestValidateSuiteInvalidInterval(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:     ModeSequential,
		Interval: "bad",
		Repeat:   1,
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "invalid interval") {
		t.Errorf("expected 'invalid interval' error, got: %v", err)
	}
}

func TestValidateSuiteRepeatZeroNoDuration(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:   ModeSequential,
		Repeat: 0,
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error for repeat=0 without duration")
	}
	if !contains(err.Error(), "requires a duration") {
		t.Errorf("expected duration requirement error, got: %v", err)
	}
}

func TestValidateSuiteNegativeRepeat(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:   ModeSequential,
		Repeat: -1,
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "repeat must be >= 0") {
		t.Errorf("expected repeat error, got: %v", err)
	}
}

func TestValidateSuiteWeightedNoDurationNoRepeat(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode: ModeWeighted,
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error for weighted mode without termination")
	}
	if !contains(err.Error(), "weighted mode requires") {
		t.Errorf("expected weighted termination error, got: %v", err)
	}
}

func TestValidateSuiteWeightedWithDuration(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:     ModeWeighted,
		Duration: "5m",
	})
	if err := ValidateSuite(suite); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSuiteWeightedWithRepeat(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:   ModeWeighted,
		Repeat: 10,
	})
	if err := ValidateSuite(suite); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSuiteEmptyMode(t *testing.T) {
	// Empty mode is valid (defaults to sequential).
	suite := validSuiteWithExecution(&Execution{
		Repeat: 1,
	})
	if err := ValidateSuite(suite); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Interaction validation tests ---

func TestValidateSuiteWithInteractions(t *testing.T) {
	suite := validSuiteWithInteractions([]Interaction{
		{
			Name: "User session",
			Mode: ModeSequential,
			Tests: []Test{
				{Name: "login", Steps: []TestStep{{Action: "request"}}},
			},
		},
	})
	if err := ValidateSuite(suite); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSuiteBothTestsAndInteractions(t *testing.T) {
	suite := &TestSuite{
		Suite: "S",
		Tests: []Test{
			{Name: "t", Steps: []TestStep{{Action: "a"}}},
		},
		Interactions: []Interaction{
			{
				Name:  "I",
				Tests: []Test{{Name: "t2", Steps: []TestStep{{Action: "b"}}}},
			},
		},
	}
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "either tests or interactions, not both") {
		t.Errorf("expected mutual exclusion error, got: %v", err)
	}
}

func TestValidateSuiteInteractionNoName(t *testing.T) {
	suite := validSuiteWithInteractions([]Interaction{
		{
			Tests: []Test{
				{Name: "t", Steps: []TestStep{{Action: "a"}}},
			},
		},
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "name is required") {
		t.Errorf("expected name error, got: %v", err)
	}
}

func TestValidateSuiteInteractionNoTests(t *testing.T) {
	suite := validSuiteWithInteractions([]Interaction{
		{Name: "Empty"},
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "at least one test is required") {
		t.Errorf("expected test requirement error, got: %v", err)
	}
}

func TestValidateSuiteInteractionNegativeWeight(t *testing.T) {
	suite := validSuiteWithInteractions([]Interaction{
		{
			Name:   "I",
			Weight: -1,
			Tests: []Test{
				{Name: "t", Steps: []TestStep{{Action: "a"}}},
			},
		},
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "weight must be >= 0") {
		t.Errorf("expected weight error, got: %v", err)
	}
}

func TestValidateSuiteInteractionInvalidMode(t *testing.T) {
	suite := validSuiteWithInteractions([]Interaction{
		{
			Name: "I",
			Mode: "bad",
			Tests: []Test{
				{Name: "t", Steps: []TestStep{{Action: "a"}}},
			},
		},
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "invalid mode") {
		t.Errorf("expected mode error, got: %v", err)
	}
}

func TestValidateSuiteInteractionTestValidation(t *testing.T) {
	// Tests within interactions are validated too.
	suite := validSuiteWithInteractions([]Interaction{
		{
			Name: "I",
			Tests: []Test{
				{Name: "t", Steps: []TestStep{{}}}, // missing action
			},
		},
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "action is required") {
		t.Errorf("expected action error, got: %v", err)
	}
}

func TestValidateSuiteTestNegativeWeight(t *testing.T) {
	suite := &TestSuite{
		Suite: "S",
		Tests: []Test{
			{Name: "t", Weight: -5, Steps: []TestStep{{Action: "a"}}},
		},
	}
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "weight must be >= 0") {
		t.Errorf("expected weight error, got: %v", err)
	}
}

// --- Fleet validation tests ---

func TestValidateFleetValid(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:   ModeSequential,
		Repeat: 1,
		Fleet: &FleetConfig{
			Providers: []FleetProvider{
				{Provider: "static", Weight: 60, TTL: 0},
				{Provider: "aws_ec2", Weight: 40, TTL: 15},
			},
		},
	})
	if err := ValidateSuite(suite); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateFleetWeightsNot100(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:   ModeSequential,
		Repeat: 1,
		Fleet: &FleetConfig{
			Providers: []FleetProvider{
				{Provider: "static", Weight: 50},
				{Provider: "aws_ec2", Weight: 30},
			},
		},
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "must sum to 100") {
		t.Errorf("expected weight sum error, got: %v", err)
	}
}

func TestValidateFleetEmptyProviderName(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:   ModeSequential,
		Repeat: 1,
		Fleet: &FleetConfig{
			Providers: []FleetProvider{
				{Provider: "", Weight: 100},
			},
		},
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "provider name is required") {
		t.Errorf("expected provider name error, got: %v", err)
	}
}

func TestValidateFleetNegativeTTL(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:   ModeSequential,
		Repeat: 1,
		Fleet: &FleetConfig{
			Providers: []FleetProvider{
				{Provider: "static", Weight: 100, TTL: -5},
			},
		},
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "ttl must be >= 0") {
		t.Errorf("expected TTL error, got: %v", err)
	}
}

func TestValidateFleetNegativeWeight(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:   ModeSequential,
		Repeat: 1,
		Fleet: &FleetConfig{
			Providers: []FleetProvider{
				{Provider: "static", Weight: -10},
			},
		},
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "weight must be >= 0") {
		t.Errorf("expected weight error, got: %v", err)
	}
}

func TestValidateFleetNoProviders(t *testing.T) {
	suite := validSuiteWithExecution(&Execution{
		Mode:   ModeSequential,
		Repeat: 1,
		Fleet: &FleetConfig{
			Providers: []FleetProvider{},
		},
	})
	err := ValidateSuite(suite)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "at least one provider is required") {
		t.Errorf("expected provider requirement error, got: %v", err)
	}
}

// --- Backward compatibility ---

func TestValidateSuiteNoExecutionBackwardCompat(t *testing.T) {
	suite := &TestSuite{
		Suite: "S",
		Tests: []Test{
			{Name: "t", Steps: []TestStep{{Action: "a"}}},
		},
	}
	if err := ValidateSuite(suite); err != nil {
		t.Errorf("backward compatible suite should still pass: %v", err)
	}
}
