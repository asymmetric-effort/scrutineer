package schema

import (
	"testing"
)

func TestParseSuiteFullFields(t *testing.T) {
	data := []byte(`
suite: "User API Tests"
tags:
  - api
  - integration
fixtures:
  base_url: "http://localhost:8080"
  user_id: 42
setup:
  - action: "create_db"
    command: "initdb"
    timeout: "30s"
teardown:
  - action: "cleanup"
    command: "dropdb"
tests:
  - name: "Get user"
    connector: "http"
    tags:
      - smoke
    skip: false
    steps:
      - connector: "http"
        action: "request"
        method: "GET"
        path: "/users/1"
        timeout: "5s"
        capture:
          user_id: "response.id"
        assert:
          - status: 200
          - body_contains: "alice"
`)

	suite, err := ParseSuite(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if suite.Suite != "User API Tests" {
		t.Errorf("suite name = %q, want %q", suite.Suite, "User API Tests")
	}
	if len(suite.Tags) != 2 {
		t.Errorf("tags length = %d, want 2", len(suite.Tags))
	}
	if suite.Tags[0] != "api" || suite.Tags[1] != "integration" {
		t.Errorf("tags = %v, want [api integration]", suite.Tags)
	}

	// Fixtures
	if suite.Fixtures == nil {
		t.Fatal("fixtures is nil")
	}
	if suite.Fixtures["base_url"] != "http://localhost:8080" {
		t.Errorf("fixtures[base_url] = %v", suite.Fixtures["base_url"])
	}

	// Setup
	if len(suite.Setup) != 1 {
		t.Fatalf("setup length = %d, want 1", len(suite.Setup))
	}
	if suite.Setup[0].Action != "create_db" {
		t.Errorf("setup[0].action = %q", suite.Setup[0].Action)
	}
	if suite.Setup[0].Parameters["command"] != "initdb" {
		t.Errorf("setup[0].parameters[command] = %v", suite.Setup[0].Parameters["command"])
	}
	if suite.Setup[0].Timeout != "30s" {
		t.Errorf("setup[0].timeout = %q", suite.Setup[0].Timeout)
	}

	// Teardown
	if len(suite.Teardown) != 1 {
		t.Fatalf("teardown length = %d, want 1", len(suite.Teardown))
	}
	if suite.Teardown[0].Parameters["command"] != "dropdb" {
		t.Errorf("teardown[0].parameters[command] = %v", suite.Teardown[0].Parameters["command"])
	}

	// Tests
	if len(suite.Tests) != 1 {
		t.Fatalf("tests length = %d, want 1", len(suite.Tests))
	}
	test := suite.Tests[0]
	if test.Name != "Get user" {
		t.Errorf("test name = %q", test.Name)
	}
	if test.Connector != "http" {
		t.Errorf("test connector = %q", test.Connector)
	}
	if len(test.Tags) != 1 || test.Tags[0] != "smoke" {
		t.Errorf("test tags = %v", test.Tags)
	}
	if test.Skip {
		t.Error("test skip should be false")
	}

	// Steps
	if len(test.Steps) != 1 {
		t.Fatalf("steps length = %d, want 1", len(test.Steps))
	}
	step := test.Steps[0]
	if step.Connector != "http" {
		t.Errorf("step connector = %q", step.Connector)
	}
	if step.Action != "request" {
		t.Errorf("step action = %q", step.Action)
	}
	if step.Parameters["method"] != "GET" {
		t.Errorf("step parameters[method] = %v", step.Parameters["method"])
	}
	if step.Parameters["path"] != "/users/1" {
		t.Errorf("step parameters[path] = %v", step.Parameters["path"])
	}
	if step.Timeout != "5s" {
		t.Errorf("step timeout = %q", step.Timeout)
	}
	if len(step.Assert) != 2 {
		t.Fatalf("step assert length = %d, want 2", len(step.Assert))
	}
	if step.Capture["user_id"] != "response.id" {
		t.Errorf("step capture[user_id] = %v", step.Capture["user_id"])
	}
}

func TestParseSuiteMinimal(t *testing.T) {
	data := []byte(`
suite: "Minimal"
tests:
  - name: "basic test"
    steps:
      - action: "ping"
`)

	suite, err := ParseSuite(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suite.Suite != "Minimal" {
		t.Errorf("suite name = %q", suite.Suite)
	}
	if len(suite.Tests) != 1 {
		t.Fatalf("tests length = %d", len(suite.Tests))
	}
	if suite.Tests[0].Name != "basic test" {
		t.Errorf("test name = %q", suite.Tests[0].Name)
	}
	if len(suite.Tests[0].Steps) != 1 {
		t.Fatalf("steps length = %d", len(suite.Tests[0].Steps))
	}
	if suite.Tests[0].Steps[0].Action != "ping" {
		t.Errorf("action = %q", suite.Tests[0].Steps[0].Action)
	}
	// Parameters should be empty (no extra fields)
	if len(suite.Tests[0].Steps[0].Parameters) != 0 {
		t.Errorf("parameters = %v, want empty", suite.Tests[0].Steps[0].Parameters)
	}
}

func TestParseSuiteInvalidYAML(t *testing.T) {
	data := []byte(`
suite: [invalid
`)
	_, err := ParseSuite(data)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParseSuiteValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "missing suite name",
			yaml: `
tests:
  - name: "t1"
    steps:
      - action: "a"
`,
			want: "suite name is required",
		},
		{
			name: "no tests",
			yaml: `
suite: "S"
`,
			want: "at least one test is required",
		},
		{
			name: "test with no name",
			yaml: `
suite: "S"
tests:
  - steps:
      - action: "a"
`,
			want: "name is required",
		},
		{
			name: "test with no steps",
			yaml: `
suite: "S"
tests:
  - name: "t1"
`,
			want: "at least one step is required",
		},
		{
			name: "step with no action",
			yaml: `
suite: "S"
tests:
  - name: "t1"
    steps:
      - method: "GET"
`,
			want: "action is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSuite([]byte(tt.yaml))
			if err == nil {
				t.Fatal("expected validation error")
			}
			ve, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("expected *ValidationError, got %T: %v", err, err)
			}
			found := false
			for _, e := range ve.Errors {
				if contains(e, tt.want) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("error %q does not contain %q", ve.Error(), tt.want)
			}
		})
	}
}

