package engine

import "testing"

func TestNewTestContext(t *testing.T) {
	fixtures := map[string]any{
		"user": map[string]any{
			"name":  "alice",
			"email": "alice@example.com",
		},
	}

	tctx := NewTestContext("my-suite", "my-test", fixtures)

	if tctx.Suite != "my-suite" {
		t.Errorf("Suite = %q, want %q", tctx.Suite, "my-suite")
	}
	if tctx.Test != "my-test" {
		t.Errorf("Test = %q, want %q", tctx.Test, "my-test")
	}
	if tctx.Store == nil {
		t.Fatal("Store is nil")
	}

	// Verify fixtures are accessible through the store.
	val, ok := tctx.Store.Resolve("fixture.user.name")
	if !ok {
		t.Fatal("fixture.user.name not resolved")
	}
	if val != "alice" {
		t.Errorf("fixture.user.name = %v, want %q", val, "alice")
	}
}

func TestNewTestContextEmptyFixtures(t *testing.T) {
	tctx := NewTestContext("s", "t", nil)

	if tctx.Store == nil {
		t.Fatal("Store is nil with nil fixtures")
	}
	if tctx.Suite != "s" {
		t.Errorf("Suite = %q, want %q", tctx.Suite, "s")
	}
	if tctx.Test != "t" {
		t.Errorf("Test = %q, want %q", tctx.Test, "t")
	}
}

func TestNewTestContextEmptyMap(t *testing.T) {
	tctx := NewTestContext("suite", "test", map[string]any{})

	if tctx.Store == nil {
		t.Fatal("Store is nil with empty fixtures map")
	}
}

func TestTestContextStoreOperations(t *testing.T) {
	tctx := NewTestContext("s", "t", map[string]any{
		"base_url": "https://api.example.com",
	})

	// Set and get capture.
	tctx.Store.SetCapture("user_id", "42")
	val, ok := tctx.Store.GetCapture("user_id")
	if !ok {
		t.Fatal("capture user_id not found")
	}
	if val != "42" {
		t.Errorf("capture user_id = %v, want %q", val, "42")
	}

	// Interpolate using fixture.
	result, err := tctx.Store.Interpolate("url: ${fixture.base_url}/users")
	if err != nil {
		t.Fatalf("Interpolate error: %v", err)
	}
	if result != "url: https://api.example.com/users" {
		t.Errorf("Interpolate = %q, want %q", result, "url: https://api.example.com/users")
	}

	// Interpolate using capture.
	result, err = tctx.Store.Interpolate("id=${capture.user_id}")
	if err != nil {
		t.Fatalf("Interpolate error: %v", err)
	}
	if result != "id=42" {
		t.Errorf("Interpolate = %q, want %q", result, "id=42")
	}
}
