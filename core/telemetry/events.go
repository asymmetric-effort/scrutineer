package telemetry

import "fmt"

// EventType represents a telemetry event category.
type EventType uint16

const (
	// SuiteStart signals the beginning of a test suite.
	SuiteStart EventType = 0x01
	// SuiteEnd signals the end of a test suite.
	SuiteEnd EventType = 0x02
	// TestStart signals the beginning of an individual test.
	TestStart EventType = 0x03
	// TestPass signals a passing test.
	TestPass EventType = 0x04
	// TestFail signals a failing test.
	TestFail EventType = 0x05
	// TestSkip signals a skipped test.
	TestSkip EventType = 0x06
	// StepStart signals the beginning of a test step.
	StepStart EventType = 0x07
	// StepEnd signals the end of a test step.
	StepEnd EventType = 0x08
	// Assertion represents an assertion evaluation.
	Assertion EventType = 0x09
	// Request represents an outbound request.
	Request EventType = 0x0A
	// Response represents an inbound response.
	Response EventType = 0x0B
	// Error represents an error event.
	Error EventType = 0x0C
	// ConnectorSetup signals connector initialization.
	ConnectorSetup EventType = 0x0D
	// ConnectorTeardown signals connector cleanup.
	ConnectorTeardown EventType = 0x0E
	// Metric represents a recorded metric.
	Metric EventType = 0x0F
	// InteractionStart signals the beginning of an interaction group.
	InteractionStart EventType = 0x10
	// InteractionEnd signals the end of an interaction group.
	InteractionEnd EventType = 0x11
)

var eventNames = map[EventType]string{
	SuiteStart:        "SuiteStart",
	SuiteEnd:          "SuiteEnd",
	TestStart:         "TestStart",
	TestPass:          "TestPass",
	TestFail:          "TestFail",
	TestSkip:          "TestSkip",
	StepStart:         "StepStart",
	StepEnd:           "StepEnd",
	Assertion:         "Assertion",
	Request:           "Request",
	Response:          "Response",
	Error:             "Error",
	ConnectorSetup:    "ConnectorSetup",
	ConnectorTeardown: "ConnectorTeardown",
	Metric:            "Metric",
	InteractionStart:  "InteractionStart",
	InteractionEnd:    "InteractionEnd",
}

// String returns the human-readable name of the event type.
func (e EventType) String() string {
	if name, ok := eventNames[e]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(0x%02X)", uint16(e))
}
