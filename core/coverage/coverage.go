package coverage

import "sync"

// suiteData holds the internal mutable state for a suite being tracked.
type suiteData struct {
	name      string
	testCount int
	tests     map[string]*testData
	order     []string // preserves insertion order of test names
}

// testData holds the internal mutable state for a test being tracked.
type testData struct {
	name           string
	stepCount      int
	assertionCount int
	ran            bool
	skipped        bool
	executedSteps  int
	passedAsserts  int
	failedAsserts  int
}

// Tracker collects coverage data during test execution.
// It is safe for concurrent use.
type Tracker struct {
	mu     sync.Mutex
	suites map[string]*suiteData
	order  []string // preserves insertion order of suite names
}

// NewTracker creates a new coverage tracker.
func NewTracker() *Tracker {
	return &Tracker{
		suites: make(map[string]*suiteData),
	}
}

// RegisterSuite registers a suite and its expected test count.
func (t *Tracker) RegisterSuite(name string, testCount int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.suites[name]; exists {
		return
	}

	t.suites[name] = &suiteData{
		name:      name,
		testCount: testCount,
		tests:     make(map[string]*testData),
	}
	t.order = append(t.order, name)
}

// RegisterTest registers a test and its expected step/assertion counts.
func (t *Tracker) RegisterTest(suite, test string, stepCount, assertionCount int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	s, ok := t.suites[suite]
	if !ok {
		return
	}

	if _, exists := s.tests[test]; exists {
		return
	}

	s.tests[test] = &testData{
		name:           test,
		stepCount:      stepCount,
		assertionCount: assertionCount,
	}
	s.order = append(s.order, test)
}

// RecordTestRun records that a test was executed.
func (t *Tracker) RecordTestRun(suite, test string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	td := t.findTest(suite, test)
	if td == nil {
		return
	}
	td.ran = true
}

// RecordTestSkip records that a test was skipped.
func (t *Tracker) RecordTestSkip(suite, test string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	td := t.findTest(suite, test)
	if td == nil {
		return
	}
	td.skipped = true
}

// RecordStep records that a step was executed.
func (t *Tracker) RecordStep(suite, test string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	td := t.findTest(suite, test)
	if td == nil {
		return
	}
	td.executedSteps++
}

// RecordAssertion records an assertion result.
func (t *Tracker) RecordAssertion(suite, test string, passed bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	td := t.findTest(suite, test)
	if td == nil {
		return
	}
	if passed {
		td.passedAsserts++
	} else {
		td.failedAsserts++
	}
}

// findTest returns the testData for the given suite and test, or nil if not found.
// Must be called with t.mu held.
func (t *Tracker) findTest(suite, test string) *testData {
	s, ok := t.suites[suite]
	if !ok {
		return nil
	}
	td, ok := s.tests[test]
	if !ok {
		return nil
	}
	return td
}

// Results returns the collected coverage data.
func (t *Tracker) Results() []SuiteCoverage {
	t.mu.Lock()
	defer t.mu.Unlock()

	results := make([]SuiteCoverage, 0, len(t.order))

	for _, suiteName := range t.order {
		s := t.suites[suiteName]
		sc := SuiteCoverage{
			Suite:      s.name,
			TotalTests: s.testCount,
		}

		tests := make([]TestCoverage, 0, len(s.order))
		for _, testName := range s.order {
			td := s.tests[testName]
			tc := TestCoverage{
				Name:             td.name,
				Ran:              td.ran,
				Skipped:          td.skipped,
				TotalSteps:       td.stepCount,
				ExecutedSteps:    td.executedSteps,
				TotalAssertions:  td.assertionCount,
				PassedAssertions: td.passedAsserts,
				FailedAssertions: td.failedAsserts,
			}
			tests = append(tests, tc)

			if td.ran {
				sc.RanTests++
			}
			if td.skipped {
				sc.SkippedTests++
			}
		}

		sc.Tests = tests
		results = append(results, sc)
	}

	return results
}

// TotalPercent returns the overall coverage percentage across all suites.
// It is the weighted average based on total test counts.
func (t *Tracker) TotalPercent() float64 {
	results := t.Results()

	var totalTests int
	var ranTests int

	for _, s := range results {
		totalTests += s.TotalTests
		ranTests += s.RanTests
	}

	if totalTests == 0 {
		return 0
	}

	return float64(ranTests) / float64(totalTests) * 100
}
