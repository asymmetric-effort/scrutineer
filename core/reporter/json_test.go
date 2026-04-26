package reporter

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestJSONReporter_NewJSONReporter(t *testing.T) {
	r := NewJSONReporter()
	if r == nil {
		t.Fatal("NewJSONReporter returned nil")
	}
}

func TestJSONReporter_ImplementsReporter(t *testing.T) {
	var _ Reporter = NewJSONReporter()
}

func TestJSONReporter_FlushEmpty(t *testing.T) {
	r := NewJSONReporter()
	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	var out jsonOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(out.Suites) != 0 {
		t.Errorf("expected 0 suites, got %d", len(out.Suites))
	}
}

func TestJSONReporter_SinglePassingTest(t *testing.T) {
	r := NewJSONReporter()

	suite := SuiteInfo{Name: "api", TestCount: 1}
	r.OnSuiteStart(suite)

	test := TestInfo{Name: "get_users", Suite: "api", Tags: []string{"rest", "smoke"}}
	r.OnTestStart(test)
	r.OnStepResult(test, StepResult{Name: "send request", Passed: true, Elapsed: 100 * time.Millisecond})
	r.OnStepResult(test, StepResult{Name: "check status", Passed: true, Elapsed: time.Millisecond})
	r.OnTestEnd(test, TestResult{Passed: true, Elapsed: 101 * time.Millisecond, Steps: nil})
	r.OnSuiteEnd(suite, SuiteSummary{Passed: 1, Elapsed: 101 * time.Millisecond})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	var out jsonOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(out.Suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(out.Suites))
	}
	s := out.Suites[0]
	if s.Name != "api" {
		t.Errorf("suite name: got %q, want %q", s.Name, "api")
	}
	if len(s.Tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(s.Tests))
	}
	tt := s.Tests[0]
	if tt.Name != "get_users" {
		t.Errorf("test name: got %q", tt.Name)
	}
	if !tt.Passed {
		t.Errorf("test should be passed")
	}
	if len(tt.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tt.Tags))
	}
	if len(tt.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(tt.Steps))
	}
	if !tt.Steps[0].Passed {
		t.Errorf("step 0 should be passed")
	}
	if tt.Steps[0].Error != "" {
		t.Errorf("step 0 error should be empty")
	}

	if s.Summary.Passed != 1 {
		t.Errorf("summary passed: got %d", s.Summary.Passed)
	}
}

func TestJSONReporter_FailingTest(t *testing.T) {
	r := NewJSONReporter()

	suite := SuiteInfo{Name: "auth", TestCount: 1}
	r.OnSuiteStart(suite)

	test := TestInfo{Name: "login_fail", Suite: "auth", Tags: nil}
	r.OnTestStart(test)
	r.OnStepResult(test, StepResult{
		Name:    "submit form",
		Passed:  false,
		Elapsed: 200 * time.Millisecond,
		Error:   errors.New("invalid credentials"),
	})
	r.OnTestEnd(test, TestResult{Passed: false, Elapsed: 200 * time.Millisecond})
	r.OnSuiteEnd(suite, SuiteSummary{Failed: 1, Elapsed: 200 * time.Millisecond})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	var out jsonOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	tt := out.Suites[0].Tests[0]
	if tt.Passed {
		t.Errorf("test should not be passed")
	}
	// Tags should be an empty array, not null.
	if tt.Tags == nil {
		t.Errorf("tags should be empty array, not nil")
	}
	if tt.Steps[0].Error != "invalid credentials" {
		t.Errorf("step error: got %q", tt.Steps[0].Error)
	}
	if out.Suites[0].Summary.Failed != 1 {
		t.Errorf("summary failed: got %d", out.Suites[0].Summary.Failed)
	}
}

