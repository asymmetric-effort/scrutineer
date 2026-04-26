package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/scrutineer/scrutineer/core/connector"
)

func TestFilesystemFileExists(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path": f,
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := result.Data["exists"].(bool); !got {
		t.Error("exists = false, want true")
	}
	if got := result.Data["size"].(int64); got != 5 {
		t.Errorf("size = %d, want 5", got)
	}
	if got := result.Data["content"].(string); got != "hello" {
		t.Errorf("content = %q, want %q", got, "hello")
	}
}

func TestFilesystemFileDoesNotExist(t *testing.T) {
	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path": "/tmp/nonexistent_file_xyz_12345",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := result.Data["exists"].(bool); got {
		t.Error("exists = true, want false")
	}
}

func TestFilesystemExpectedExistsTrue(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path":   f,
			"exists": true,
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if _, hasErr := result.Data["error"]; hasErr {
		t.Errorf("unexpected error in result: %v", result.Data["error"])
	}
}

func TestFilesystemExpectedExistsFalse(t *testing.T) {
	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path":   "/tmp/nonexistent_file_xyz_12345",
			"exists": false,
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if _, hasErr := result.Data["error"]; hasErr {
		t.Errorf("unexpected error in result: %v", result.Data["error"])
	}
}

func TestFilesystemExpectedExistsMismatchMissing(t *testing.T) {
	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path":   "/tmp/nonexistent_file_xyz_12345",
			"exists": true,
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if _, hasErr := result.Data["error"]; !hasErr {
		t.Error("expected error in result for existence mismatch")
	}
}

func TestFilesystemExpectedExistsMismatchPresent(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path":   f,
			"exists": false,
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if _, hasErr := result.Data["error"]; !hasErr {
		t.Error("expected error in result for existence mismatch")
	}
}

func TestFilesystemContains(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path":     f,
			"contains": "world",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := result.Data["contains"].(bool); !got {
		t.Error("contains = false, want true")
	}
}

func TestFilesystemContainsNotFound(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path":     f,
			"contains": "xyz",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := result.Data["contains"].(bool); got {
		t.Error("contains = true, want false")
	}
}

func TestFilesystemSizeGreaterThan(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path": f,
			"size": map[string]any{
				"greater_than": 5,
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := result.Data["size_greater_than"].(bool); !got {
		t.Error("size_greater_than = false, want true")
	}
}

func TestFilesystemSizeLessThan(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path": f,
			"size": map[string]any{
				"less_than": int64(100),
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := result.Data["size_less_than"].(bool); !got {
		t.Error("size_less_than = false, want true")
	}
}

func TestFilesystemSizeBothConstraints(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path": f,
			"size": map[string]any{
				"greater_than": 1,
				"less_than":    100,
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := result.Data["size_greater_than"].(bool); !got {
		t.Error("size_greater_than = false, want true")
	}
	if got := result.Data["size_less_than"].(bool); !got {
		t.Error("size_less_than = false, want true")
	}
}

func TestFilesystemDirectoryExists(t *testing.T) {
	dir := t.TempDir()

	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path": dir,
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := result.Data["exists"].(bool); !got {
		t.Error("exists = false, want true")
	}
	if got := result.Data["is_dir"].(bool); !got {
		t.Error("is_dir = false, want true")
	}
}

func TestFilesystemMissingPath(t *testing.T) {
	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action:     "filesystem",
		Parameters: map[string]any{},
	})
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestFilesystemInvalidExistsType(t *testing.T) {
	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path":   "/tmp",
			"exists": "yes",
		},
	})
	if err == nil {
		t.Fatal("expected error for non-bool exists")
	}
}

func TestFilesystemInvalidSizeType(t *testing.T) {
	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path": "/tmp",
			"size": "invalid",
		},
	})
	if err == nil {
		t.Fatal("expected error for non-map size")
	}
}

func TestFilesystemInvalidSizeValueType(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path": f,
			"size": map[string]any{
				"greater_than": "abc",
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for non-numeric size value")
	}
}

func TestFilesystemInvalidLessThanType(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path": f,
			"size": map[string]any{
				"less_than": "abc",
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for non-numeric less_than value")
	}
}

func TestFilesystemSizeWithFloat(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path": f,
			"size": map[string]any{
				"greater_than": float64(1),
				"less_than":    float32(100),
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := result.Data["size_greater_than"].(bool); !got {
		t.Error("size_greater_than = false, want true")
	}
	if got := result.Data["size_less_than"].(bool); !got {
		t.Error("size_less_than = false, want true")
	}
}

func TestFilesystemMeta(t *testing.T) {
	dir := t.TempDir()

	c := New()
	result, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path": dir,
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Meta["connector"] != "cli" {
		t.Errorf("Meta[connector] = %q, want %q", result.Meta["connector"], "cli")
	}
	if result.Meta["action"] != "filesystem" {
		t.Errorf("Meta[action] = %q, want %q", result.Meta["action"], "filesystem")
	}
}

func TestFilesystemInvalidContainsType(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path":     f,
			"contains": 123,
		},
	})
	if err == nil {
		t.Fatal("expected error for non-string contains")
	}
}

func TestFilesystemInvalidPathType(t *testing.T) {
	c := New()
	_, err := c.Execute(context.Background(), connector.Step{
		Action: "filesystem",
		Parameters: map[string]any{
			"path": 123,
		},
	})
	if err == nil {
		t.Fatal("expected error for non-string path")
	}
}
