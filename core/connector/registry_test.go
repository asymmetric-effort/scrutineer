package connector

import (
	"testing"
)

func newMockFactory(name string) Factory {
	return func() Connector {
		return &mockConnector{name: name}
	}
}

func TestNewRegistryEmpty(t *testing.T) {
	r := NewRegistry()
	names := r.Names()
	if len(names) != 0 {
		t.Fatalf("expected empty registry, got %v", names)
	}
}

func TestRegisterAndGet(t *testing.T) {
	r := NewRegistry()

	if err := r.Register("http", newMockFactory("http")); err != nil {
		t.Fatalf("unexpected Register error: %v", err)
	}

	c, err := r.Get("http")
	if err != nil {
		t.Fatalf("unexpected Get error: %v", err)
	}
	if c.Name() != "http" {
		t.Fatalf("expected connector name 'http', got %q", c.Name())
	}
}

func TestRegisterDuplicateReturnsError(t *testing.T) {
	r := NewRegistry()

	if err := r.Register("cli", newMockFactory("cli")); err != nil {
		t.Fatalf("unexpected error on first register: %v", err)
	}

	err := r.Register("cli", newMockFactory("cli"))
	if err == nil {
		t.Fatal("expected error on duplicate register, got nil")
	}
}

func TestGetUnregisteredReturnsError(t *testing.T) {
	r := NewRegistry()

	_, err := r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered connector, got nil")
	}
}

func TestNamesSorted(t *testing.T) {
	r := NewRegistry()

	for _, name := range []string{"ssh", "http", "cli", "grpc"} {
		if err := r.Register(name, newMockFactory(name)); err != nil {
			t.Fatalf("unexpected Register error for %q: %v", name, err)
		}
	}

	names := r.Names()
	expected := []string{"cli", "grpc", "http", "ssh"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d names, got %d", len(expected), len(names))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Fatalf("expected names[%d] = %q, got %q", i, expected[i], name)
		}
	}
}

func TestRegisterMultipleAllRetrievable(t *testing.T) {
	r := NewRegistry()
	connectorNames := []string{"http", "cli", "ssh"}

	for _, name := range connectorNames {
		if err := r.Register(name, newMockFactory(name)); err != nil {
			t.Fatalf("unexpected Register error for %q: %v", name, err)
		}
	}

	for _, name := range connectorNames {
		c, err := r.Get(name)
		if err != nil {
			t.Fatalf("unexpected Get error for %q: %v", name, err)
		}
		if c.Name() != name {
			t.Fatalf("expected connector name %q, got %q", name, c.Name())
		}
	}
}

func TestGetCreatesNewInstance(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("http", newMockFactory("http")); err != nil {
		t.Fatalf("unexpected Register error: %v", err)
	}

	c1, _ := r.Get("http")
	c2, _ := r.Get("http")

	// Each call should return a distinct instance.
	if c1 == c2 {
		t.Fatal("expected distinct instances from Get, got same pointer")
	}
}