func TestJSONReporter_MixedResults(t *testing.T) {
	r := NewJSONReporter()

	suite := SuiteInfo{Name: "mixed", TestCount: 3}
	r.OnSuiteStart(suite)

	// Passing test.
	t1 := TestInfo{Name: "t1", Suite: "mixed", Tags: []string{}}
	r.OnTestStart(t1)
	r.OnStepResult(t1, StepResult{Name: "s1", Passed: true})
	r.OnTestEnd(t1, TestResult{Passed: true})

	// Failing test.
	t2 := TestInfo{Name: "t2", Suite: "mixed", Tags: []string{"slow"}}
	r.OnTestStart(t2)
	r.OnStepResult(t2, StepResult{Name: "s1", Passed: false, Error: errors.New("err")})
	r.OnTestEnd(t2, TestResult{Passed: false})

	// Another passing test.
	t3 := TestInfo{Name: "t3", Suite: "mixed", Tags: []string{}}
	r.OnTestStart(t3)
	r.OnTestEnd(t3, TestResult{Passed: true})

	r.OnSuiteEnd(suite, SuiteSummary{Passed: 2, Failed: 1, Skipped: 0, Elapsed: time.Second})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	var out jsonOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(out.Suites[0].Tests) != 3 {
		t.Fatalf("expected 3 tests, got %d", len(out.Suites[0].Tests))
	}

	// Verify order.
	if out.Suites[0].Tests[0].Name != "t1" {
		t.Errorf("first test should be t1")
	}
	if out.Suites[0].Tests[1].Name != "t2" {
		t.Errorf("second test should be t2")
	}
	if out.Suites[0].Tests[2].Name != "t3" {
		t.Errorf("third test should be t3")
	}
}

func TestJSONReporter_MultipleSuites(t *testing.T) {
	r := NewJSONReporter()

	s1 := SuiteInfo{Name: "suite1", TestCount: 1}
	r.OnSuiteStart(s1)
	t1 := TestInfo{Name: "t1", Suite: "suite1", Tags: []string{}}
	r.OnTestStart(t1)
	r.OnTestEnd(t1, TestResult{Passed: true})
	r.OnSuiteEnd(s1, SuiteSummary{Passed: 1})

	s2 := SuiteInfo{Name: "suite2", TestCount: 1}
	r.OnSuiteStart(s2)
	t2 := TestInfo{Name: "t2", Suite: "suite2", Tags: []string{}}
	r.OnTestStart(t2)
	r.OnTestEnd(t2, TestResult{Passed: false})
	r.OnSuiteEnd(s2, SuiteSummary{Failed: 1})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	var out jsonOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(out.Suites) != 2 {
		t.Fatalf("expected 2 suites, got %d", len(out.Suites))
	}
	if out.Suites[0].Name != "suite1" {
		t.Errorf("first suite: got %q", out.Suites[0].Name)
	}
	if out.Suites[1].Name != "suite2" {
		t.Errorf("second suite: got %q", out.Suites[1].Name)
	}
}

func TestJSONReporter_EmptySuiteName(t *testing.T) {
	r := NewJSONReporter()

	suite := SuiteInfo{Name: "", TestCount: 0}
	r.OnSuiteStart(suite)
	r.OnSuiteEnd(suite, SuiteSummary{})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	var out jsonOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if out.Suites[0].Name != "" {
		t.Errorf("expected empty suite name")
	}
}

func TestJSONReporter_ZeroElapsed(t *testing.T) {
	r := NewJSONReporter()

	suite := SuiteInfo{Name: "zero", TestCount: 1}
	r.OnSuiteStart(suite)

	test := TestInfo{Name: "t", Suite: "zero", Tags: []string{}}
	r.OnTestStart(test)
	r.OnStepResult(test, StepResult{Name: "s", Passed: true, Elapsed: 0})
	r.OnTestEnd(test, TestResult{Passed: true, Elapsed: 0})
	r.OnSuiteEnd(suite, SuiteSummary{Passed: 1, Elapsed: 0})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	var out jsonOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if out.Suites[0].Tests[0].Steps[0].Elapsed != "0s" {
		t.Errorf("expected 0s elapsed, got %q", out.Suites[0].Tests[0].Steps[0].Elapsed)
	}
}

