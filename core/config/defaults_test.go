package config

import "testing"

func TestDefaults(t *testing.T) {
	cfg := Defaults()

	if cfg.Parallelism != 1 {
		t.Errorf("Parallelism: expected 1, got %d", cfg.Parallelism)
	}

	if cfg.Timeout != "30s" {
		t.Errorf("Timeout: expected '30s', got %q", cfg.Timeout)
	}

	if len(cfg.Reporters) != 1 {
		t.Fatalf("Reporters: expected 1, got %d", len(cfg.Reporters))
	}
	if cfg.Reporters[0].Type != "ansi" {
		t.Errorf("Reporters[0].Type: expected 'ansi', got %q", cfg.Reporters[0].Type)
	}
	if cfg.Reporters[0].Output != "" {
		t.Errorf("Reporters[0].Output: expected empty, got %q", cfg.Reporters[0].Output)
	}

	if cfg.Coverage.Threshold != 98.0 {
		t.Errorf("Coverage.Threshold: expected 98.0, got %f", cfg.Coverage.Threshold)
	}

	if cfg.Browsers.Chromium {
		t.Error("Browsers.Chromium: expected false")
	}
	if cfg.Browsers.Firefox {
		t.Error("Browsers.Firefox: expected false")
	}
	if cfg.Browsers.WebKit {
		t.Error("Browsers.WebKit: expected false")
	}

	if !cfg.Telemetry.Enabled {
		t.Error("Telemetry.Enabled: expected true")
	}
	if cfg.Telemetry.Output != "scrutineer.log" {
		t.Errorf("Telemetry.Output: expected 'scrutineer.log', got %q", cfg.Telemetry.Output)
	}

	if cfg.Version != "" {
		t.Errorf("Version: expected empty, got %q", cfg.Version)
	}
	if cfg.Tests != nil {
		t.Errorf("Tests: expected nil, got %v", cfg.Tests)
	}
	if cfg.Connectors != nil {
		t.Errorf("Connectors: expected nil, got %v", cfg.Connectors)
	}
}
