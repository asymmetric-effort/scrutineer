package reporter

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func init() {
	// Ensure colors are enabled for ANSI tests.
	ColorEnabled = true
}

func TestANSIReporter_NewANSIReporter(t *testing.T) {
	r := NewANSIReporter()
	if r == nil {
		t.Fatal("NewANSIReporter returned nil")
	}
}

func TestANSIReporter_SuiteStart(t *testing.T) {
	r := NewANSIReporter()
	r.OnSuiteStart(SuiteInfo{Name: "MySuite", TestCount: 3})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Suite: MySuite (3 tests)") {
		t.Errorf("expected suite header, got: %q", out)
	}
	if !strings.Contains(out, Bold) {
		t.Errorf("expected bold ANSI code in output")
	}
}

func TestANSIReporter_TestStart(t *testing.T) {
	r := NewANSIReporter()
	r.OnTestStart(TestInfo{Name: "test_login", Suite: "auth"})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	if !strings.Contains(buf.String(), "Test: test_login") {
		t.Errorf("expected test name in output, got: %q", buf.String())
	}
}

func TestANSIReporter_StepResult_Pass(t *testing.T) {
	r := NewANSIReporter()
	test := TestInfo{Name: "t1", Suite: "s1"}
	r.OnStepResult(test, StepResult{
		Name:    "click button",
		Passed:  true,
		Elapsed: 50 * time.Millisecond,
	})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "\u2713") {
		t.Errorf("expected checkmark for passing step")
	}
	if !strings.Contains(out, Green) {
		t.Errorf("expected green color for passing step")
	}
	if !strings.Contains(out, "click button") {
		t.Errorf("expected step name")
	}
	if !strings.Contains(out, "50ms") {
		t.Errorf("expected elapsed time")
	}
}

func TestANSIReporter_StepResult_Fail(t *testing.T) {
	r := NewANSIReporter()
	test := TestInfo{Name: "t1", Suite: "s1"}
	r.OnStepResult(test, StepResult{
		Name:    "verify title",
		Passed:  false,
		Elapsed: 100 * time.Millisecond,
		Error:   errors.New("title mismatch"),
	})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "\u2717") {
		t.Errorf("expected X mark for failing step")
	}
	if !strings.Contains(out, Red) {
		t.Errorf("expected red color for failing step")
	}
	if !strings.Contains(out, "title mismatch") {
		t.Errorf("expected error message")
	}
}

func TestANSIReporter_StepResult_FailNilError(t *testing.T) {
	r := NewANSIReporter()
	test := TestInfo{Name: "t1", Suite: "s1"}
	r.OnStepResult(test, StepResult{
		Name:   "step with nil error",
		Passed: false,
		Error:  nil,
	})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "\u2717") {
		t.Errorf("expected X mark for failing step")
	}
	// Should not have an extra error line since Error is nil.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line (no error detail), got %d: %q", len(lines), out)
	}
}

func TestANSIReporter_StepResult_ZeroElapsed(t *testing.T) {
	r := NewANSIReporter()
	test := TestInfo{Name: "t1", Suite: "s1"}
	r.OnStepResult(test, StepResult{
		Name:    "instant step",
		Passed:  true,
		Elapsed: 0,
	})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	out := buf.String()

	// Zero elapsed should not show a duration in parentheses.
	if strings.Contains(out, "(") {
		t.Errorf("expected no parenthesized duration for zero elapsed, got: %q", out)
	}
}

func TestANSIReporter_TestEnd_Pass(t *testing.T) {
	r := NewANSIReporter()
	test := TestInfo{Name: "login_test", Suite: "auth"}
	r.OnTestEnd(test, TestResult{
		Passed:  true,
		Elapsed: 200 * time.Millisecond,
	})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "PASS") {
		t.Errorf("expected PASS in output")
	}
	if !strings.Contains(out, Green) {
		t.Errorf("expected green for PASS")
	}
	if !strings.Contains(out, "200ms") {
		t.Errorf("expected elapsed time")
	}
}