func TestJSONReporter_StepResultNilError(t *testing.T) {
	r := NewJSONReporter()

	suite := SuiteInfo{Name: "s", TestCount: 1}
	r.OnSuiteStart(suite)

	test := TestInfo{Name: "t", Suite: "s", Tags: []string{}}
	r.OnTestStart(test)
	r.OnStepResult(test, StepResult{Name: "step", Passed: false, Error: nil})
	r.OnTestEnd(test, TestResult{Passed: false})
	r.OnSuiteEnd(suite, SuiteSummary{Failed: 1})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	// Verify error field is omitted (omitempty).
	if bytes.Contains(buf.Bytes(), []byte(`"error"`)) {
		t.Errorf("expected error field to be omitted for nil error")
	}
}

func TestJSONReporter_StepResultUnknownTest(t *testing.T) {
	r := NewJSONReporter()

	suite := SuiteInfo{Name: "s", TestCount: 0}
	r.OnSuiteStart(suite)

	// OnStepResult for a test that was never started should not panic.
	unknownTest := TestInfo{Name: "unknown", Suite: "s"}
	r.OnStepResult(unknownTest, StepResult{Name: "step", Passed: true})
	r.OnTestEnd(unknownTest, TestResult{Passed: true})

	r.OnSuiteEnd(suite, SuiteSummary{})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	var out jsonOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// The unknown test should not appear.
	if len(out.Suites[0].Tests) != 0 {
		t.Errorf("expected 0 tests for unknown test scenario, got %d", len(out.Suites[0].Tests))
	}
}

func TestJSONReporter_OnSuiteEndNilCurrent(t *testing.T) {
	r := NewJSONReporter()
	// Calling OnSuiteEnd without OnSuiteStart should not panic.
	r.OnSuiteEnd(SuiteInfo{Name: "x"}, SuiteSummary{Passed: 1})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	var out jsonOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(out.Suites) != 0 {
		t.Errorf("expected 0 suites")
	}
}

func TestJSONReporter_ValidJSON(t *testing.T) {
	r := NewJSONReporter()

	suite := SuiteInfo{Name: "validation", TestCount: 1}
	r.OnSuiteStart(suite)

	test := TestInfo{Name: "json_test", Suite: "validation", Tags: []string{"json"}}
	r.OnTestStart(test)
	r.OnStepResult(test, StepResult{Name: "parse", Passed: true, Elapsed: time.Microsecond})
	r.OnTestEnd(test, TestResult{Passed: true, Elapsed: time.Microsecond})
	r.OnSuiteEnd(suite, SuiteSummary{Passed: 1, Elapsed: time.Microsecond})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	// Verify it's valid JSON by attempting to decode into a generic map.
	var raw map[string]any
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}

	// Verify top-level structure.
	if _, ok := raw["suites"]; !ok {
		t.Error("expected 'suites' key in JSON output")
	}
}

func TestJSONReporter_SkippedInSummary(t *testing.T) {
	r := NewJSONReporter()

	suite := SuiteInfo{Name: "skip", TestCount: 3}
	r.OnSuiteStart(suite)
	r.OnSuiteEnd(suite, SuiteSummary{Passed: 1, Failed: 0, Skipped: 2, Elapsed: time.Second})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	var out jsonOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if out.Suites[0].Summary.Skipped != 2 {
		t.Errorf("expected 2 skipped, got %d", out.Suites[0].Summary.Skipped)
	}
}

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{time.Millisecond, "1ms"},
		{5 * time.Second, "5s"},
		{time.Microsecond, "1\u00b5s"},
	}
	for _, tt := range tests {
		got := formatElapsed(tt.d)
		if got != tt.want {
			t.Errorf("formatElapsed(%v): got %q, want %q", tt.d, got, tt.want)
		}
	}
}