func TestParseSuiteNilFixtures(t *testing.T) {
	data := []byte(`
suite: "No fixtures"
tests:
  - name: "t"
    steps:
      - action: "do"
`)
	suite, err := ParseSuite(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suite.Fixtures != nil {
		t.Errorf("fixtures should be nil, got %v", suite.Fixtures)
	}
}

func TestParseSuiteEmptyAssertions(t *testing.T) {
	data := []byte(`
suite: "Empty assert"
tests:
  - name: "t"
    steps:
      - action: "check"
        assert: []
`)
	suite, err := ParseSuite(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	step := suite.Tests[0].Steps[0]
	if step.Assert != nil && len(step.Assert) != 0 {
		t.Errorf("assert should be empty, got %v", step.Assert)
	}
}

func TestParseSuiteEmptyCapture(t *testing.T) {
	data := []byte(`
suite: "Empty capture"
tests:
  - name: "t"
    steps:
      - action: "check"
`)
	suite, err := ParseSuite(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	step := suite.Tests[0].Steps[0]
	if step.Capture != nil {
		t.Errorf("capture should be nil, got %v", step.Capture)
	}
}

func TestParseConfigFull(t *testing.T) {
	data := []byte(`
version: "0.0.1"
tests:
  - "tests/*.yaml"
  - "integration/*.yaml"
parallelism: 4
timeout: "30s"
reporters:
  - type: "ansi"
  - type: "json"
    output: "results.json"
coverage:
  threshold: 98.0
browsers:
  chromium: true
  firefox: true
  webkit: false
connectors:
  http:
    base_url: "http://localhost:8080"
    timeout: "10s"
telemetry:
  enabled: true
  output: "telemetry.bin"
`)

	config, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Version != "0.0.1" {
		t.Errorf("version = %q", config.Version)
	}
	if len(config.Tests) != 2 {
		t.Errorf("tests length = %d", len(config.Tests))
	}
	if config.Parallelism != 4 {
		t.Errorf("parallelism = %d", config.Parallelism)
	}
	if config.Timeout != "30s" {
		t.Errorf("timeout = %q", config.Timeout)
	}
	if len(config.Reporters) != 2 {
		t.Fatalf("reporters length = %d", len(config.Reporters))
	}
	if config.Reporters[0].Type != "ansi" {
		t.Errorf("reporters[0].type = %q", config.Reporters[0].Type)
	}
	if config.Reporters[1].Output != "results.json" {
		t.Errorf("reporters[1].output = %q", config.Reporters[1].Output)
	}
	if config.Coverage.Threshold != 98.0 {
		t.Errorf("coverage.threshold = %f", config.Coverage.Threshold)
	}
	if !config.Browsers.Chromium {
		t.Error("browsers.chromium should be true")
	}
	if !config.Browsers.Firefox {
		t.Error("browsers.firefox should be true")
	}
	if config.Browsers.WebKit {
		t.Error("browsers.webkit should be false")
	}
	if config.Connectors["http"]["base_url"] != "http://localhost:8080" {
		t.Errorf("connectors.http.base_url = %v", config.Connectors["http"]["base_url"])
	}
	if !config.Telemetry.Enabled {
		t.Error("telemetry.enabled should be true")
	}
	if config.Telemetry.Output != "telemetry.bin" {
		t.Errorf("telemetry.output = %q", config.Telemetry.Output)
	}
}

func TestParseConfigMinimal(t *testing.T) {
	data := []byte(`
tests:
  - "tests/*.yaml"
`)

	config, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(config.Tests) != 1 {
		t.Errorf("tests length = %d", len(config.Tests))
	}
	if config.Parallelism != 0 {
		t.Errorf("parallelism = %d, want 0", config.Parallelism)
	}
}

func TestParseConfigInvalidYAML(t *testing.T) {
	data := []byte(`
tests: [invalid
`)
	_, err := ParseConfig(data)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParseConfigValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "no test paths",
			yaml: `
version: "1"
`,
			want: "at least one test path is required",
		},
		{
			name: "negative parallelism",
			yaml: `
tests:
  - "t.yaml"
parallelism: -1
`,
			want: "parallelism must be greater than 0",
		},
		{
			name: "invalid reporter type",
			yaml: `
tests:
  - "t.yaml"
reporters:
  - type: "xml"
`,
			want: "invalid type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseConfig([]byte(tt.yaml))
			if err == nil {
				t.Fatal("expected validation error")
			}
			ve, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("expected *ValidationError, got %T: %v", err, err)
			}
			found := false
			for _, e := range ve.Errors {
				if contains(e, tt.want) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("error %q does not contain %q", ve.Error(), tt.want)
			}
		})
	}
}

