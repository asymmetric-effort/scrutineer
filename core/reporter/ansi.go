package reporter

import (
	"bytes"
	"fmt"
	"io"
	"time"
)

// ANSIReporter implements Reporter with color-coded terminal output.
type ANSIReporter struct {
	buf bytes.Buffer
}

// NewANSIReporter returns a new ANSIReporter ready for use.
func NewANSIReporter() *ANSIReporter {
	return &ANSIReporter{}
}

// OnSuiteStart prints the suite name in bold.
func (r *ANSIReporter) OnSuiteStart(suite SuiteInfo) {
	header := fmt.Sprintf("Suite: %s (%d tests)", suite.Name, suite.TestCount)
	fmt.Fprintln(&r.buf, Colorize(header, Bold))
}

// OnTestStart prints the test name.
func (r *ANSIReporter) OnTestStart(test TestInfo) {
	fmt.Fprintf(&r.buf, "  Test: %s\n", test.Name)
}

// OnStepResult prints a green checkmark for passing steps or a red X with
// the error message for failing steps.
func (r *ANSIReporter) OnStepResult(test TestInfo, step StepResult) {
	if step.Passed {
		fmt.Fprintf(&r.buf, "    %s %s %s\n",
			Colorize("\u2713", Green),
			step.Name,
			formatDuration(step.Elapsed),
		)
	} else {
		errMsg := ""
		if step.Error != nil {
			errMsg = step.Error.Error()
		}
		fmt.Fprintf(&r.buf, "    %s %s %s\n",
			Colorize("\u2717", Red),
			step.Name,
			formatDuration(step.Elapsed),
		)
		if errMsg != "" {
			fmt.Fprintf(&r.buf, "      %s\n", Colorize(errMsg, Red))
		}
	}
}

// OnTestEnd prints a summary line for the test with elapsed time.
func (r *ANSIReporter) OnTestEnd(test TestInfo, result TestResult) {
	status := Colorize("PASS", Green)
	if !result.Passed {
		status = Colorize("FAIL", Red)
	}
	fmt.Fprintf(&r.buf, "  %s %s (%s)\n", status, test.Name, result.Elapsed)
}

// OnSuiteEnd prints a color-coded summary of passed, failed, and skipped counts.
func (r *ANSIReporter) OnSuiteEnd(suite SuiteInfo, summary SuiteSummary) {
	fmt.Fprintln(&r.buf, "")
	fmt.Fprintf(&r.buf, "Results for %s:\n", suite.Name)
	fmt.Fprintf(&r.buf, "  %s  %s  %s  (%s)\n",
		Colorize(fmt.Sprintf("%d passed", summary.Passed), Green),
		Colorize(fmt.Sprintf("%d failed", summary.Failed), Red),
		Colorize(fmt.Sprintf("%d skipped", summary.Skipped), Yellow),
		summary.Elapsed,
	)
}

// Flush writes the accumulated buffer contents to w.
func (r *ANSIReporter) Flush(w io.Writer) error {
	_, err := r.buf.WriteTo(w)
	return err
}

// formatDuration returns a human-friendly duration string in parentheses,
// or an empty string for zero duration.
func formatDuration(d time.Duration) string {
	if d == 0 {
		return ""
	}
	return fmt.Sprintf("(%s)", d)
}
