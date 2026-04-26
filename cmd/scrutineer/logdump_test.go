package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/scrutineer/scrutineer/core/exitcode"
	"github.com/scrutineer/scrutineer/core/telemetry"
)

func TestCmdLogDump_NoArgs(t *testing.T) {
	code := cmdLogDump(nil)
	if code != exitcode.ConfigError {
		t.Errorf("expected ConfigError (%d), got %d", exitcode.ConfigError, code)
	}
}

func TestCmdLogDump_MissingFile(t *testing.T) {
	code := cmdLogDump([]string{"/nonexistent/file.log"})
	if code != exitcode.ConfigError {
		t.Errorf("expected ConfigError (%d), got %d", exitcode.ConfigError, code)
	}
}

func TestCmdLogDump_ValidFile(t *testing.T) {
	// Create a valid telemetry log file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	w := telemetry.NewWriter(f)
	w.Write(telemetry.Record{
		Timestamp: 1000000000,
		EventType: telemetry.TestStart,
		Tags:      map[string]string{"test": "example"},
		Detail:    []byte("test detail"),
	})
	w.Write(telemetry.Record{
		Timestamp: 2000000000,
		EventType: telemetry.TestPass,
		Tags:      nil,
		Detail:    nil,
	})
	w.Close()
	f.Close()

	code := cmdLogDump([]string{path})
	if code != exitcode.OK {
		t.Errorf("expected OK (%d), got %d", exitcode.OK, code)
	}
}

func TestCmdLogDump_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.log")
	os.WriteFile(path, []byte("not a valid TLV file"), 0644)

	code := cmdLogDump([]string{path})
	if code != exitcode.InternalError {
		t.Errorf("expected InternalError (%d), got %d", exitcode.InternalError, code)
	}
}

func TestPrintRecord_WithJSON(t *testing.T) {
	r := telemetry.Record{
		Timestamp: 1000000000,
		EventType: telemetry.Request,
		Tags:      map[string]string{"url": "http://example.com"},
		Detail:    []byte(`{"method":"GET"}`),
	}
	// Just verify no panic
	printRecord(r)
}

func TestPrintRecord_WithRawDetail(t *testing.T) {
	r := telemetry.Record{
		Timestamp: 1000000000,
		EventType: telemetry.Error,
		Tags:      nil,
		Detail:    []byte("some error message"),
	}
	printRecord(r)
}

func TestPrintRecord_NoDetail(t *testing.T) {
	r := telemetry.Record{
		Timestamp: 1000000000,
		EventType: telemetry.SuiteStart,
	}
	printRecord(r)
}
