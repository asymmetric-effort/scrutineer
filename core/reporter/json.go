package reporter

import (
	"encoding/json"
	"io"
	"time"
)

// jsonSuiteResult is the top-level JSON structure for a suite.
type jsonSuiteResult struct {
	Name    string          `json:"name"`
	Tests   []jsonTestEntry `json:"tests"`
	Summary jsonSummary     `json:"summary"`
}

// jsonTestEntry is the JSON structure for a single test.
type jsonTestEntry struct {
	Name    string          `json:"name"`
	Suite   string          `json:"suite"`
	Tags    []string        `json:"tags"`
	Passed  bool            `json:"passed"`
	Elapsed string          `json:"elapsed"`
	Steps   []jsonStepEntry `json:"steps"`
}

// jsonStepEntry is the JSON structure for a single step.
type jsonStepEntry struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Elapsed string `json:"elapsed"`
	Error   string `json:"error,omitempty"`
}

// jsonSummary is the JSON structure for the suite summary.
type jsonSummary struct {
	Passed  int    `json:"passed"`
	Failed  int    `json:"failed"`
	Skipped int    `json:"skipped"`
	Elapsed string `json:"elapsed"`
}

// jsonOutput is the root JSON structure written by Flush.
type jsonOutput struct {
	Suites []jsonSuiteResult `json:"suites"`
}

// JSONReporter implements Reporter by collecting events and serializing
// them as JSON on Flush.
type JSONReporter struct {
	suites   []jsonSuiteResult
	current  *jsonSuiteResult
	tests    map[string]*jsonTestEntry // keyed by suite+test name
	testOrder []string                 // insertion order keys
}

// NewJSONReporter returns a new JSONReporter ready for use.
func NewJSONReporter() *JSONReporter {
	return &JSONReporter{
		tests: make(map[string]*jsonTestEntry),
	}
}

func testKey(suite, name string) string {
	return suite + "\x00" + name
}

// OnSuiteStart begins a new suite entry.
func (r *JSONReporter) OnSuiteStart(suite SuiteInfo) {
	s := jsonSuiteResult{
		Name:  suite.Name,
		Tests: []jsonTestEntry{},
	}
	r.suites = append(r.suites, s)
	r.current = &r.suites[len(r.suites)-1]
	// Reset per-suite state.
	r.tests = make(map[string]*jsonTestEntry)
	r.testOrder = nil
}

// OnTestStart registers a new test entry within the current suite.
func (r *JSONReporter) OnTestStart(test TestInfo) {
	tags := test.Tags
	if tags == nil {
		tags = []string{}
	}
	entry := jsonTestEntry{
		Name:  test.Name,
		Suite: test.Suite,
		Tags:  tags,
		Steps: []jsonStepEntry{},
	}
	key := testKey(test.Suite, test.Name)
	r.tests[key] = &entry
	r.testOrder = append(r.testOrder, key)
}

// OnStepResult appends a step to the current test entry.
func (r *JSONReporter) OnStepResult(test TestInfo, step StepResult) {
	key := testKey(test.Suite, test.Name)
	t, ok := r.tests[key]
	if !ok {
		return
	}
	entry := jsonStepEntry{
		Name:    step.Name,
		Passed:  step.Passed,
		Elapsed: formatElapsed(step.Elapsed),
	}
	if step.Error != nil {
		entry.Error = step.Error.Error()
	}
	t.Steps = append(t.Steps, entry)
}

// OnTestEnd finalizes the test entry with result data.
func (r *JSONReporter) OnTestEnd(test TestInfo, result TestResult) {
	key := testKey(test.Suite, test.Name)
	t, ok := r.tests[key]
	if !ok {
		return
	}
	t.Passed = result.Passed
	t.Elapsed = formatElapsed(result.Elapsed)
}

// OnSuiteEnd sets the summary on the current suite and finalizes the test list.
func (r *JSONReporter) OnSuiteEnd(suite SuiteInfo, summary SuiteSummary) {
	if r.current == nil {
		return
	}
	r.current.Summary = jsonSummary{
		Passed:  summary.Passed,
		Failed:  summary.Failed,
		Skipped: summary.Skipped,
		Elapsed: formatElapsed(summary.Elapsed),
	}
	// Build tests slice in insertion order.
	for _, key := range r.testOrder {
		if t, ok := r.tests[key]; ok {
			r.current.Tests = append(r.current.Tests, *t)
		}
	}
}

// Flush writes the accumulated results as JSON to w.
func (r *JSONReporter) Flush(w io.Writer) error {
	out := jsonOutput{Suites: r.suites}
	if out.Suites == nil {
		out.Suites = []jsonSuiteResult{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// formatElapsed returns a duration as a string.
func formatElapsed(d time.Duration) string {
	return d.String()
}
