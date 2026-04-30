// Package engine orchestrates test suite execution, connecting schema
// definitions to connectors, reporters, telemetry, and coverage tracking.
package engine

import "github.com/scrutineer/scrutineer/core/fixture"

// TestContext carries state for a single test execution.
type TestContext struct {
	Store       *fixture.Store // fixture + capture variable store
	Suite       string         // current suite name
	Test        string         // current test name
	Interaction string         // current interaction name (empty for simple suites)
	PassNum     int            // current pass number (0 for simple suites)
}

// NewTestContext creates a TestContext with the given suite and test names,
// initialising a fixture Store from the provided fixtures map.
func NewTestContext(suite, test string, fixtures map[string]any) *TestContext {
	return &TestContext{
		Store: fixture.NewStore(fixtures),
		Suite: suite,
		Test:  test,
	}
}
