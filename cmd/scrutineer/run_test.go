package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/scrutineer/scrutineer/core/connector"
	"github.com/scrutineer/scrutineer/core/exitcode"
	"github.com/scrutineer/scrutineer/core/schema"
)

func TestCmdRun_NoConfig(t *testing.T) {
	// Run from a temp dir with no scrutineer.yaml
	origDir, _ := os.Getwd()
	dir := t.TempDir()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	registry := connector.NewRegistry()
	registerConnectors(registry)

	code := cmdRun(registry, nil)
	if code != exitcode.ConfigError {
		t.Errorf("expected ConfigError (%d), got %d", exitcode.ConfigError, code)
	}
}

func TestCmdRun_InvalidFlags(t *testing.T) {
	registry := connector.NewRegistry()
	code := cmdRun(registry, []string{"--unknown-flag"})
	if code != exitcode.ConfigError {
		t.Errorf("expected ConfigError (%d), got %d", exitcode.ConfigError, code)
	}
}

func TestCmdRun_EmptyManifest(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "scrutineer.yaml")
	os.WriteFile(configPath, []byte("version: \"0.0.1\"\ntests: []\n"), 0644)

	registry := connector.NewRegistry()
	registerConnectors(registry)

	code := cmdRun(registry, []string{"--config", configPath})
	if code != exitcode.ConfigError {
		t.Errorf("expected ConfigError (%d), got %d", exitcode.ConfigError, code)
	}
}

func TestLoadSuites_ValidFile(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.yaml")
	content := `suite: "Test Suite"
tests:
  - name: "test1"
    connector: cli
    steps:
      - action: exec
        command: "echo hello"
`
	os.WriteFile(testFile, []byte(content), 0644)

	suites, err := loadSuites([]string{testFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(suites))
	}
	if suites[0].Suite != "Test Suite" {
		t.Errorf("suite name = %q, want %q", suites[0].Suite, "Test Suite")
	}
}

func TestLoadSuites_MissingFile(t *testing.T) {
	_, err := loadSuites([]string{"/nonexistent/test.yaml"})
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadSuites_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "bad.yaml")
	os.WriteFile(testFile, []byte("not valid: [yaml: {broken"), 0644)

	_, err := loadSuites([]string{testFile})
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestBuildReporter_ANSI(t *testing.T) {
	cfg := &schema.Config{
		Reporters: []schema.ReporterConfig{{Type: "ansi"}},
	}
	r := buildReporter(cfg)
	if r == nil {
		t.Error("expected non-nil reporter")
	}
}

func TestBuildReporter_JSON(t *testing.T) {
	cfg := &schema.Config{
		Reporters: []schema.ReporterConfig{{Type: "json"}},
	}
	r := buildReporter(cfg)
	if r == nil {
		t.Error("expected non-nil reporter")
	}
}

func TestBuildReporter_Default(t *testing.T) {
	cfg := &schema.Config{}
	r := buildReporter(cfg)
	if r == nil {
		t.Error("expected non-nil reporter")
	}
}

func TestCmdBrowsers_NoArgs(t *testing.T) {
	code := cmdBrowsers(nil)
	if code != exitcode.ConfigError {
		t.Errorf("expected ConfigError, got %d", code)
	}
}

func TestCmdBrowsers_Install(t *testing.T) {
	code := cmdBrowsers([]string{"install"})
	if code != exitcode.OK {
		t.Errorf("expected OK, got %d", code)
	}
}

func TestCmdBrowsers_List(t *testing.T) {
	code := cmdBrowsers([]string{"list"})
	if code != exitcode.OK {
		t.Errorf("expected OK, got %d", code)
	}
}

func TestCmdBrowsers_Help(t *testing.T) {
	code := cmdBrowsers([]string{"help"})
	if code != exitcode.OK {
		t.Errorf("expected OK, got %d", code)
	}
}

func TestCmdBrowsers_Unknown(t *testing.T) {
	code := cmdBrowsers([]string{"unknown"})
	if code != exitcode.ConfigError {
		t.Errorf("expected ConfigError, got %d", code)
	}
}
