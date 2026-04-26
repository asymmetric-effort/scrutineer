package install

import (
	"strings"
	"testing"
)

func TestLookupRevision(t *testing.T) {
	tests := []struct {
		browser string
		found   bool
	}{
		{"chromium", true},
		{"firefox", true},
		{"webkit", true},
		{"opera", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.browser, func(t *testing.T) {
			rev, ok := LookupRevision(tt.browser)
			if ok != tt.found {
				t.Errorf("LookupRevision(%q) found = %v, want %v", tt.browser, ok, tt.found)
			}
			if ok {
				if rev.Browser != tt.browser {
					t.Errorf("Browser = %q, want %q", rev.Browser, tt.browser)
				}
				if rev.Revision == "" {
					t.Error("Revision should not be empty")
				}
				if rev.BaseURL == "" {
					t.Error("BaseURL should not be empty")
				}
			}
		})
	}
}

func TestDownloadURL(t *testing.T) {
	tests := []struct {
		name     string
		browser  string
		platform Platform
		wantErr  bool
		contains []string
	}{
		{
			"chromium linux",
			"chromium",
			Platform{"linux", "amd64"},
			false,
			[]string{"chromium", "linux", ".zip"},
		},
		{
			"firefox darwin arm64",
			"firefox",
			Platform{"darwin", "arm64"},
			false,
			[]string{"firefox", "mac-arm64", ".zip"},
		},
		{
			"webkit windows",
			"webkit",
			Platform{"windows", "amd64"},
			false,
			[]string{"webkit", "win64", ".zip"},
		},
		{
			"unknown browser",
			"opera",
			Platform{"linux", "amd64"},
			true,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := DownloadURL(tt.browser, tt.platform)
			if (err != nil) != tt.wantErr {
				t.Errorf("DownloadURL error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for _, substr := range tt.contains {
				if !strings.Contains(url, substr) {
					t.Errorf("URL %q should contain %q", url, substr)
				}
			}
		})
	}
}

func TestSupportedBrowsers(t *testing.T) {
	browsers := SupportedBrowsers()
	if len(browsers) != 3 {
		t.Errorf("expected 3 browsers, got %d", len(browsers))
	}

	expected := map[string]bool{"chromium": true, "firefox": true, "webkit": true}
	for _, b := range browsers {
		if !expected[b] {
			t.Errorf("unexpected browser: %s", b)
		}
	}
}

func TestDownloadURL_Format(t *testing.T) {
	url, err := DownloadURL("chromium", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatalf("DownloadURL: %v", err)
	}

	// Should follow pattern: baseURL/revision/browser-suffix.zip
	if !strings.HasPrefix(url, "https://") {
		t.Errorf("URL should start with https://: %s", url)
	}
	if !strings.HasSuffix(url, ".zip") {
		t.Errorf("URL should end with .zip: %s", url)
	}
}
