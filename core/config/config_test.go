package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/scrutineer/scrutineer/core/schema"
)

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	content := []byte("version: \"1.0\"\nparallelism: 4\ntimeout: \"60s\"\n")
	if err := os.WriteFile(filepath.Join(dir, "scrutineer.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != "1.0" {
		t.Errorf("Version: expected '1.0', got %q", cfg.Version)
	}
	if cfg.Parallelism != 4 {
		t.Errorf("Parallelism: expected 4, got %d", cfg.Parallelism)
	}
	if cfg.Timeout != "60s" {
		t.Errorf("Timeout: expected '60s', got %q", cfg.Timeout)
	}
	// Defaults should still be present for unset fields
	if len(cfg.Reporters) != 1 || cfg.Reporters[0].Type != "ansi" {
		t.Errorf("Reporters: expected default [ansi], got %v", cfg.Reporters)
	}
	if cfg.Coverage.Threshold != 98.0 {
		t.Errorf("Coverage.Threshold: expected 98.0, got %f", cfg.Coverage.Threshold)
	}
}

func TestLoadMissingConfig(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil {
		t.Error("expected error for missing config file")
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.yaml")
	content := []byte("parallelism: 8\ntimeout: \"120s\"\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Parallelism != 8 {
		t.Errorf("Parallelism: expected 8, got %d", cfg.Parallelism)
	}
	if cfg.Timeout != "120s" {
		t.Errorf("Timeout: expected '120s', got %q", cfg.Timeout)
	}
}

func TestLoadFromFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	content := []byte(":\n  :\n    bad: [unterminated\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromFile(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadFromFileNotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestMergeParallelism(t *testing.T) {
	cfg := Defaults()
	flags := &Flags{Parallelism: 16}

	result := Merge(cfg, flags)
	if result.Parallelism != 16 {
		t.Errorf("Parallelism: expected 16, got %d", result.Parallelism)
	}
}

func TestMergeTimeout(t *testing.T) {
	cfg := Defaults()
	flags := &Flags{Timeout: "5m"}

	result := Merge(cfg, flags)
	if result.Timeout != "5m" {
		t.Errorf("Timeout: expected '5m', got %q", result.Timeout)
	}
}

func TestMergeFormatJSON(t *testing.T) {
	cfg := Defaults()
	flags := &Flags{Format: "json"}

	result := Merge(cfg, flags)
	if len(result.Reporters) != 1 {
		t.Fatalf("Reporters: expected 1, got %d", len(result.Reporters))
	}
	if result.Reporters[0].Type != "json" {
		t.Errorf("Reporters[0].Type: expected 'json', got %q", result.Reporters[0].Type)
	}
}

func TestMergeFormatANSI(t *testing.T) {
	cfg := Defaults()
	flags := &Flags{Format: "ansi"}

	result := Merge(cfg, flags)
	if len(result.Reporters) != 1 {
		t.Fatalf("Reporters: expected 1, got %d", len(result.Reporters))
	}
	if result.Reporters[0].Type != "ansi" {
		t.Errorf("Reporters[0].Type: expected 'ansi', got %q", result.Reporters[0].Type)
	}
}

func TestMergeVerbose(t *testing.T) {
	cfg := Defaults()
	cfg.Telemetry.Enabled = false
	flags := &Flags{Verbose: true}

	result := Merge(cfg, flags)
	if !result.Telemetry.Enabled {
		t.Error("Telemetry.Enabled: expected true after verbose flag")
	}
}

func TestMergeTags(t *testing.T) {
	cfg := Defaults()
	cfg.Tests = []string{"original.yaml"}
	flags := &Flags{Tags: []string{"smoke", "regression"}}

	result := Merge(cfg, flags)
	if len(result.Tests) != 2 {
		t.Fatalf("Tests: expected 2, got %d", len(result.Tests))
	}
	if result.Tests[0] != "smoke" || result.Tests[1] != "regression" {
		t.Errorf("Tests: expected [smoke regression], got %v", result.Tests)
	}
}

func TestMergeEmptyFlags(t *testing.T) {
	cfg := Defaults()
	original := *cfg
	originalReporters := make([]schema.ReporterConfig, len(cfg.Reporters))
	copy(originalReporters, cfg.Reporters)

	flags := &Flags{}
	result := Merge(cfg, flags)

	if result.Parallelism != original.Parallelism {
		t.Errorf("Parallelism changed: expected %d, got %d", original.Parallelism, result.Parallelism)
	}
	if result.Timeout != original.Timeout {
		t.Errorf("Timeout changed: expected %q, got %q", original.Timeout, result.Timeout)
	}
	if len(result.Reporters) != len(originalReporters) {
		t.Errorf("Reporters changed: expected %d, got %d", len(originalReporters), len(result.Reporters))
	}
	if result.Reporters[0].Type != originalReporters[0].Type {
		t.Errorf("Reporters[0].Type changed: expected %q, got %q", originalReporters[0].Type, result.Reporters[0].Type)
	}
}

func TestMergeMultipleFlags(t *testing.T) {
	cfg := Defaults()
	flags := &Flags{
		Parallelism: 4,
		Timeout:     "10s",
		Format:      "json",
		Verbose:     true,
		Tags:        []string{"unit"},
	}

	result := Merge(cfg, flags)
	if result.Parallelism != 4 {
		t.Errorf("Parallelism: expected 4, got %d", result.Parallelism)
	}
	if result.Timeout != "10s" {
		t.Errorf("Timeout: expected '10s', got %q", result.Timeout)
	}
	if result.Reporters[0].Type != "json" {
		t.Errorf("Format: expected 'json', got %q", result.Reporters[0].Type)
	}
	if !result.Telemetry.Enabled {
		t.Error("Telemetry.Enabled: expected true")
	}
	if len(result.Tests) != 1 || result.Tests[0] != "unit" {
		t.Errorf("Tags: expected [unit], got %v", result.Tests)
	}
}
