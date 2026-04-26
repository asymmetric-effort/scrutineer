package config

import "testing"

func TestParseFlagsNoArgs(t *testing.T) {
	f, err := ParseFlags([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.ConfigFile != "" {
		t.Errorf("ConfigFile: expected empty, got %q", f.ConfigFile)
	}
	if f.Parallelism != 0 {
		t.Errorf("Parallelism: expected 0, got %d", f.Parallelism)
	}
	if f.Timeout != "" {
		t.Errorf("Timeout: expected empty, got %q", f.Timeout)
	}
	if f.Format != "" {
		t.Errorf("Format: expected empty, got %q", f.Format)
	}
	if f.Verbose {
		t.Error("Verbose: expected false")
	}
	if len(f.Tags) != 0 {
		t.Errorf("Tags: expected empty, got %v", f.Tags)
	}
}

func TestParseFlagsParallelism(t *testing.T) {
	f, err := ParseFlags([]string{"--parallelism", "8"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Parallelism != 8 {
		t.Errorf("Parallelism: expected 8, got %d", f.Parallelism)
	}
}

func TestParseFlagsTimeout(t *testing.T) {
	f, err := ParseFlags([]string{"--timeout", "2m"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Timeout != "2m" {
		t.Errorf("Timeout: expected '2m', got %q", f.Timeout)
	}
}

func TestParseFlagsFormat(t *testing.T) {
	f, err := ParseFlags([]string{"--format", "json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Format != "json" {
		t.Errorf("Format: expected 'json', got %q", f.Format)
	}
}

func TestParseFlagsVerbose(t *testing.T) {
	f, err := ParseFlags([]string{"--verbose"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.Verbose {
		t.Error("Verbose: expected true")
	}
}

func TestParseFlagsTags(t *testing.T) {
	f, err := ParseFlags([]string{"--tags", "smoke,regression"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.Tags) != 2 {
		t.Fatalf("Tags: expected 2, got %d", len(f.Tags))
	}
	if f.Tags[0] != "smoke" || f.Tags[1] != "regression" {
		t.Errorf("Tags: expected [smoke regression], got %v", f.Tags)
	}
}

func TestParseFlagsTagsMultiple(t *testing.T) {
	f, err := ParseFlags([]string{"--tags", "smoke", "--tags", "regression"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.Tags) != 2 {
		t.Fatalf("Tags: expected 2, got %d", len(f.Tags))
	}
	if f.Tags[0] != "smoke" || f.Tags[1] != "regression" {
		t.Errorf("Tags: expected [smoke regression], got %v", f.Tags)
	}
}

func TestParseFlagsConfig(t *testing.T) {
	f, err := ParseFlags([]string{"--config", "/path/to/config.yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.ConfigFile != "/path/to/config.yaml" {
		t.Errorf("ConfigFile: expected '/path/to/config.yaml', got %q", f.ConfigFile)
	}
}

func TestParseFlagsAllCombined(t *testing.T) {
	f, err := ParseFlags([]string{
		"--config", "my.yaml",
		"--parallelism", "4",
		"--timeout", "5s",
		"--format", "ansi",
		"--verbose",
		"--tags", "unit,integration",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.ConfigFile != "my.yaml" {
		t.Errorf("ConfigFile: expected 'my.yaml', got %q", f.ConfigFile)
	}
	if f.Parallelism != 4 {
		t.Errorf("Parallelism: expected 4, got %d", f.Parallelism)
	}
	if f.Timeout != "5s" {
		t.Errorf("Timeout: expected '5s', got %q", f.Timeout)
	}
	if f.Format != "ansi" {
		t.Errorf("Format: expected 'ansi', got %q", f.Format)
	}
	if !f.Verbose {
		t.Error("Verbose: expected true")
	}
	if len(f.Tags) != 2 {
		t.Fatalf("Tags: expected 2, got %d", len(f.Tags))
	}
}

func TestParseFlagsInvalid(t *testing.T) {
	_, err := ParseFlags([]string{"--unknown"})
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestParseFlagsInvalidParallelism(t *testing.T) {
	_, err := ParseFlags([]string{"--parallelism", "notanumber"})
	if err == nil {
		t.Error("expected error for non-integer parallelism")
	}
}

func TestParseFlagsTagsWithSpaces(t *testing.T) {
	f, err := ParseFlags([]string{"--tags", " smoke , regression "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.Tags) != 2 {
		t.Fatalf("Tags: expected 2, got %d", len(f.Tags))
	}
	if f.Tags[0] != "smoke" || f.Tags[1] != "regression" {
		t.Errorf("Tags: expected [smoke regression], got %v", f.Tags)
	}
}

func TestParseFlagsTagsEmpty(t *testing.T) {
	f, err := ParseFlags([]string{"--tags", ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.Tags) != 0 {
		t.Errorf("Tags: expected empty, got %v", f.Tags)
	}
}

func TestTagsValueString(t *testing.T) {
	tags := []string{"a", "b"}
	tv := &tagsValue{tags: &tags}
	s := tv.String()
	if s != "a,b" {
		t.Errorf("expected 'a,b', got %q", s)
	}
}

func TestTagsValueStringNil(t *testing.T) {
	tv := &tagsValue{tags: nil}
	s := tv.String()
	if s != "" {
		t.Errorf("expected empty, got %q", s)
	}
}