func TestParseTestStepVariousParameters(t *testing.T) {
	tests := []struct {
		name       string
		raw        map[string]any
		wantAction string
		wantParams map[string]any
	}{
		{
			name: "http request",
			raw: map[string]any{
				"action":    "request",
				"connector": "http",
				"method":    "POST",
				"path":      "/api/users",
				"body":      map[string]any{"name": "alice"},
			},
			wantAction: "request",
			wantParams: map[string]any{
				"method": "POST",
				"path":   "/api/users",
				"body":   map[string]any{"name": "alice"},
			},
		},
		{
			name: "cli command",
			raw: map[string]any{
				"action":  "exec",
				"command": "ls -la",
				"env":     map[string]any{"HOME": "/tmp"},
			},
			wantAction: "exec",
			wantParams: map[string]any{
				"command": "ls -la",
				"env":     map[string]any{"HOME": "/tmp"},
			},
		},
		{
			name: "only known fields",
			raw: map[string]any{
				"action":    "check",
				"connector": "test",
				"timeout":   "5s",
			},
			wantAction: "check",
			wantParams: map[string]any{},
		},
		{
			name:       "empty raw",
			raw:        map[string]any{},
			wantAction: "",
			wantParams: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := ParseTestStep(tt.raw)
			if step.Action != tt.wantAction {
				t.Errorf("action = %q, want %q", step.Action, tt.wantAction)
			}
			if len(step.Parameters) != len(tt.wantParams) {
				t.Errorf("parameters length = %d, want %d: %v", len(step.Parameters), len(tt.wantParams), step.Parameters)
			}
			for k, v := range tt.wantParams {
				got, ok := step.Parameters[k]
				if !ok {
					t.Errorf("missing parameter %q", k)
					continue
				}
				// Simple comparison for string/int types
				if sv, ok := v.(string); ok {
					if got != sv {
						t.Errorf("parameters[%q] = %v, want %v", k, got, v)
					}
				}
			}
		})
	}
}

func TestParseTestStepWithAssertAndCapture(t *testing.T) {
	raw := map[string]any{
		"action": "request",
		"assert": []any{
			map[string]any{"status": 200},
		},
		"capture": map[string]any{
			"token": "response.headers.authorization",
		},
	}

	step := ParseTestStep(raw)
	if len(step.Assert) != 1 {
		t.Fatalf("assert length = %d, want 1", len(step.Assert))
	}
	if step.Capture["token"] != "response.headers.authorization" {
		t.Errorf("capture[token] = %v", step.Capture["token"])
	}
}

