package telemetry

import (
	"testing"
	"time"
)

func TestNowNano(t *testing.T) {
	before := time.Now().UnixNano()
	got := NowNano()
	after := time.Now().UnixNano()

	if got < before || got > after {
		t.Errorf("NowNano() = %d, want between %d and %d", got, before, after)
	}
}

func TestNowNanoMonotonic(t *testing.T) {
	a := NowNano()
	b := NowNano()
	if b < a {
		t.Errorf("NowNano() not monotonic: %d >= %d", a, b)
	}
}
