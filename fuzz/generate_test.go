package fuzz

import (
	"testing"
)

func TestNewGenerator(t *testing.T) {
	target := validTarget()
	g := NewGenerator(target, 42)
	if g == nil {
		t.Fatal("expected non-nil generator")
	}
	if g.target != target {
		t.Error("target mismatch")
	}
}

func TestGenerator_NextProducesOutput(t *testing.T) {
	target := validTarget()
	g := NewGenerator(target, 42)

	result := g.Next()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if _, ok := result["field1"]; !ok {
		t.Error("expected field1 in result")
	}
	if _, ok := result["field2"]; !ok {
		t.Error("expected field2 in result")
	}
}

func TestGenerator_NonFuzzFieldsUnchanged(t *testing.T) {
	target := validTarget()
	// Only field1 is fuzzed, field2 should remain 42.
	g := NewGenerator(target, 42)

	for i := 0; i < 50; i++ {
		result := g.Next()
		if result["field2"] != 42 {
			t.Fatalf("non-fuzz field changed: field2=%v", result["field2"])
		}
	}
}

func TestGenerator_FuzzFieldsMutated(t *testing.T) {
	target := validTarget()
	g := NewGenerator(target, 42)

	different := false
	for i := 0; i < 100; i++ {
		result := g.Next()
		if result["field1"] != "hello" {
			different = true
			break
		}
	}
	if !different {
		t.Error("expected at least one mutation to field1 over 100 iterations")
	}
}

func TestGenerator_StringMutation(t *testing.T) {
	target := &Target{
		Name:       "str-test",
		Connector:  "mock",
		Action:     "test",
		Parameters: map[string]any{"s": "abcdef"},
		FuzzFields: []string{"s"},
	}
	g := NewGenerator(target, 99)

	seen := make(map[string]bool)
	for i := 0; i < 200; i++ {
		result := g.Next()
		s, ok := result["s"].(string)
		if !ok {
			t.Fatalf("expected string, got %T", result["s"])
		}
		seen[s] = true
	}
	if len(seen) < 5 {
		t.Errorf("expected diverse string mutations, got only %d unique values", len(seen))
	}
}

func TestGenerator_IntMutation(t *testing.T) {
	target := &Target{
		Name:       "int-test",
		Connector:  "mock",
		Action:     "test",
		Parameters: map[string]any{"n": 100},
		FuzzFields: []string{"n"},
	}
	g := NewGenerator(target, 99)

	seen := make(map[int]bool)
	for i := 0; i < 200; i++ {
		result := g.Next()
		n, ok := result["n"].(int)
		if !ok {
			t.Fatalf("expected int, got %T", result["n"])
		}
		seen[n] = true
	}
	if len(seen) < 5 {
		t.Errorf("expected diverse int mutations, got only %d unique values", len(seen))
	}
}

func TestGenerator_BoolMutation(t *testing.T) {
	target := &Target{
		Name:       "bool-test",
		Connector:  "mock",
		Action:     "test",
		Parameters: map[string]any{"b": true},
		FuzzFields: []string{"b"},
	}
	g := NewGenerator(target, 42)

	result := g.Next()
	b, ok := result["b"].(bool)
	if !ok {
		t.Fatalf("expected bool, got %T", result["b"])
	}
	if b != false {
		t.Error("expected bool to flip from true to false")
	}
}

func TestGenerator_Float64Mutation(t *testing.T) {
	target := &Target{
		Name:       "float-test",
		Connector:  "mock",
		Action:     "test",
		Parameters: map[string]any{"f": 3.14},
		FuzzFields: []string{"f"},
	}
	g := NewGenerator(target, 99)

	seen := make(map[float64]bool)
	for i := 0; i < 200; i++ {
		result := g.Next()
		f, ok := result["f"].(float64)
		if !ok {
			t.Fatalf("expected float64, got %T", result["f"])
		}
		seen[f] = true
	}
	if len(seen) < 5 {
		t.Errorf("expected diverse float mutations, got only %d unique values", len(seen))
	}
}

func TestGenerator_NilMutation(t *testing.T) {
	target := &Target{
		Name:       "nil-test",
		Connector:  "mock",
		Action:     "test",
		Parameters: map[string]any{"x": nil},
		FuzzFields: []string{"x"},
	}
	g := NewGenerator(target, 42)

	result := g.Next()
	if result["x"] == nil {
		t.Error("expected nil to be mutated to some value")
	}
}

