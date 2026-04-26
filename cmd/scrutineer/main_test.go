package main

import (
	"testing"

	"github.com/scrutineer/scrutineer/core/connector"
)

func TestRegisterConnectors(t *testing.T) {
	registry := connector.NewRegistry()
	registerConnectors(registry)

	names := registry.Names()
	expected := []string{"browser", "cli", "grpc", "http", "ssh"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d connectors, got %d: %v", len(expected), len(names), names)
	}

	for _, name := range expected {
		c, err := registry.Get(name)
		if err != nil {
			t.Fatalf("get %s: %v", name, err)
		}
		if c.Name() != name {
			t.Errorf("%s connector Name() = %q, want %q", name, c.Name(), name)
		}
	}
}

func TestPrintUsage(t *testing.T) {
	// Just ensure it doesn't panic
	printUsage()
}

func TestPrintBrowsersUsage(t *testing.T) {
	printBrowsersUsage()
}
