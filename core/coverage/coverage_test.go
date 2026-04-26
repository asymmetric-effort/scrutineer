package coverage

import (
	"sync"
	"testing"
)

func TestTrackerFullLifecycle(t *testing.T) {
	tr := NewTracker()

	tr.RegisterSuite("auth", 2)
	tr.RegisterTest("auth", "login", 3, 2)
	tr.RegisterTest("auth", "logout", 2, 1)

	tr.RecordTestRun("auth", "login")
	tr.RecordStep("auth", "login")
	tr.RecordStep("auth", "login")
	tr.RecordStep("auth", "login")
	tr.RecordAssertion("auth", "login", true)
	tr.RecordAssertion("auth", "login", true)

	tr.RecordTestRun("auth", "logout")
	tr.RecordStep("auth", "logout")
	tr.RecordAssertion("auth", "logout", true)

	results := tr.Results()
	if len(results) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(results))
	}

	s := results[0]
	if s.Suite != "auth" {
		t.Errorf("expected suite name 'auth', got %q", s.Suite)
	}
	if s.TotalTests != 2 {
		t.Errorf("expected 2 total tests, got %d", s.TotalTests)
	}
	if s.RanTests != 2 {
		t.Errorf("expected 2 ran tests, got %d", s.RanTests)
	}
	if len(s.Tests) != 2 {
		t.Fatalf("expected 2 test records, got %d", len(s.Tests))
	}

	login := s.Tests[0]
	if login.Name != "login" {
		t.Errorf("expected test name 'login', got %q", login.Name)
	}
	if !login.Ran {
		t.Error("expected login to have ran")
	}
	if login.ExecutedSteps != 3 {
		t.Errorf("expected 3 executed steps, got %d", login.ExecutedSteps)
	}
	if login.PassedAssertions != 2 {
		t.Errorf("expected 2 passed assertions, got %d", login.PassedAssertions)
	}

	if !floatEqual(tr.TotalPercent(), 100) {
		t.Errorf("expected 100%% total, got %f", tr.TotalPercent())
	}
}

func TestTrackerMultipleSuites(t *testing.T) {
	tr := NewTracker()

	tr.RegisterSuite("suite-a", 2)
	tr.RegisterSuite("suite-b", 2)

	tr.RegisterTest("suite-a", "t1", 1, 1)
	tr.RegisterTest("suite-a", "t2", 1, 1)
	tr.RegisterTest("suite-b", "t3", 1, 1)
	tr.RegisterTest("suite-b", "t4", 1, 1)

	tr.RecordTestRun("suite-a", "t1")
	tr.RecordTestRun("suite-a", "t2")
	tr.RecordTestRun("suite-b", "t3")
	// t4 not run

	results := tr.Results()
	if len(results) != 2 {
		t.Fatalf("expected 2 suites, got %d", len(results))
	}

	if results[0].RanTests != 2 {
		t.Errorf("suite-a: expected 2 ran, got %d", results[0].RanTests)
	}
	if results[1].RanTests != 1 {
		t.Errorf("suite-b: expected 1 ran, got %d", results[1].RanTests)
	}

	// 3 out of 4 total = 75%
	if !floatEqual(tr.TotalPercent(), 75) {
		t.Errorf("expected 75%%, got %f", tr.TotalPercent())
	}
}

func TestTrackerSkippedTests(t *testing.T) {
	tr := NewTracker()
	tr.RegisterSuite("s", 3)
	tr.RegisterTest("s", "t1", 1, 1)
	tr.RegisterTest("s", "t2", 1, 1)
	tr.RegisterTest("s", "t3", 1, 1)

	tr.RecordTestRun("s", "t1")
	tr.RecordTestSkip("s", "t2")

	results := tr.Results()
	s := results[0]

	if s.RanTests != 1 {
		t.Errorf("expected 1 ran, got %d", s.RanTests)
	}
	if s.SkippedTests != 1 {
		t.Errorf("expected 1 skipped, got %d", s.SkippedTests)
	}

	if !s.Tests[1].Skipped {
		t.Error("expected t2 to be skipped")
	}
}