func TestGenerator_Int64Mutation(t *testing.T) {
	target := &Target{
		Name:       "int64-test",
		Connector:  "mock",
		Action:     "test",
		Parameters: map[string]any{"n": int64(100)},
		FuzzFields: []string{"n"},
	}
	g := NewGenerator(target, 99)

	seen := make(map[int64]bool)
	for i := 0; i < 200; i++ {
		result := g.Next()
		n, ok := result["n"].(int64)
		if !ok {
			t.Fatalf("expected int64, got %T", result["n"])
		}
		seen[n] = true
	}
	if len(seen) < 3 {
		t.Errorf("expected diverse int64 mutations, got only %d unique values", len(seen))
	}
}

func TestGenerator_UnknownTypeMutation(t *testing.T) {
	type custom struct{ val int }
	target := &Target{
		Name:       "unknown-test",
		Connector:  "mock",
		Action:     "test",
		Parameters: map[string]any{"x": custom{42}},
		FuzzFields: []string{"x"},
	}
	g := NewGenerator(target, 42)

	result := g.Next()
	v, ok := result["x"].(custom)
	if !ok {
		t.Fatalf("expected custom type, got %T", result["x"])
	}
	if v.val != 42 {
		t.Error("unknown type should be returned as-is")
	}
}

func TestGenerator_DeterministicWithSameSeed(t *testing.T) {
	target := validTarget()
	g1 := NewGenerator(target, 12345)
	g2 := NewGenerator(target, 12345)

	for i := 0; i < 50; i++ {
		r1 := g1.Next()
		r2 := g2.Next()
		// field1 is the fuzzed field.
		if r1["field1"] != r2["field1"] {
			t.Fatalf("iteration %d: same seed produced different results: %v vs %v", i, r1["field1"], r2["field1"])
		}
	}
}

func TestGenerator_DifferentWithDifferentSeeds(t *testing.T) {
	target := validTarget()
	g1 := NewGenerator(target, 111)
	g2 := NewGenerator(target, 222)

	different := false
	for i := 0; i < 50; i++ {
		r1 := g1.Next()
		r2 := g2.Next()
		if r1["field1"] != r2["field1"] {
			different = true
			break
		}
	}
	if !different {
		t.Error("different seeds should produce different outputs")
	}
}

func TestGenerator_EmptyStringMutation(t *testing.T) {
	target := &Target{
		Name:       "empty-str-test",
		Connector:  "mock",
		Action:     "test",
		Parameters: map[string]any{"s": ""},
		FuzzFields: []string{"s"},
	}
	g := NewGenerator(target, 42)

	result := g.Next()
	s, ok := result["s"].(string)
	if !ok {
		t.Fatalf("expected string, got %T", result["s"])
	}
	if s == "" {
		// Empty string should mutate to non-empty.
		t.Error("expected non-empty string from empty string mutation")
	}
}

func TestGenerator_MultipleFuzzFields(t *testing.T) {
	target := &Target{
		Name:       "multi-fuzz",
		Connector:  "mock",
		Action:     "test",
		Parameters: map[string]any{"a": "hello", "b": 42, "c": "unchanged"},
		FuzzFields: []string{"a", "b"},
	}
	g := NewGenerator(target, 42)

	result := g.Next()
	if result["c"] != "unchanged" {
		t.Error("non-fuzz field c should not change")
	}
	// Both a and b should exist.
	if _, ok := result["a"]; !ok {
		t.Error("expected a in result")
	}
	if _, ok := result["b"]; !ok {
		t.Error("expected b in result")
	}
}

func TestGenerator_NilMutationAllBranches(t *testing.T) {
	target := &Target{
		Name:       "nil-branches",
		Connector:  "mock",
		Action:     "test",
		Parameters: map[string]any{"x": nil},
		FuzzFields: []string{"x"},
	}

	seenString := false
	seenInt := false
	seenBool := false

	// Run many iterations to hit all three nil mutation branches.
	for seed := int64(0); seed < 500; seed++ {
		g := NewGenerator(target, seed)
		result := g.Next()
		switch result["x"].(type) {
		case string:
			seenString = true
		case int:
			seenInt = true
		case bool:
			seenBool = true
		}
		if seenString && seenInt && seenBool {
			break
		}
	}
	if !seenString {
		t.Error("never saw string from nil mutation")
	}
	if !seenInt {
		t.Error("never saw int from nil mutation")
	}
	if !seenBool {
		t.Error("never saw bool from nil mutation")
	}
}

func TestGenerator_SingleCharStringMutationDelete(t *testing.T) {
	// Test that deleting from a single-char string works (produces empty string).
	target := &Target{
		Name:       "single-char",
		Connector:  "mock",
		Action:     "test",
		Parameters: map[string]any{"s": "x"},
		FuzzFields: []string{"s"},
	}

	// Run many iterations to hit all mutation branches.
	g := NewGenerator(target, 42)
	for i := 0; i < 200; i++ {
		result := g.Next()
		if _, ok := result["s"].(string); !ok {
			t.Fatalf("expected string, got %T", result["s"])
		}
	}
}
