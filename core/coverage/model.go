package coverage

// SuiteCoverage holds coverage data for a test suite.
type SuiteCoverage struct {
	Suite        string
	Tests        []TestCoverage
	TotalTests   int
	RanTests     int
	SkippedTests int
}

// TestCoverage holds coverage data for a single test.
type TestCoverage struct {
	Name             string
	Ran              bool
	Skipped          bool
	TotalSteps       int
	ExecutedSteps    int
	TotalAssertions  int
	PassedAssertions int
	FailedAssertions int
}

// Percent returns the coverage percentage (0-100) for a suite.
// Coverage is based on the ratio of tests that ran to total tests.
// If there are no tests, returns 0.
func (s *SuiteCoverage) Percent() float64 {
	if s.TotalTests == 0 {
		return 0
	}
	return float64(s.RanTests) / float64(s.TotalTests) * 100
}

// Percent returns the coverage percentage (0-100) for a test.
// Coverage is the average of step coverage and assertion coverage.
// If both totals are zero, returns 0 for a test that didn't run, 100 for one that did.
func (t *TestCoverage) Percent() float64 {
	if !t.Ran {
		return 0
	}

	hasSteps := t.TotalSteps > 0
	hasAssertions := t.TotalAssertions > 0

	if !hasSteps && !hasAssertions {
		return 100
	}

	var total float64
	var count float64

	if hasSteps {
		total += float64(t.ExecutedSteps) / float64(t.TotalSteps) * 100
		count++
	}

	if hasAssertions {
		total += float64(t.PassedAssertions) / float64(t.TotalAssertions) * 100
		count++
	}

	return total / count
}
