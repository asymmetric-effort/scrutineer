// Package reporter defines the Reporter interface and associated types for
// reporting test suite execution results. Two implementations are provided:
// ANSIReporter for human-readable terminal output with color, and
// JSONReporter for machine-readable JSON output.
package reporter

import (
	"io"
	"time"
)

// SuiteInfo describes a test suite before or after execution.
type SuiteInfo struct {
	Name      string
	TestCount int
}

// TestInfo describes an individual test within a suite.
type TestInfo struct {
	Name  string
	Suite string
	Tags  []string
}

// StepResult captures the outcome of a single step within a test.
type StepResult struct {
	Name    string
	Passed  bool
	Elapsed time.Duration
	Error   error
}

// TestResult captures the aggregate outcome of a test.
type TestResult struct {
	Passed  bool
	Steps   []StepResult
	Elapsed time.Duration
}

// SuiteSummary captures the aggregate outcome of a suite.
type SuiteSummary struct {
	Passed  int
	Failed  int
	Skipped int
	Elapsed time.Duration
}

// Reporter is the interface that test reporters must implement. Events are
// delivered in lifecycle order: OnSuiteStart, then for each test
// OnTestStart / OnStepResult* / OnTestEnd, then OnSuiteEnd. Flush writes
// accumulated output to w.
type Reporter interface {
	OnSuiteStart(suite SuiteInfo)
	OnTestStart(test TestInfo)
	OnStepResult(test TestInfo, step StepResult)
	OnTestEnd(test TestInfo, result TestResult)
	OnSuiteEnd(suite SuiteInfo, summary SuiteSummary)
	Flush(w io.Writer) error
}
