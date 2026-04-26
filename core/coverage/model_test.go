package coverage

import (
	"math"
	"testing"
)

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}

func TestSuiteCoveragePercent_AllRan(t *testing.T) {
	s := &SuiteCoverage{TotalTests: 5, RanTests: 5}
	if !floatEqual(s.Percent(), 100) {
		t.Errorf("expected 100, got %f", s.Percent())
	}
}

func TestSuiteCoveragePercent_SomeSkipped(t *testing.T) {
	s := &SuiteCoverage{TotalTests: 10, RanTests: 7, SkippedTests: 3}
	if !floatEqual(s.Percent(), 70) {
		t.Errorf("expected 70, got %f", s.Percent())
	}
}

func TestSuiteCoveragePercent_NoneRan(t *testing.T) {
	s := &SuiteCoverage{TotalTests: 5, RanTests: 0}
	if !floatEqual(s.Percent(), 0) {
		t.Errorf("expected 0, got %f", s.Percent())
	}
}

func TestSuiteCoveragePercent_ZeroTotal(t *testing.T) {
	s := &SuiteCoverage{TotalTests: 0, RanTests: 0}
	if !floatEqual(s.Percent(), 0) {
		t.Errorf("expected 0, got %f", s.Percent())
	}
}

func TestTestCoveragePercent_AllStepsAndAssertions(t *testing.T) {
	tc := &TestCoverage{
		Ran:              true,
		TotalSteps:       4,
		ExecutedSteps:    4,
		TotalAssertions:  6,
		PassedAssertions: 6,
	}
	if !floatEqual(tc.Percent(), 100) {
		t.Errorf("expected 100, got %f", tc.Percent())
	}
}

func TestTestCoveragePercent_Partial(t *testing.T) {
	tc := &TestCoverage{
		Ran:              true,
		TotalSteps:       4,
		ExecutedSteps:    2,
		TotalAssertions:  4,
		PassedAssertions: 2,
	}
	// step coverage = 50%, assertion coverage = 50%, average = 50%
	if !floatEqual(tc.Percent(), 50) {
		t.Errorf("expected 50, got %f", tc.Percent())
	}
}

func TestTestCoveragePercent_NoStepsOrAssertions(t *testing.T) {
	tc := &TestCoverage{
		Ran:             true,
		TotalSteps:      0,
		TotalAssertions: 0,
	}
	// Ran with nothing to measure => 100%
	if !floatEqual(tc.Percent(), 100) {
		t.Errorf("expected 100, got %f", tc.Percent())
	}
}

func TestTestCoveragePercent_DidNotRun(t *testing.T) {
	tc := &TestCoverage{
		Ran:             false,
		TotalSteps:      4,
		TotalAssertions: 4,
	}
	if !floatEqual(tc.Percent(), 0) {
		t.Errorf("expected 0, got %f", tc.Percent())
	}
}

func TestTestCoveragePercent_OnlySteps(t *testing.T) {
	tc := &TestCoverage{
		Ran:             true,
		TotalSteps:      4,
		ExecutedSteps:   3,
		TotalAssertions: 0,
	}
	if !floatEqual(tc.Percent(), 75) {
		t.Errorf("expected 75, got %f", tc.Percent())
	}
}

func TestTestCoveragePercent_OnlyAssertions(t *testing.T) {
	tc := &TestCoverage{
		Ran:              true,
		TotalSteps:       0,
		TotalAssertions:  4,
		PassedAssertions: 1,
	}
	if !floatEqual(tc.Percent(), 25) {
		t.Errorf("expected 25, got %f", tc.Percent())
	}
}

func TestTestCoveragePercent_MixedStepAndAssertionCoverage(t *testing.T) {
	tc := &TestCoverage{
		Ran:              true,
		TotalSteps:       10,
		ExecutedSteps:    10,
		TotalAssertions:  10,
		PassedAssertions: 5,
	}
	// steps = 100%, assertions = 50%, average = 75%
	if !floatEqual(tc.Percent(), 75) {
		t.Errorf("expected 75, got %f", tc.Percent())
	}
}
