// Package install handles downloading and managing browser binaries for
// browser automation testing.
package install

import "runtime"

// Platform holds the OS and architecture for browser download URL construction.
type Platform struct {
	OS   string
	Arch string
}

// DetectPlatform returns the current OS and architecture.
func DetectPlatform() Platform {
	return Platform{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
}

// Suffix returns the platform-specific suffix for download URLs.
func (p Platform) Suffix() string {
	switch p.OS {
	case "linux":
		if p.Arch == "arm64" {
			return "linux-arm64"
		}
		return "linux"
	case "darwin":
		if p.Arch == "arm64" {
			return "mac-arm64"
		}
		return "mac"
	case "windows":
		if p.Arch == "arm64" {
			return "win64-arm64"
		}
		return "win64"
	default:
		return p.OS + "-" + p.Arch
	}
}

// BinaryName returns the browser executable name for the platform.
func (p Platform) BinaryName(browser string) string {
	switch browser {
	case "chromium":
		if p.OS == "windows" {
			return "chrome.exe"
		}
		if p.OS == "darwin" {
			return "Chromium.app/Contents/MacOS/Chromium"
		}
		return "chrome"
	case "firefox":
		if p.OS == "windows" {
			return "firefox.exe"
		}
		if p.OS == "darwin" {
			return "Nightly.app/Contents/MacOS/firefox"
		}
		return "firefox"
	case "webkit":
		if p.OS == "windows" {
			return "MiniBrowser.exe"
		}
		if p.OS == "darwin" {
			return "MiniBrowser.app/Contents/MacOS/MiniBrowser"
		}
		return "minibrowser"
	default:
		return browser
	}
}