func TestANSIReporter_TestEnd_Fail(t *testing.T) {
	r := NewANSIReporter()
	test := TestInfo{Name: "login_test", Suite: "auth"}
	r.OnTestEnd(test, TestResult{
		Passed:  false,
		Elapsed: 1 * time.Second,
	})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "FAIL") {
		t.Errorf("expected FAIL in output")
	}
	if !strings.Contains(out, Red) {
		t.Errorf("expected red for FAIL")
	}
}

func TestANSIReporter_SuiteEnd(t *testing.T) {
	r := NewANSIReporter()
	suite := SuiteInfo{Name: "integration", TestCount: 5}
	summary := SuiteSummary{
		Passed:  3,
		Failed:  1,
		Skipped: 1,
		Elapsed: 5 * time.Second,
	}
	r.OnSuiteEnd(suite, summary)

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Results for integration") {
		t.Errorf("expected suite name in results")
	}
	if !strings.Contains(out, "3 passed") {
		t.Errorf("expected passed count")
	}
	if !strings.Contains(out, "1 failed") {
		t.Errorf("expected failed count")
	}
	if !strings.Contains(out, "1 skipped") {
		t.Errorf("expected skipped count")
	}
}

func TestANSIReporter_FullLifecycle(t *testing.T) {
	r := NewANSIReporter()

	suite := SuiteInfo{Name: "smoke", TestCount: 2}
	r.OnSuiteStart(suite)

	test1 := TestInfo{Name: "test_a", Suite: "smoke", Tags: []string{"fast"}}
	r.OnTestStart(test1)
	r.OnStepResult(test1, StepResult{Name: "step1", Passed: true, Elapsed: 10 * time.Millisecond})
	r.OnTestEnd(test1, TestResult{Passed: true, Elapsed: 10 * time.Millisecond})

	test2 := TestInfo{Name: "test_b", Suite: "smoke"}
	r.OnTestStart(test2)
	r.OnStepResult(test2, StepResult{Name: "step1", Passed: false, Error: errors.New("boom"), Elapsed: 5 * time.Millisecond})
	r.OnTestEnd(test2, TestResult{Passed: false, Elapsed: 5 * time.Millisecond})

	r.OnSuiteEnd(suite, SuiteSummary{Passed: 1, Failed: 1, Elapsed: 15 * time.Millisecond})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "smoke") {
		t.Errorf("expected suite name")
	}
	if !strings.Contains(out, "test_a") {
		t.Errorf("expected test_a")
	}
	if !strings.Contains(out, "test_b") {
		t.Errorf("expected test_b")
	}
	if !strings.Contains(out, "boom") {
		t.Errorf("expected error message")
	}
}

func TestANSIReporter_EmptySuite(t *testing.T) {
	r := NewANSIReporter()
	suite := SuiteInfo{Name: "", TestCount: 0}
	r.OnSuiteStart(suite)
	r.OnSuiteEnd(suite, SuiteSummary{})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Suite:  (0 tests)") {
		t.Errorf("expected empty suite header, got: %q", out)
	}
	if !strings.Contains(out, "0 passed") {
		t.Errorf("expected zero counts")
	}
}

func TestANSIReporter_FlushEmptyBuffer(t *testing.T) {
	r := NewANSIReporter()
	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %d bytes", buf.Len())
	}
}

func TestANSIReporter_ColorsDisabled(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	r := NewANSIReporter()
	r.OnSuiteStart(SuiteInfo{Name: "nocolor", TestCount: 1})
	test := TestInfo{Name: "t", Suite: "nocolor"}
	r.OnTestStart(test)
	r.OnStepResult(test, StepResult{Name: "s", Passed: true, Elapsed: time.Millisecond})
	r.OnTestEnd(test, TestResult{Passed: true, Elapsed: time.Millisecond})
	r.OnSuiteEnd(SuiteInfo{Name: "nocolor", TestCount: 1}, SuiteSummary{Passed: 1})

	var buf bytes.Buffer
	if err := r.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	out := buf.String()

	// Should not contain any ANSI escape codes.
	if strings.Contains(out, "\033[") {
		t.Errorf("expected no ANSI codes when colors disabled, got: %q", out)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, ""},
		{time.Millisecond, "(1ms)"},
		{5 * time.Second, "(5s)"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v): got %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestANSIReporter_ImplementsReporter(t *testing.T) {
	var _ Reporter = NewANSIReporter()
}
