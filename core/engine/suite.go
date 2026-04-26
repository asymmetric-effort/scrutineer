package engine

import (
	"time"

	"github.com/scrutineer/scrutineer/core/reporter"
)

// SuiteResult holds results for a complete test suite.
type SuiteResult struct {
	Suite   string
	Results []reporter.TestResult
	Passed  int
	Failed  int
	Skipped int
	Elapsed time.Duration
}

// summarise computes Passed, Failed, and Skipped counts from the Results slice.
func (sr *SuiteResult) summarise() {
	for _, r := range sr.Results {
		if r.Passed {
			sr.Passed++
		} else {
			sr.Failed++
		}
	}
}
