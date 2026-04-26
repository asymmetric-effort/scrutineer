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