func TestTrackerConcurrentAccess(t *testing.T) {
	tr := NewTracker()
	tr.RegisterSuite("concurrent", 100)

	for i := range 100 {
		name := "test-" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		tr.RegisterTest("concurrent", name, 5, 5)
	}

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "test-" + string(rune('A'+idx%26)) + string(rune('0'+idx/26))
			tr.RecordTestRun("concurrent", name)
			for range 5 {
				tr.RecordStep("concurrent", name)
			}
			for range 5 {
				tr.RecordAssertion("concurrent", name, true)
			}
		}(i)
	}
	wg.Wait()

	if !floatEqual(tr.TotalPercent(), 100) {
		t.Errorf("expected 100%%, got %f", tr.TotalPercent())
	}

	results := tr.Results()
	if results[0].RanTests != 100 {
		t.Errorf("expected 100 ran tests, got %d", results[0].RanTests)
	}
}

func TestTrackerFailedAssertions(t *testing.T) {
	tr := NewTracker()
	tr.RegisterSuite("s", 1)
	tr.RegisterTest("s", "t1", 2, 4)

	tr.RecordTestRun("s", "t1")
	tr.RecordStep("s", "t1")
	tr.RecordStep("s", "t1")
	tr.RecordAssertion("s", "t1", true)
	tr.RecordAssertion("s", "t1", false)
	tr.RecordAssertion("s", "t1", true)
	tr.RecordAssertion("s", "t1", false)

	results := tr.Results()
	tc := results[0].Tests[0]
	if tc.PassedAssertions != 2 {
		t.Errorf("expected 2 passed, got %d", tc.PassedAssertions)
	}
	if tc.FailedAssertions != 2 {
		t.Errorf("expected 2 failed, got %d", tc.FailedAssertions)
	}
}

func TestTrackerUnregisteredSuiteDoesNotPanic(t *testing.T) {
	tr := NewTracker()

	// None of these should panic.
	tr.RegisterTest("nosuite", "t1", 1, 1)
	tr.RecordTestRun("nosuite", "t1")
	tr.RecordTestSkip("nosuite", "t1")
	tr.RecordStep("nosuite", "t1")
	tr.RecordAssertion("nosuite", "t1", true)
}

func TestTrackerUnregisteredTestDoesNotPanic(t *testing.T) {
	tr := NewTracker()
	tr.RegisterSuite("s", 1)

	// These should not panic despite "notest" not being registered.
	tr.RecordTestRun("s", "notest")
	tr.RecordTestSkip("s", "notest")
	tr.RecordStep("s", "notest")
	tr.RecordAssertion("s", "notest", true)
}

func TestTrackerTotalPercentEmpty(t *testing.T) {
	tr := NewTracker()
	if !floatEqual(tr.TotalPercent(), 0) {
		t.Errorf("expected 0 for empty tracker, got %f", tr.TotalPercent())
	}
}

func TestTrackerTotalPercentMixedResults(t *testing.T) {
	tr := NewTracker()
	tr.RegisterSuite("a", 4)
	tr.RegisterSuite("b", 6)

	tr.RegisterTest("a", "a1", 0, 0)
	tr.RegisterTest("b", "b1", 0, 0)

	tr.RecordTestRun("a", "a1")
	tr.RecordTestRun("b", "b1")

	// a: 1/4 ran, b: 1/6 ran => total 2/10 = 20%
	if !floatEqual(tr.TotalPercent(), 20) {
		t.Errorf("expected 20%%, got %f", tr.TotalPercent())
	}
}

func TestTrackerDuplicateRegistration(t *testing.T) {
	tr := NewTracker()
	tr.RegisterSuite("s", 2)
	tr.RegisterSuite("s", 5) // duplicate, should be ignored

	tr.RegisterTest("s", "t1", 1, 1)
	tr.RegisterTest("s", "t1", 9, 9) // duplicate, should be ignored

	results := tr.Results()
	if results[0].TotalTests != 2 {
		t.Errorf("expected original test count 2, got %d", results[0].TotalTests)
	}
	if results[0].Tests[0].TotalSteps != 1 {
		t.Errorf("expected original step count 1, got %d", results[0].Tests[0].TotalSteps)
	}
}

func TestTrackerResultsOrder(t *testing.T) {
	tr := NewTracker()
	tr.RegisterSuite("z-suite", 1)
	tr.RegisterSuite("a-suite", 1)
	tr.RegisterSuite("m-suite", 1)

	results := tr.Results()
	if results[0].Suite != "z-suite" || results[1].Suite != "a-suite" || results[2].Suite != "m-suite" {
		t.Errorf("results not in insertion order: %s, %s, %s",
			results[0].Suite, results[1].Suite, results[2].Suite)
	}
}
