package connector

import (
	"testing"
)

func TestGetStringPresent(t *testing.T) {
	r := &Result{Data: map[string]any{"body": "hello"}}
	v, ok := r.GetString("body")
	if !ok || v != "hello" {
		t.Fatalf("expected (\"hello\", true), got (%q, %v)", v, ok)
	}
}

func TestGetStringMissing(t *testing.T) {
	r := &Result{Data: map[string]any{}}
	v, ok := r.GetString("missing")
	if ok || v != "" {
		t.Fatalf("expected (\"\", false), got (%q, %v)", v, ok)
	}
}

func TestGetStringWrongType(t *testing.T) {
	r := &Result{Data: map[string]any{"num": 42}}
	v, ok := r.GetString("num")
	if ok || v != "" {
		t.Fatalf("expected (\"\", false), got (%q, %v)", v, ok)
	}
}

func TestGetStringNilData(t *testing.T) {
	r := &Result{}
	v, ok := r.GetString("any")
	if ok || v != "" {
		t.Fatalf("expected (\"\", false), got (%q, %v)", v, ok)
	}
}

func TestGetIntPresent(t *testing.T) {
	r := &Result{Data: map[string]any{"code": 200}}
	v, ok := r.GetInt("code")
	if !ok || v != 200 {
		t.Fatalf("expected (200, true), got (%d, %v)", v, ok)
	}
}

func TestGetIntFromInt64(t *testing.T) {
	r := &Result{Data: map[string]any{"big": int64(9999)}}
	v, ok := r.GetInt("big")
	if !ok || v != 9999 {
		t.Fatalf("expected (9999, true), got (%d, %v)", v, ok)
	}
}

func TestGetIntFromFloat64(t *testing.T) {
	r := &Result{Data: map[string]any{"float": 42.0}}
	v, ok := r.GetInt("float")
	if !ok || v != 42 {
		t.Fatalf("expected (42, true), got (%d, %v)", v, ok)
	}
}

func TestGetIntMissing(t *testing.T) {
	r := &Result{Data: map[string]any{}}
	v, ok := r.GetInt("missing")
	if ok || v != 0 {
		t.Fatalf("expected (0, false), got (%d, %v)", v, ok)
	}
}

func TestGetIntWrongType(t *testing.T) {
	r := &Result{Data: map[string]any{"str": "hello"}}
	v, ok := r.GetInt("str")
	if ok || v != 0 {
		t.Fatalf("expected (0, false), got (%d, %v)", v, ok)
	}
}

func TestGetIntNilData(t *testing.T) {
	r := &Result{}
	v, ok := r.GetInt("any")
	if ok || v != 0 {
		t.Fatalf("expected (0, false), got (%d, %v)", v, ok)
	}
}

func TestGetFloatPresent(t *testing.T) {
	r := &Result{Data: map[string]any{"latency": 1.5}}
	v, ok := r.GetFloat("latency")
	if !ok || v != 1.5 {
		t.Fatalf("expected (1.5, true), got (%f, %v)", v, ok)
	}
}

func TestGetFloatFromInt(t *testing.T) {
	r := &Result{Data: map[string]any{"count": 10}}
	v, ok := r.GetFloat("count")
	if !ok || v != 10.0 {
		t.Fatalf("expected (10.0, true), got (%f, %v)", v, ok)
	}
}

func TestGetFloatMissing(t *testing.T) {
	r := &Result{Data: map[string]any{}}
	v, ok := r.GetFloat("missing")
	if ok || v != 0 {
		t.Fatalf("expected (0, false), got (%f, %v)", v, ok)
	}
}

func TestGetFloatWrongType(t *testing.T) {
	r := &Result{Data: map[string]any{"str": "hello"}}
	v, ok := r.GetFloat("str")
	if ok || v != 0 {
		t.Fatalf("expected (0, false), got (%f, %v)", v, ok)
	}
}

func TestGetFloatNilData(t *testing.T) {
	r := &Result{}
	v, ok := r.GetFloat("any")
	if ok || v != 0 {
		t.Fatalf("expected (0, false), got (%f, %v)", v, ok)
	}
}

func TestGetBoolPresent(t *testing.T) {
	r := &Result{Data: map[string]any{"ok": true}}
	v, ok := r.GetBool("ok")
	if !ok || v != true {
		t.Fatalf("expected (true, true), got (%v, %v)", v, ok)
	}
}

func TestGetBoolFalse(t *testing.T) {
	r := &Result{Data: map[string]any{"ok": false}}
	v, ok := r.GetBool("ok")
	if !ok || v != false {
		t.Fatalf("expected (false, true), got (%v, %v)", v, ok)
	}
}

func TestGetBoolMissing(t *testing.T) {
	r := &Result{Data: map[string]any{}}
	v, ok := r.GetBool("missing")
	if ok || v != false {
		t.Fatalf("expected (false, false), got (%v, %v)", v, ok)
	}
}

func TestGetBoolWrongType(t *testing.T) {
	r := &Result{Data: map[string]any{"num": 1}}
	v, ok := r.GetBool("num")
	if ok || v != false {
		t.Fatalf("expected (false, false), got (%v, %v)", v, ok)
	}
}

func TestGetBoolNilData(t *testing.T) {
	r := &Result{}
	v, ok := r.GetBool("any")
	if ok || v != false {
		t.Fatalf("expected (false, false), got (%v, %v)", v, ok)
	}
}

func TestGetMapPresent(t *testing.T) {
	inner := map[string]any{"nested": "value"}
	r := &Result{Data: map[string]any{"headers": inner}}
	v, ok := r.GetMap("headers")
	if !ok {
		t.Fatal("expected ok=true for GetMap")
	}
	if v["nested"] != "value" {
		t.Fatalf("expected nested value 'value', got %v", v["nested"])
	}
}

func TestGetMapMissing(t *testing.T) {
	r := &Result{Data: map[string]any{}}
	v, ok := r.GetMap("missing")
	if ok || v != nil {
		t.Fatalf("expected (nil, false), got (%v, %v)", v, ok)
	}
}

func TestGetMapWrongType(t *testing.T) {
	r := &Result{Data: map[string]any{"str": "hello"}}
	v, ok := r.GetMap("str")
	if ok || v != nil {
		t.Fatalf("expected (nil, false), got (%v, %v)", v, ok)
	}
}

func TestGetMapNilData(t *testing.T) {
	r := &Result{}
	v, ok := r.GetMap("any")
	if ok || v != nil {
		t.Fatalf("expected (nil, false), got (%v, %v)", v, ok)
	}
}
