package coverage

import (
	"strings"
	"testing"
)

func setupTracker(ranCount, totalCount int) *Tracker {
	tr := NewTracker()
	tr.RegisterSuite("s", totalCount)
	for i := range totalCount {
		name := "t" + string(rune('0'+i))
		tr.RegisterTest("s", name, 0, 0)
		if i < ranCount {
			tr.RecordTestRun("s", name)
		}
	}
	return tr
}

func TestGateCheckPasses(t *testing.T) {
	tr := setupTracker(9, 10) // 90%
	g := &Gate{Threshold: 80.0}

	if err := g.Check(tr); err != nil {
		t.Errorf("expected pass, got error: %v", err)
	}
}

func TestGateCheckFails(t *testing.T) {
	tr := setupTracker(5, 10) // 50%
	g := &Gate{Threshold: 80.0}

	err := g.Check(tr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "50.0%") {
		t.Errorf("expected actual percentage in error, got: %s", msg)
	}
	if !strings.Contains(msg, "80.0%") {
		t.Errorf("expected threshold in error, got: %s", msg)
	}
}

func TestGateCheck100Threshold100Coverage(t *testing.T) {
	tr := setupTracker(5, 5) // 100%
	g := &Gate{Threshold: 100.0}

	if err := g.Check(tr); err != nil {
		t.Errorf("expected pass with 100/100, got: %v", err)
	}
}

func TestGateCheckZeroThreshold(t *testing.T) {
	tr := setupTracker(0, 10) // 0%
	g := &Gate{Threshold: 0.0}

	if err := g.Check(tr); err != nil {
		t.Errorf("expected pass with 0 threshold, got: %v", err)
	}
}

func TestGateCheckExactThreshold(t *testing.T) {
	tr := setupTracker(8, 10) // 80%
	g := &Gate{Threshold: 80.0}

	if err := g.Check(tr); err != nil {
		t.Errorf("expected pass when exactly at threshold, got: %v", err)
	}
}

func TestGateCheckJustBelowThreshold(t *testing.T) {
	tr := setupTracker(7, 10) // 70%
	g := &Gate{Threshold: 80.0}

	err := g.Check(tr)
	if err == nil {
		t.Fatal("expected error for 70% < 80% threshold")
	}
}

func TestGateCheckEmptyTracker(t *testing.T) {
	tr := NewTracker()
	g := &Gate{Threshold: 0.0}

	if err := g.Check(tr); err != nil {
		t.Errorf("expected pass for empty tracker with 0 threshold, got: %v", err)
	}
}

func TestGateCheckEmptyTrackerWithThreshold(t *testing.T) {
	tr := NewTracker()
	g := &Gate{Threshold: 50.0}

	err := g.Check(tr)
	if err == nil {
		t.Fatal("expected error for empty tracker with 50% threshold")
	}
}
