package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// Manager handles browser binary downloads and lifecycle.
type Manager struct {
	// BaseDir is the root directory for storing browser binaries.
	BaseDir string
}

// NewManager creates a new install manager with the given base directory.
func NewManager(baseDir string) *Manager {
	return &Manager{BaseDir: baseDir}
}

// DefaultBaseDir returns the default directory for browser binaries.
func DefaultBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".scrutineer", "browsers")
}

// BrowserPath returns the expected path to the browser binary.
func (m *Manager) BrowserPath(browser string) (string, error) {
	rev, ok := LookupRevision(browser)
	if !ok {
		return "", fmt.Errorf("install: unknown browser %q", browser)
	}

	platform := DetectPlatform()
	binaryName := platform.BinaryName(browser)
	return filepath.Join(m.BaseDir, browser+"-"+rev.Revision, binaryName), nil
}

// IsInstalled checks whether the browser binary exists.
func (m *Manager) IsInstalled(browser string) (bool, error) {
	path, err := m.BrowserPath(browser)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// NeedsInstall returns browsers that are not yet installed from the given list.
func (m *Manager) NeedsInstall(browsers []string) ([]string, error) {
	var missing []string
	for _, b := range browsers {
		installed, err := m.IsInstalled(b)
		if err != nil {
			return nil, err
		}
		if !installed {
			missing = append(missing, b)
		}
	}
	return missing, nil
}
