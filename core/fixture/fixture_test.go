package fixture

import (
	"os"
	"strings"
	"testing"
)

func TestNewStore_WithFixtures(t *testing.T) {
	fixtures := map[string]any{
		"user": map[string]any{"name": "alice", "age": 30},
	}
	s := NewStore(fixtures)
	if s.fixtures == nil {
		t.Fatal("expected fixtures to be set")
	}
	if s.captures == nil {
		t.Fatal("expected captures to be initialized")
	}
}

func TestNewStore_EmptyFixtures(t *testing.T) {
	s := NewStore(map[string]any{})
	if len(s.fixtures) != 0 {
		t.Fatal("expected empty fixtures map")
	}
}

func TestNewStore_NilFixtures(t *testing.T) {
	s := NewStore(nil)
	if s.fixtures == nil {
		t.Fatal("expected non-nil fixtures map when nil is passed")
	}
	if len(s.fixtures) != 0 {
		t.Fatal("expected empty fixtures map")
	}
}

func TestSetCapture_And_GetCapture(t *testing.T) {
	s := NewStore(nil)
	s.SetCapture("user_id", 42)

	val, ok := s.GetCapture("user_id")
	if !ok {
		t.Fatal("expected capture to be found")
	}
	if val != 42 {
		t.Fatalf("expected 42, got %v", val)
	}
}

func TestGetCapture_Missing(t *testing.T) {
	s := NewStore(nil)
	_, ok := s.GetCapture("nonexistent")
	if ok {
		t.Fatal("expected capture not to be found")
	}
}

func TestSetCapture_Overwrite(t *testing.T) {
	s := NewStore(nil)
	s.SetCapture("key", "first")
	s.SetCapture("key", "second")

	val, ok := s.GetCapture("key")
	if !ok {
		t.Fatal("expected capture to be found")
	}
	if val != "second" {
		t.Fatalf("expected 'second', got %v", val)
	}
}

func TestResolve_FixtureSimple(t *testing.T) {
	s := NewStore(map[string]any{"greeting": "hello"})
	val, ok := s.Resolve("fixture.greeting")
	if !ok {
		t.Fatal("expected to resolve fixture.greeting")
	}
	if val != "hello" {
		t.Fatalf("expected 'hello', got %v", val)
	}
}

func TestResolve_FixtureNested(t *testing.T) {
	s := NewStore(map[string]any{
		"user": map[string]any{"name": "bob"},
	})
	val, ok := s.Resolve("fixture.user.name")
	if !ok {
		t.Fatal("expected to resolve fixture.user.name")
	}
	if val != "bob" {
		t.Fatalf("expected 'bob', got %v", val)
	}
}

func TestResolve_FixtureDeepNesting(t *testing.T) {
	s := NewStore(map[string]any{
		"db": map[string]any{
			"primary": map[string]any{
				"host": "localhost",
			},
		},
	})
	val, ok := s.Resolve("fixture.db.primary.host")
	if !ok {
		t.Fatal("expected to resolve deep fixture path")
	}
	if val != "localhost" {
		t.Fatalf("expected 'localhost', got %v", val)
	}
}

func TestResolve_Capture(t *testing.T) {
	s := NewStore(nil)
	s.SetCapture("user_id", "abc123")

	val, ok := s.Resolve("capture.user_id")
	if !ok {
		t.Fatal("expected to resolve capture.user_id")
	}
	if val != "abc123" {
		t.Fatalf("expected 'abc123', got %v", val)
	}
}

func TestResolve_CaptureNested(t *testing.T) {
	s := NewStore(nil)
	s.SetCapture("response", map[string]any{"id": 99})

	val, ok := s.Resolve("capture.response.id")
	if !ok {
		t.Fatal("expected to resolve capture.response.id")
	}
	if val != 99 {
		t.Fatalf("expected 99, got %v", val)
	}
}

func TestResolve_Env(t *testing.T) {
	t.Setenv("SCRUTINEER_TEST_VAR", "test_value")

	s := NewStore(nil)
	val, ok := s.Resolve("env.SCRUTINEER_TEST_VAR")
	if !ok {
		t.Fatal("expected to resolve env var")
	}
	if val != "test_value" {
		t.Fatalf("expected 'test_value', got %v", val)
	}
}

func TestResolve_EnvEmpty(t *testing.T) {
	t.Setenv("SCRUTINEER_EMPTY_VAR", "")

	s := NewStore(nil)
	val, ok := s.Resolve("env.SCRUTINEER_EMPTY_VAR")
	if !ok {
		t.Fatal("expected to resolve empty env var")
	}
	if val != "" {
		t.Fatalf("expected empty string, got %v", val)
	}
}

func TestResolve_EnvMissing(t *testing.T) {
	os.Unsetenv("SCRUTINEER_NONEXISTENT_VAR")

	s := NewStore(nil)
	_, ok := s.Resolve("env.SCRUTINEER_NONEXISTENT_VAR")
	if ok {
		t.Fatal("expected env var not to be found")
	}
}

func TestResolve_UnknownPrefix(t *testing.T) {
	s := NewStore(nil)
	_, ok := s.Resolve("unknown.key")
	if ok {
		t.Fatal("expected unknown prefix to return false")
	}
}

func TestResolve_NoDot(t *testing.T) {
	s := NewStore(nil)
	_, ok := s.Resolve("nodot")
	if ok {
		t.Fatal("expected ref without dot to return false")
	}
}

