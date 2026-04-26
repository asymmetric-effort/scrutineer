package cli

import (
	"testing"
)

func TestParseShellArgsSimple(t *testing.T) {
	args, err := parseShellArgs("echo hello world")
	if err != nil {
		t.Fatalf("parseShellArgs() error = %v", err)
	}
	if len(args) != 3 || args[0] != "echo" || args[1] != "hello" || args[2] != "world" {
		t.Errorf("args = %v, want [echo hello world]", args)
	}
}

func TestParseShellArgsDoubleQuotes(t *testing.T) {
	args, err := parseShellArgs(`echo "hello world"`)
	if err != nil {
		t.Fatalf("parseShellArgs() error = %v", err)
	}
	if len(args) != 2 || args[1] != "hello world" {
		t.Errorf("args = %v, want [echo, hello world]", args)
	}
}

func TestParseShellArgsSingleQuotes(t *testing.T) {
	args, err := parseShellArgs("echo 'hello world'")
	if err != nil {
		t.Fatalf("parseShellArgs() error = %v", err)
	}
	if len(args) != 2 || args[1] != "hello world" {
		t.Errorf("args = %v, want [echo, hello world]", args)
	}
}

func TestParseShellArgsBackslashInDoubleQuotes(t *testing.T) {
	args, err := parseShellArgs(`echo "hello\"world"`)
	if err != nil {
		t.Fatalf("parseShellArgs() error = %v", err)
	}
	if len(args) != 2 || args[1] != `hello"world` {
		t.Errorf("args = %v, want [echo, hello\"world]", args)
	}
}

func TestParseShellArgsUnterminatedSingleQuote(t *testing.T) {
	_, err := parseShellArgs("echo 'unterminated")
	if err == nil {
		t.Fatal("expected error for unterminated single quote")
	}
}

func TestParseShellArgsUnterminatedDoubleQuote(t *testing.T) {
	_, err := parseShellArgs(`echo "unterminated`)
	if err == nil {
		t.Fatal("expected error for unterminated double quote")
	}
}

func TestParseShellArgsEmpty(t *testing.T) {
	args, err := parseShellArgs("")
	if err != nil {
		t.Fatalf("parseShellArgs() error = %v", err)
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want empty", args)
	}
}

func TestParseShellArgsTabs(t *testing.T) {
	args, err := parseShellArgs("echo\thello")
	if err != nil {
		t.Fatalf("parseShellArgs() error = %v", err)
	}
	if len(args) != 2 || args[0] != "echo" || args[1] != "hello" {
		t.Errorf("args = %v, want [echo hello]", args)
	}
}

func TestParamStringMissing(t *testing.T) {
	_, ok, err := paramString(map[string]any{}, "key")
	if err != nil {
		t.Fatalf("paramString() error = %v", err)
	}
	if ok {
		t.Error("ok = true, want false")
	}
}

func TestParamStringWrongType(t *testing.T) {
	_, _, err := paramString(map[string]any{"key": 123}, "key")
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestParamBoolMissing(t *testing.T) {
	_, ok, err := paramBool(map[string]any{}, "key")
	if err != nil {
		t.Fatalf("paramBool() error = %v", err)
	}
	if ok {
		t.Error("ok = true, want false")
	}
}

func TestParamBoolWrongType(t *testing.T) {
	_, _, err := paramBool(map[string]any{"key": "yes"}, "key")
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestParamMapMissing(t *testing.T) {
	_, ok, err := paramMap(map[string]any{}, "key")
	if err != nil {
		t.Fatalf("paramMap() error = %v", err)
	}
	if ok {
		t.Error("ok = true, want false")
	}
}

func TestParamMapWrongType(t *testing.T) {
	_, _, err := paramMap(map[string]any{"key": "not-a-map"}, "key")
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}
