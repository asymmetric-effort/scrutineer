package telemetry

import "testing"

func TestEventTypeValues(t *testing.T) {
	tests := []struct {
		et   EventType
		want uint16
	}{
		{SuiteStart, 0x01},
		{SuiteEnd, 0x02},
		{TestStart, 0x03},
		{TestPass, 0x04},
		{TestFail, 0x05},
		{TestSkip, 0x06},
		{StepStart, 0x07},
		{StepEnd, 0x08},
		{Assertion, 0x09},
		{Request, 0x0A},
		{Response, 0x0B},
		{Error, 0x0C},
		{ConnectorSetup, 0x0D},
		{ConnectorTeardown, 0x0E},
		{Metric, 0x0F},
	}
	for _, tt := range tests {
		if uint16(tt.et) != tt.want {
			t.Errorf("%s = 0x%02X, want 0x%02X", tt.et, uint16(tt.et), tt.want)
		}
	}
}

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		et   EventType
		want string
	}{
		{SuiteStart, "SuiteStart"},
		{SuiteEnd, "SuiteEnd"},
		{TestStart, "TestStart"},
		{TestPass, "TestPass"},
		{TestFail, "TestFail"},
		{TestSkip, "TestSkip"},
		{StepStart, "StepStart"},
		{StepEnd, "StepEnd"},
		{Assertion, "Assertion"},
		{Request, "Request"},
		{Response, "Response"},
		{Error, "Error"},
		{ConnectorSetup, "ConnectorSetup"},
		{ConnectorTeardown, "ConnectorTeardown"},
		{Metric, "Metric"},
	}
	for _, tt := range tests {
		got := tt.et.String()
		if got != tt.want {
			t.Errorf("EventType(0x%02X).String() = %q, want %q", uint16(tt.et), got, tt.want)
		}
	}
}

func TestEventTypeStringUnknown(t *testing.T) {
	unknown := EventType(0xFF)
	got := unknown.String()
	want := "Unknown(0xFF)"
	if got != want {
		t.Errorf("EventType(0xFF).String() = %q, want %q", got, want)
	}
}
