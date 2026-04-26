package fuzz

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewCorpus(t *testing.T) {
	c := NewCorpus("/tmp/test-corpus")
	if c == nil {
		t.Fatal("expected non-nil corpus")
	}
	if c.dir != "/tmp/test-corpus" {
		t.Errorf("unexpected dir: %s", c.dir)
	}
}

func TestCorpus_LoadEmptyDir(t *testing.T) {
	dir := t.TempDir()
	c := NewCorpus(dir)
	if err := c.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Entries()) != 0 {
		t.Errorf("expected 0 entries, got %d", len(c.Entries()))
	}
}

func TestCorpus_LoadNonExistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir", "deep")
	c := NewCorpus(dir)
	if err := c.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Directory should have been created.
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestCorpus_AddAndLoad(t *testing.T) {
	dir := t.TempDir()
	c := NewCorpus(dir)

	entry := map[string]any{"key": "value", "num": float64(42)}
	if err := c.Add(entry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.Entries()) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(c.Entries()))
	}

	// Reload from disk.
	c2 := NewCorpus(dir)
	if err := c2.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c2.Entries()) != 1 {
		t.Fatalf("expected 1 entry after reload, got %d", len(c2.Entries()))
	}

	loaded := c2.Entries()[0]
	if loaded["key"] != "value" {
		t.Errorf("expected key=value, got %v", loaded["key"])
	}
}

func TestCorpus_SaveEntryWritesValidJSON(t *testing.T) {
	dir := t.TempDir()
	c := NewCorpus(dir)

	entry := map[string]any{"hello": "world"}
	if err := c.SaveEntry(entry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the written file.
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	data, err := os.ReadFile(filepath.Join(dir, files[0].Name()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("file is not valid JSON: %v", err)
	}
	if parsed["hello"] != "world" {
		t.Errorf("expected hello=world, got %v", parsed["hello"])
	}
}

func TestCorpus_LoadWithCorruptFile(t *testing.T) {
	dir := t.TempDir()

	// Write a valid file.
	validData, _ := json.Marshal(map[string]any{"valid": true})
	if err := os.WriteFile(filepath.Join(dir, "a_valid.json"), validData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a corrupt file.
	if err := os.WriteFile(filepath.Join(dir, "b_corrupt.json"), []byte("not json{{{"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewCorpus(dir)
	if err := c.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the valid entry should be loaded.
	if len(c.Entries()) != 1 {
		t.Errorf("expected 1 entry (corrupt skipped), got %d", len(c.Entries()))
	}
}

func TestCorpus_LoadSkipsNonJSON(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewCorpus(dir)
	if err := c.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Entries()) != 0 {
		t.Errorf("expected 0 entries, got %d", len(c.Entries()))
	}
}

func TestCorpus_LoadSkipsDirectories(t *testing.T) {
	dir := t.TempDir()

	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	c := NewCorpus(dir)
	if err := c.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Entries()) != 0 {
		t.Errorf("expected 0 entries, got %d", len(c.Entries()))
	}
}

func TestCorpus_LoadInvalidDir(t *testing.T) {
	// Use a path that cannot be created (file as parent).
	tmpFile := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(tmpFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewCorpus(filepath.Join(tmpFile, "subdir"))
	if err := c.Load(); err == nil {
		t.Fatal("expected error for invalid dir path")
	}
}

func TestCorpus_AddErrorPropagation(t *testing.T) {
	// Use a path where SaveEntry will fail.
	tmpFile := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(tmpFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewCorpus(filepath.Join(tmpFile, "subdir"))
	err := c.Add(map[string]any{"key": "val"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCorpus_SaveEntryInvalidDir(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(tmpFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewCorpus(filepath.Join(tmpFile, "subdir"))
	err := c.SaveEntry(map[string]any{"key": "val"})
	if err == nil {
		t.Fatal("expected error for invalid dir")
	}
}

func TestCorpus_SaveEntryWriteError(t *testing.T) {
	dir := t.TempDir()
	c := NewCorpus(dir)
	// Make the directory read-only so file creation fails.
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	err := c.SaveEntry(map[string]any{"key": "val"})
	if err == nil {
		t.Fatal("expected error writing to read-only dir")
	}
}

func TestCorpus_LoadUnreadableFile(t *testing.T) {
	dir := t.TempDir()

	// Write a valid JSON file then make it unreadable.
	path := filepath.Join(dir, "unreadable.json")
	if err := os.WriteFile(path, []byte(`{"x":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o644) })

	c := NewCorpus(dir)
	if err := c.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Unreadable file should be skipped.
	if len(c.Entries()) != 0 {
		t.Errorf("expected 0 entries (unreadable skipped), got %d", len(c.Entries()))
	}
}

func TestCorpus_MultipleAdds(t *testing.T) {
	dir := t.TempDir()
	c := NewCorpus(dir)

	for i := 0; i < 5; i++ {
		if err := c.Add(map[string]any{"i": float64(i)}); err != nil {
			t.Fatalf("unexpected error on add %d: %v", i, err)
		}
	}

	if len(c.Entries()) != 5 {
		t.Errorf("expected 5 entries, got %d", len(c.Entries()))
	}
}
