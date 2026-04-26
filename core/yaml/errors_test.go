package yaml

import (
	"strings"
	"testing"
)

func TestParseError_Error(t *testing.T) {
	e := &ParseError{Line: 3, Column: 5, Message: "unexpected character"}
	got := e.Error()
	if got != "yaml: line 3, column 5: unexpected character" {
		t.Errorf("unexpected error string: %s", got)
	}
}

func TestNewParseError(t *testing.T) {
	e := newParseError(1, 2, "test message")
	if e.Line != 1 || e.Column != 2 || e.Message != "test message" {
		t.Errorf("unexpected error: %+v", e)
	}
}

func TestNewParseErrorf(t *testing.T) {
	e := newParseErrorf(10, 20, "got %q expected %s", "foo", "bar")
	if e.Line != 10 || e.Column != 20 {
		t.Errorf("unexpected line/col: %d/%d", e.Line, e.Column)
	}
	if !strings.Contains(e.Message, "foo") || !strings.Contains(e.Message, "bar") {
		t.Errorf("unexpected message: %s", e.Message)
	}
}

func TestParseError_ImplementsError(t *testing.T) {
	var err error = &ParseError{Line: 1, Column: 1, Message: "test"}
	if err.Error() == "" {
		t.Error("error should not be empty")
	}
}
