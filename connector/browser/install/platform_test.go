package install

import (
	"runtime"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	p := DetectPlatform()
	if p.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", p.OS, runtime.GOOS)
	}
	if p.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", p.Arch, runtime.GOARCH)
	}
}

func TestPlatform_Suffix(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		want     string
	}{
		{"linux amd64", Platform{"linux", "amd64"}, "linux"},
		{"linux arm64", Platform{"linux", "arm64"}, "linux-arm64"},
		{"darwin amd64", Platform{"darwin", "amd64"}, "mac"},
		{"darwin arm64", Platform{"darwin", "arm64"}, "mac-arm64"},
		{"windows amd64", Platform{"windows", "amd64"}, "win64"},
		{"windows arm64", Platform{"windows", "arm64"}, "win64-arm64"},
		{"unknown", Platform{"freebsd", "amd64"}, "freebsd-amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.platform.Suffix()
			if got != tt.want {
				t.Errorf("Suffix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPlatform_BinaryName(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		browser  string
		want     string
	}{
		// Chromium.
		{"chromium linux", Platform{"linux", "amd64"}, "chromium", "chrome"},
		{"chromium darwin", Platform{"darwin", "amd64"}, "chromium", "Chromium.app/Contents/MacOS/Chromium"},
		{"chromium windows", Platform{"windows", "amd64"}, "chromium", "chrome.exe"},
		// Firefox.
		{"firefox linux", Platform{"linux", "amd64"}, "firefox", "firefox"},
		{"firefox darwin", Platform{"darwin", "amd64"}, "firefox", "Nightly.app/Contents/MacOS/firefox"},
		{"firefox windows", Platform{"windows", "amd64"}, "firefox", "firefox.exe"},
		// WebKit.
		{"webkit linux", Platform{"linux", "amd64"}, "webkit", "minibrowser"},
		{"webkit darwin", Platform{"darwin", "amd64"}, "webkit", "MiniBrowser.app/Contents/MacOS/MiniBrowser"},
		{"webkit windows", Platform{"windows", "amd64"}, "webkit", "MiniBrowser.exe"},
		// Unknown.
		{"unknown browser", Platform{"linux", "amd64"}, "opera", "opera"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.platform.BinaryName(tt.browser)
			if got != tt.want {
				t.Errorf("BinaryName(%q) = %q, want %q", tt.browser, got, tt.want)
			}
		})
	}
}
