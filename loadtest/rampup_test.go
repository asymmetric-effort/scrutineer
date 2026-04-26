package loadtest

import (
	"testing"
	"time"
)

func TestSchedule_TenWorkers_OneSecondRampUp(t *testing.T) {
	steps := Schedule(10, time.Second)

	if len(steps) != 10 {
		t.Fatalf("expected 10 steps, got %d", len(steps))
	}

	// First step: 1 worker at time 0.
	if steps[0].At != 0 {
		t.Errorf("step 0: expected At=0, got %s", steps[0].At)
	}
	if steps[0].Workers != 1 {
		t.Errorf("step 0: expected 1 worker, got %d", steps[0].Workers)
	}

	// Last step: 10 workers at rampUp duration.
	if steps[9].At != time.Second {
		t.Errorf("step 9: expected At=1s, got %s", steps[9].At)
	}
	if steps[9].Workers != 10 {
		t.Errorf("step 9: expected 10 workers, got %d", steps[9].Workers)
	}

	// Verify monotonic increase.
	for i := 1; i < len(steps); i++ {
		if steps[i].Workers != steps[i-1].Workers+1 {
			t.Errorf("step %d: expected %d workers, got %d", i, steps[i-1].Workers+1, steps[i].Workers)
		}
		if steps[i].At <= steps[i-1].At {
			t.Errorf("step %d: At %s should be > step %d At %s", i, steps[i].At, i-1, steps[i-1].At)
		}
	}

	// Verify even spacing.
	expectedInterval := time.Second / 9 // 9 intervals for 10 steps
	for i := 1; i < len(steps); i++ {
		interval := steps[i].At - steps[i-1].At
		diff := interval - expectedInterval
		if diff < 0 {
			diff = -diff
		}
		if diff > time.Microsecond {
			t.Errorf("step %d: interval %s deviates from expected %s by %s", i, interval, expectedInterval, diff)
		}
	}
}

func TestSchedule_OneWorker(t *testing.T) {
	steps := Schedule(1, time.Second)

	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].At != 0 {
		t.Errorf("expected At=0, got %s", steps[0].At)
	}
	if steps[0].Workers != 1 {
		t.Errorf("expected 1 worker, got %d", steps[0].Workers)
	}
}

func TestSchedule_ZeroRampUp(t *testing.T) {
	steps := Schedule(5, 0)

	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].At != 0 {
		t.Errorf("expected At=0, got %s", steps[0].At)
	}
	if steps[0].Workers != 5 {
		t.Errorf("expected 5 workers, got %d", steps[0].Workers)
	}
}

func TestSchedule_ZeroConcurrency(t *testing.T) {
	steps := Schedule(0, time.Second)
	if steps != nil {
		t.Errorf("expected nil, got %v", steps)
	}
}

func TestSchedule_NegativeConcurrency(t *testing.T) {
	steps := Schedule(-1, time.Second)
	if steps != nil {
		t.Errorf("expected nil, got %v", steps)
	}
}

func TestSchedule_NegativeRampUp(t *testing.T) {
	steps := Schedule(5, -time.Second)

	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Workers != 5 {
		t.Errorf("expected 5 workers, got %d", steps[0].Workers)
	}
}

func TestSchedule_TwoWorkers(t *testing.T) {
	steps := Schedule(2, time.Second)

	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].At != 0 || steps[0].Workers != 1 {
		t.Errorf("step 0: unexpected %+v", steps[0])
	}
	if steps[1].At != time.Second || steps[1].Workers != 2 {
		t.Errorf("step 1: unexpected %+v", steps[1])
	}
}

func TestSchedule_LargeCount(t *testing.T) {
	steps := Schedule(100, 10*time.Second)

	if len(steps) != 100 {
		t.Fatalf("expected 100 steps, got %d", len(steps))
	}
	if steps[0].Workers != 1 {
		t.Errorf("first step should have 1 worker, got %d", steps[0].Workers)
	}
	if steps[99].Workers != 100 {
		t.Errorf("last step should have 100 workers, got %d", steps[99].Workers)
	}
	if steps[99].At != 10*time.Second {
		t.Errorf("last step at %s, expected 10s", steps[99].At)
	}
}