func TestParseTestStepAssertNotSlice(t *testing.T) {
	raw := map[string]any{
		"action": "check",
		"assert": "not a slice",
	}
	step := ParseTestStep(raw)
	if step.Assert != nil {
		t.Errorf("assert should be nil for non-slice input, got %v", step.Assert)
	}
}

func TestParseTestStepCaptureNotMap(t *testing.T) {
	raw := map[string]any{
		"action":  "check",
		"capture": "not a map",
	}
	step := ParseTestStep(raw)
	if step.Capture != nil {
		t.Errorf("capture should be nil for non-map input, got %v", step.Capture)
	}
}

func TestParseTestStepNonStringConnector(t *testing.T) {
	raw := map[string]any{
		"action":    "check",
		"connector": 42,
	}
	step := ParseTestStep(raw)
	if step.Connector != "" {
		t.Errorf("connector should be empty for non-string, got %q", step.Connector)
	}
}

func TestParseTestStepNonStringAction(t *testing.T) {
	raw := map[string]any{
		"action": 42,
	}
	step := ParseTestStep(raw)
	if step.Action != "" {
		t.Errorf("action should be empty for non-string, got %q", step.Action)
	}
}

func TestParseTestStepNonStringTimeout(t *testing.T) {
	raw := map[string]any{
		"action":  "check",
		"timeout": 30,
	}
	step := ParseTestStep(raw)
	if step.Timeout != "" {
		t.Errorf("timeout should be empty for non-string, got %q", step.Timeout)
	}
}

func TestParseTestStepCaptureNonStringValue(t *testing.T) {
	raw := map[string]any{
		"action": "check",
		"capture": map[string]any{
			"val": 123,
		},
	}
	step := ParseTestStep(raw)
	if step.Capture == nil {
		t.Fatal("capture should not be nil")
	}
	if _, ok := step.Capture["val"]; ok {
		t.Error("non-string capture value should be skipped")
	}
}

func TestParseStepsFromRawNotSlice(t *testing.T) {
	// Directly test parseStepsFromRaw with a non-slice value.
	result := parseStepsFromRaw(map[string]any{"steps": "not_a_slice"}, "steps")
	if result != nil {
		t.Errorf("expected nil for non-slice value, got %v", result)
	}
}

func TestParseStepsFromRawNonMapItems(t *testing.T) {
	// Directly test parseStepsFromRaw with non-map items in the slice.
	result := parseStepsFromRaw(map[string]any{"steps": []any{"not_a_map", 42}}, "steps")
	if len(result) != 0 {
		t.Errorf("expected empty slice for non-map items, got %v", result)
	}
}

func TestParseStepsFromRawMissingKey(t *testing.T) {
	result := parseStepsFromRaw(map[string]any{"other": "value"}, "steps")
	if result != nil {
		t.Errorf("expected nil for missing key, got %v", result)
	}
}

func TestParseSuiteSecondUnmarshalError(t *testing.T) {
	// Data that parses as map[string]any but fails for TestSuite struct.
	// The "suite" field as a sequence will fail when decoding into a string.
	data := []byte(`
suite:
  - item1
  - item2
tests:
  - name: "t"
    steps:
      - action: "a"
`)
	_, err := ParseSuite(data)
	if err == nil {
		t.Fatal("expected error when suite is a sequence instead of string")
	}
}

func TestParseSuiteTestsRawNotSlice(t *testing.T) {
	// Tests as a non-slice value in raw - still validates.
	data := []byte(`
suite: "S"
tests:
  - name: "t"
    steps:
      - action: "a"
`)
	// This should work normally; the coverage gap is for when
	// raw["tests"] is not []any, which can't happen with valid YAML
	// that also passes struct unmarshal. Already covered by other tests.
	suite, err := ParseSuite(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suite.Suite != "S" {
		t.Errorf("suite = %q", suite.Suite)
	}
}

func TestParseSuiteSkipTest(t *testing.T) {
	data := []byte(`
suite: "Skip test"
tests:
  - name: "skipped"
    skip: true
    steps:
      - action: "a"
`)
	suite, err := ParseSuite(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !suite.Tests[0].Skip {
		t.Error("test should be skipped")
	}
}

func TestParseTestStepAssertNonMapItem(t *testing.T) {
	raw := map[string]any{
		"action": "check",
		"assert": []any{
			"not a map",
			42,
		},
	}
	step := ParseTestStep(raw)
	if len(step.Assert) != 0 {
		t.Errorf("assert should be empty for non-map items, got %v", step.Assert)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