func TestResolve_MissingKey(t *testing.T) {
	s := NewStore(map[string]any{"user": "alice"})
	_, ok := s.Resolve("fixture.nonexistent")
	if ok {
		t.Fatal("expected missing key to return false")
	}
}

func TestResolve_NonMapIntermediate(t *testing.T) {
	s := NewStore(map[string]any{"user": "alice"})
	_, ok := s.Resolve("fixture.user.name")
	if ok {
		t.Fatal("expected non-map intermediate to return false")
	}
}

func TestInterpolate_SimpleVar(t *testing.T) {
	s := NewStore(map[string]any{"x": "world"})
	result, err := s.Interpolate("hello ${fixture.x}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Fatalf("expected 'hello world', got %q", result)
	}
}

func TestInterpolate_MultipleVars(t *testing.T) {
	s := NewStore(map[string]any{"first": "John", "last": "Doe"})
	result, err := s.Interpolate("${fixture.first} ${fixture.last}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "John Doe" {
		t.Fatalf("expected 'John Doe', got %q", result)
	}
}

func TestInterpolate_NoVars(t *testing.T) {
	s := NewStore(nil)
	result, err := s.Interpolate("plain text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "plain text" {
		t.Fatalf("expected 'plain text', got %q", result)
	}
}

func TestInterpolate_AdjacentVars(t *testing.T) {
	s := NewStore(map[string]any{"a": "X", "b": "Y"})
	result, err := s.Interpolate("${fixture.a}${fixture.b}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "XY" {
		t.Fatalf("expected 'XY', got %q", result)
	}
}

func TestInterpolate_UnresolvedVar(t *testing.T) {
	s := NewStore(nil)
	_, err := s.Interpolate("hello ${fixture.missing}")
	if err == nil {
		t.Fatal("expected error for unresolved variable")
	}
	if !strings.Contains(err.Error(), "unresolved variable") {
		t.Fatalf("expected 'unresolved variable' in error, got %v", err)
	}
}

func TestInterpolate_EscapedVar(t *testing.T) {
	s := NewStore(nil)
	result, err := s.Interpolate(`hello \${fixture.x}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello ${fixture.x}" {
		t.Fatalf("expected literal '${fixture.x}', got %q", result)
	}
}

func TestInterpolate_UnclosedBrace(t *testing.T) {
	s := NewStore(nil)
	result, err := s.Interpolate("hello ${unclosed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello ${unclosed" {
		t.Fatalf("expected literal pass-through, got %q", result)
	}
}

func TestInterpolate_NonStringValue(t *testing.T) {
	s := NewStore(map[string]any{"count": 42})
	result, err := s.Interpolate("count is ${fixture.count}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "count is 42" {
		t.Fatalf("expected 'count is 42', got %q", result)
	}
}

func TestInterpolateMap_NestedMaps(t *testing.T) {
	s := NewStore(map[string]any{"host": "example.com"})
	input := map[string]any{
		"url": "https://${fixture.host}/api",
		"nested": map[string]any{
			"header": "Host: ${fixture.host}",
		},
	}

	result, err := s.InterpolateMap(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["url"] != "https://example.com/api" {
		t.Fatalf("expected interpolated url, got %v", result["url"])
	}

	nested, ok := result["nested"].(map[string]any)
	if !ok {
		t.Fatal("expected nested map")
	}
	if nested["header"] != "Host: example.com" {
		t.Fatalf("expected interpolated header, got %v", nested["header"])
	}
}

func TestInterpolateMap_MixedTypes(t *testing.T) {
	s := NewStore(map[string]any{"name": "test"})
	input := map[string]any{
		"str":   "hello ${fixture.name}",
		"num":   42,
		"flag":  true,
		"null":  nil,
		"slice": []any{"item1", "${fixture.name}"},
	}

	result, err := s.InterpolateMap(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["str"] != "hello test" {
		t.Fatalf("expected interpolated string, got %v", result["str"])
	}
	if result["num"] != 42 {
		t.Fatalf("expected 42 pass-through, got %v", result["num"])
	}
	if result["flag"] != true {
		t.Fatalf("expected true pass-through, got %v", result["flag"])
	}
	if result["null"] != nil {
		t.Fatalf("expected nil pass-through, got %v", result["null"])
	}

	sl, ok := result["slice"].([]any)
	if !ok {
		t.Fatal("expected slice")
	}
	if sl[0] != "item1" {
		t.Fatalf("expected 'item1', got %v", sl[0])
	}
	if sl[1] != "test" {
		t.Fatalf("expected 'test', got %v", sl[1])
	}
}

func TestInterpolateMap_Error(t *testing.T) {
	s := NewStore(nil)
	input := map[string]any{
		"bad": "${fixture.missing}",
	}

	_, err := s.InterpolateMap(input)
	if err == nil {
		t.Fatal("expected error for unresolved variable in map")
	}
}

func TestInterpolateMap_ErrorInNestedMap(t *testing.T) {
	s := NewStore(nil)
	input := map[string]any{
		"nested": map[string]any{
			"bad": "${fixture.missing}",
		},
	}

	_, err := s.InterpolateMap(input)
	if err == nil {
		t.Fatal("expected error for unresolved variable in nested map")
	}
}

func TestInterpolateMap_ErrorInSlice(t *testing.T) {
	s := NewStore(nil)
	input := map[string]any{
		"items": []any{"${fixture.missing}"},
	}

	_, err := s.InterpolateMap(input)
	if err == nil {
		t.Fatal("expected error for unresolved variable in slice")
	}
}
