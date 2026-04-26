package browser

import (
	"strings"
	"testing"
	"time"
)

func TestChromiumArgs_Headless(t *testing.T) {
	args := chromiumArgs(true, nil)

	found := false
	for _, a := range args {
		if a == "--headless=new" {
			found = true
		}
	}
	if !found {
		t.Error("headless args should contain --headless=new")
	}

	// Should contain debugging port.
	hasDebugPort := false
	for _, a := range args {
		if a == "--remote-debugging-port=0" {
			hasDebugPort = true
		}
	}
	if !hasDebugPort {
		t.Error("should contain --remote-debugging-port=0")
	}
}

func TestChromiumArgs_NotHeadless(t *testing.T) {
	args := chromiumArgs(false, nil)

	for _, a := range args {
		if strings.Contains(a, "headless") {
			t.Errorf("non-headless args should not contain headless flag: %s", a)
		}
	}
}

func TestChromiumArgs_ExtraArgs(t *testing.T) {
	extra := []string{"--window-size=1920,1080", "--disable-gpu"}
	args := chromiumArgs(true, extra)

	for _, e := range extra {
		found := false
		for _, a := range args {
			if a == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("extra arg %q not found in args", e)
		}
	}
}

func TestFirefoxArgs_Headless(t *testing.T) {
	args := firefoxArgs(true, nil)

	found := false
	for _, a := range args {
		if a == "--headless" {
			found = true
		}
	}
	if !found {
		t.Error("headless firefox should contain --headless")
	}

	hasNoRemote := false
	for _, a := range args {
		if a == "--no-remote" {
			hasNoRemote = true
		}
	}
	if !hasNoRemote {
		t.Error("should contain --no-remote")
	}
}

func TestFirefoxArgs_NotHeadless(t *testing.T) {
	args := firefoxArgs(false, nil)

	for _, a := range args {
		if a == "--headless" {
			t.Error("non-headless should not contain --headless")
		}
	}
}

func TestFirefoxArgs_ExtraArgs(t *testing.T) {
	extra := []string{"--profile", "/tmp/test"}
	args := firefoxArgs(true, extra)

	for _, e := range extra {
		found := false
		for _, a := range args {
			if a == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("extra arg %q not found", e)
		}
	}
}

func TestWebkitArgs_Headless(t *testing.T) {
	args := webkitArgs(true, nil)

	found := false
	for _, a := range args {
		if a == "--headless" {
			found = true
		}
	}
	if !found {
		t.Error("headless webkit should contain --headless")
	}
}

func TestWebkitArgs_NotHeadless(t *testing.T) {
	args := webkitArgs(false, nil)

	for _, a := range args {
		if a == "--headless" {
			t.Error("non-headless should not contain --headless")
		}
	}
}

func TestWebkitArgs_ExtraArgs(t *testing.T) {
	extra := []string{"--extra"}
	args := webkitArgs(false, extra)

	found := false
	for _, a := range args {
		if a == "--extra" {
			found = true
		}
	}
	if !found {
		t.Error("extra arg not found")
	}
}

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		browser  string
		headless bool
	}{
		{"chromium", true},
		{"chromium", false},
		{"firefox", true},
		{"firefox", false},
		{"webkit", true},
		{"webkit", false},
		{"unknown", true}, // defaults to chromium
	}

	for _, tt := range tests {
		t.Run(tt.browser, func(t *testing.T) {
			args := buildArgs(tt.browser, tt.headless, nil)
			if len(args) == 0 {
				t.Error("should produce args")
			}
		})
	}
}

func TestExtractWSURL_Success(t *testing.T) {
	// Simulate browser output with a WebSocket URL.
	output := "DevTools listening on ws://127.0.0.1:9222/devtools/browser/abc123\n"
	reader := strings.NewReader(output)

	url, err := extractWSURL(reader, 2*time.Second)
	if err != nil {
		t.Fatalf("extractWSURL: %v", err)
	}

	expected := "ws://127.0.0.1:9222/devtools/browser/abc123"
	if url != expected {
		t.Errorf("url = %q, want %q", url, expected)
	}
}

func TestExtractWSURL_NoURL(t *testing.T) {
	output := "Some other output\nNo websocket here\n"
	reader := strings.NewReader(output)

	_, err := extractWSURL(reader, 2*time.Second)
	if err == nil {
		t.Error("expected error when no URL found")
	}
	if !strings.Contains(err.Error(), "no websocket url") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtractWSURL_URLInMiddle(t *testing.T) {
	output := "Starting browser...\nLoading profiles...\nDevTools listening on ws://localhost:12345/devtools/browser/xyz\nBrowser ready.\n"
	reader := strings.NewReader(output)

	url, err := extractWSURL(reader, 2*time.Second)
	if err != nil {
		t.Fatalf("extractWSURL: %v", err)
	}

	if !strings.HasPrefix(url, "ws://") {
		t.Errorf("url should start with ws://: %s", url)
	}
}

func TestExtractWSURL_Timeout(t *testing.T) {
	// Use a reader that blocks forever.
	pr, _ := newBlockingReader()

	_, err := extractWSURL(pr, 100*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("unexpected error: %v", err)
	}
}

// newBlockingReader returns a reader that never returns data.
type blockingReader struct{}

func (b *blockingReader) Read(p []byte) (int, error) {
	select {} // block forever
}

func newBlockingReader() (*blockingReader, func()) {
	return &blockingReader{}, func() {}
}

func TestBrowserProcess_Kill_Nil(t *testing.T) {
	bp := &browserProcess{}
	err := bp.kill()
	if err != nil {
		t.Errorf("kill nil process: %v", err)
	}
}

func TestChromiumArgs_SecurityFlags(t *testing.T) {
	args := chromiumArgs(true, nil)

	requiredFlags := []string{
		"--no-first-run",
		"--disable-extensions",
		"--disable-dev-shm-usage",
		"--disable-background-networking",
	}

	for _, flag := range requiredFlags {
		found := false
		for _, a := range args {
			if a == flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing required flag: %s", flag)
		}
	}
}

func TestExtractWSURL_WithTrailingSpaces(t *testing.T) {
	output := "DevTools listening on ws://127.0.0.1:9222/test   \n"
	reader := strings.NewReader(output)

	url, err := extractWSURL(reader, 2*time.Second)
	if err != nil {
		t.Fatalf("extractWSURL: %v", err)
	}

	if strings.Contains(url, " ") {
		t.Errorf("url should not contain spaces: %q", url)
	}
}
