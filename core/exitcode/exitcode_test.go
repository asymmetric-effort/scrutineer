package exitcode

import "testing"

func TestConstants(t *testing.T) {
	if OK != 0 {
		t.Errorf("OK = %d, want 0", OK)
	}
	if TestFailure != 1 {
		t.Errorf("TestFailure = %d, want 1", TestFailure)
	}
	if ConnectionError != 2 {
		t.Errorf("ConnectionError = %d, want 2", ConnectionError)
	}
	if ConfigError != 3 {
		t.Errorf("ConfigError = %d, want 3", ConfigError)
	}
	if InternalError != 4 {
		t.Errorf("InternalError = %d, want 4", InternalError)
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{OK, "all tests passed"},
		{TestFailure, "one or more test assertions failed"},
		{ConnectionError, "connection or network error"},
		{ConfigError, "configuration or YAML parse error"},
		{InternalError, "framework or internal error"},
		{99, "unknown exit code"},
		{-1, "unknown exit code"},
	}

	for _, tt := range tests {
		got := String(tt.code)
		if got != tt.want {
			t.Errorf("String(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
