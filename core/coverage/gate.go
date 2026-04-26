package coverage

import "fmt"

// Gate checks if coverage meets the required threshold.
type Gate struct {
	Threshold float64 // e.g. 98.0
}

// Check returns nil if coverage meets threshold, error otherwise.
// The error message clearly states actual vs required percentage.
func (g *Gate) Check(tracker *Tracker) error {
	actual := tracker.TotalPercent()
	if actual < g.Threshold {
		return fmt.Errorf("coverage %.1f%% is below required threshold %.1f%%", actual, g.Threshold)
	}
	return nil
}
