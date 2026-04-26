package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager("/tmp/test-browsers")
	if m.BaseDir != "/tmp/test-browsers" {
		t.Errorf("BaseDir = %q, want %q", m.BaseDir, "/tmp/test-browsers")
	}
}

func TestDefaultBaseDir(t *testing.T) {
	dir := DefaultBaseDir()
	if dir == "" {
		t.Error("DefaultBaseDir should not be empty")
	}
	if !strings.Contains(dir, "browsers") {
		t.Errorf("DefaultBaseDir should contain 'browsers': %s", dir)
	}
	if !strings.Contains(dir, ".scrutineer") {
		t.Errorf("DefaultBaseDir should contain '.scrutineer': %s", dir)
	}
}

func TestManager_BrowserPath(t *testing.T) {
	m := NewManager("/opt/browsers")

	tests := []struct {
		browser string
		wantErr bool
	}{
		{"chromium", false},
		{"firefox", false},
		{"webkit", false},
		{"opera", true},
	}

	for _, tt := range tests {
		t.Run(tt.browser, func(t *testing.T) {
			path, err := m.BrowserPath(tt.browser)
			if (err != nil) != tt.wantErr {
				t.Errorf("BrowserPath(%q) error = %v, wantErr %v", tt.browser, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !strings.HasPrefix(path, "/opt/browsers") {
					t.Errorf("path should start with base dir: %s", path)
				}
				if !strings.Contains(path, tt.browser) {
					t.Errorf("path should contain browser name: %s", path)
				}
			}
		})
	}
}

func TestManager_IsInstalled_NotInstalled(t *testing.T) {
	m := NewManager(t.TempDir())

	installed, err := m.IsInstalled("chromium")
	if err != nil {
		t.Fatalf("IsInstalled: %v", err)
	}
	if installed {
		t.Error("should not be installed")
	}
}

func TestManager_IsInstalled_Installed(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Create the expected binary path.
	path, err := m.BrowserPath("chromium")
	if err != nil {
		t.Fatalf("BrowserPath: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("fake binary"), 0755); err != nil {
		t.Fatalf("write: %v", err)
	}

	installed, err := m.IsInstalled("chromium")
	if err != nil {
		t.Fatalf("IsInstalled: %v", err)
	}
	if !installed {
		t.Error("should be installed")
	}
}

func TestManager_IsInstalled_UnknownBrowser(t *testing.T) {
	m := NewManager(t.TempDir())
	_, err := m.IsInstalled("opera")
	if err == nil {
		t.Error("expected error for unknown browser")
	}
}

func TestManager_NeedsInstall(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// None installed.
	missing, err := m.NeedsInstall([]string{"chromium", "firefox"})
	if err != nil {
		t.Fatalf("NeedsInstall: %v", err)
	}
	if len(missing) != 2 {
		t.Errorf("expected 2 missing, got %d", len(missing))
	}

	// Install chromium.
	path, _ := m.BrowserPath("chromium")
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte("binary"), 0755)

	missing, err = m.NeedsInstall([]string{"chromium", "firefox"})
	if err != nil {
		t.Fatalf("NeedsInstall: %v", err)
	}
	if len(missing) != 1 {
		t.Errorf("expected 1 missing, got %d", len(missing))
	}
	if missing[0] != "firefox" {
		t.Errorf("expected firefox missing, got %s", missing[0])
	}
}

func TestManager_NeedsInstall_AllInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Install all.
	for _, b := range []string{"chromium", "firefox", "webkit"} {
		path, _ := m.BrowserPath(b)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte("binary"), 0755)
	}

	missing, err := m.NeedsInstall([]string{"chromium", "firefox", "webkit"})
	if err != nil {
		t.Fatalf("NeedsInstall: %v", err)
	}
	if len(missing) != 0 {
		t.Errorf("expected 0 missing, got %d", len(missing))
	}
}

func TestManager_NeedsInstall_UnknownBrowser(t *testing.T) {
	m := NewManager(t.TempDir())
	_, err := m.NeedsInstall([]string{"opera"})
	if err == nil {
		t.Error("expected error for unknown browser")
	}
}

func TestDefaultBaseDir_NoHome(t *testing.T) {
	// Temporarily unset HOME to trigger the fallback path.
	origHome := os.Getenv("HOME")
	os.Unsetenv("HOME")
	defer os.Setenv("HOME", origHome)

	dir := DefaultBaseDir()
	if dir == "" {
		t.Error("should not be empty even without HOME")
	}
	if !strings.Contains(dir, "browsers") {
		t.Errorf("should contain 'browsers': %s", dir)
	}
}

func TestManager_NeedsInstall_Empty(t *testing.T) {
	m := NewManager(t.TempDir())
	missing, err := m.NeedsInstall(nil)
	if err != nil {
		t.Fatalf("NeedsInstall: %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil, got %v", missing)
	}
}
