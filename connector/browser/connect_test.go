package browser

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/scrutineer/scrutineer/connector/browser/cdp"
)

func TestParseConfig_Defaults(t *testing.T) {
	b := New()
	err := b.parseConfig(map[string]any{})
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if b.browserType != "chromium" {
		t.Errorf("browserType = %q, want chromium", b.browserType)
	}
	if !b.headless {
		t.Error("headless should be true by default")
	}
}

func TestParseConfig_AllOptions(t *testing.T) {
	b := New()
	err := b.parseConfig(map[string]any{
		"browser":  "firefox",
		"headless": false,
		"args":     []string{"--flag1"},
	})
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if b.browserType != "firefox" {
		t.Errorf("browserType = %q, want firefox", b.browserType)
	}
	if b.headless {
		t.Error("headless should be false")
	}
	if len(b.extraArgs) != 1 || b.extraArgs[0] != "--flag1" {
		t.Errorf("extraArgs = %v", b.extraArgs)
	}
}

func TestParseConfig_InvalidBrowser(t *testing.T) {
	b := New()
	err := b.parseConfig(map[string]any{"browser": "ie"})
	if err == nil {
		t.Error("expected error for invalid browser")
	}
}

func TestParseConfig_ArgsAsAny(t *testing.T) {
	b := New()
	err := b.parseConfig(map[string]any{
		"args": []any{"--a", "--b", 42}, // 42 is not a string, should be skipped
	})
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if len(b.extraArgs) != 2 {
		t.Errorf("extraArgs length = %d, want 2", len(b.extraArgs))
	}
}

func TestConnectToWSURL_Success(t *testing.T) {
	wsURL, cleanup := mockCDPBrowserServer(nil)
	defer cleanup()

	b := New()
	err := b.connectToWSURL(nil, wsURL)
	if err != nil {
		t.Fatalf("connectToWSURL: %v", err)
	}
	defer b.client.Close()

	if b.client == nil {
		t.Error("client should not be nil")
	}
	if b.session == nil {
		t.Error("session should not be nil")
	}
}

func TestConnectToWSURL_DialError(t *testing.T) {
	b := New()
	err := b.connectToWSURL(nil, "ws://127.0.0.1:1/invalid")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
	if b.client != nil {
		t.Error("client should be nil on error")
	}
}

func TestConnectToWSURL_CreateTargetError(t *testing.T) {
	wsURL, cleanup := mockCDPSetup(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Target.createTarget" {
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "Cannot create target"},
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	defer cleanup()

	b := New()
	err := b.connectToWSURL(nil, wsURL)
	if err == nil {
		t.Error("expected error for create target failure")
	}
	if b.client != nil {
		t.Error("client should be nil on error")
	}
}

func TestConnectToWSURL_NoTargetID(t *testing.T) {
	wsURL, cleanup := mockCDPSetup(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Target.createTarget" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{}`), // missing targetId
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	defer cleanup()

	b := New()
	err := b.connectToWSURL(nil, wsURL)
	if err == nil {
		t.Error("expected error for missing targetId")
	}
}

func TestConnectToWSURL_SessionError(t *testing.T) {
	wsURL, cleanup := mockCDPSetup(func(msg cdp.Message) *cdp.Message {
		switch msg.Method {
		case "Target.createTarget":
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"targetId":"page-1"}`),
			}
		case "Target.attachToTarget":
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "Attach failed"},
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	defer cleanup()

	b := New()
	err := b.connectToWSURL(nil, wsURL)
	if err == nil {
		t.Error("expected error for session creation failure")
	}
}

func TestSetup_BrowserPathOverride(t *testing.T) {
	b := New()
	ctx := context.Background()
	// Override browser_path with a nonexistent binary; should fail at launch.
	err := b.Setup(ctx, map[string]any{
		"browser_path": "/nonexistent/browser",
	})
	if err == nil {
		t.Error("expected error for nonexistent browser path")
	}
}

func TestSetup_LaunchFailure(t *testing.T) {
	b := New()
	ctx := context.Background()
	// Use a valid program that won't output a WebSocket URL.
	err := b.Setup(ctx, map[string]any{
		"browser_path": "/bin/echo",
	})
	if err == nil {
		t.Error("expected error when browser doesn't output WebSocket URL")
	}
}

func TestSetup_LaunchSuccessConnectFailure(t *testing.T) {
	// Create a script that outputs a ws:// URL to a non-existent server.
	tmpDir := t.TempDir()
	script := tmpDir + "/fake-browser.sh"
	os.WriteFile(script, []byte("#!/bin/sh\necho 'DevTools listening on ws://127.0.0.1:1/devtools/browser/fake' >&2\nsleep 10\n"), 0755)

	b := New()
	ctx := context.Background()
	err := b.Setup(ctx, map[string]any{
		"browser_path": script,
	})
	if err == nil {
		t.Error("expected error when CDP connection fails")
		b.Teardown(ctx)
	}
	// Process should be cleaned up.
	if b.process != nil {
		b.process.kill()
	}
}

func TestSetup_BrowserPathResolveError(t *testing.T) {
	b := New()
	b.browserType = "invalid" // Force invalid type past parseConfig.
	ctx := context.Background()
	err := b.Setup(ctx, map[string]any{})
	if err == nil {
		t.Error("expected error for invalid browser type in BrowserPath")
	}
}

func TestSetup_SuccessfulLaunchButConnectFails(t *testing.T) {
	// Browser launches successfully but connectToWSURL fails.
	// This tests the proc.kill() and b.process = nil cleanup path.
	tmpDir := t.TempDir()
	script := tmpDir + "/fake-browser2.sh"
	os.WriteFile(script, []byte("#!/bin/sh\necho 'DevTools listening on ws://127.0.0.1:1/devtools/browser/test' >&2\nsleep 10\n"), 0755)

	b := New()
	ctx := context.Background()
	err := b.Setup(ctx, map[string]any{
		"browser_path": script,
	})
	if err == nil {
		t.Error("expected error")
		b.Teardown(ctx)
	}
	if b.process != nil {
		t.Error("process should be cleaned up on connect failure")
		b.process.kill()
	}
}

func TestSetup_FullSuccess(t *testing.T) {
	// Use the mock server to simulate a full successful Setup.
	wsURL, srvCleanup := mockCDPBrowserServer(nil)
	defer srvCleanup()

	// Create a script that outputs the mock server's URL.
	tmpDir := t.TempDir()
	script := tmpDir + "/mock-browser.sh"
	os.WriteFile(script, []byte("#!/bin/sh\necho 'DevTools listening on "+wsURL+"' >&2\nsleep 60\n"), 0755)

	b := New()
	ctx := context.Background()
	err := b.Setup(ctx, map[string]any{
		"browser_path": script,
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer b.Teardown(ctx)

	if b.client == nil {
		t.Error("client should not be nil")
	}
	if b.session == nil {
		t.Error("session should not be nil")
	}
	if b.process == nil {
		t.Error("process should not be nil")
	}
}

func TestConnectToWSURL_EnableDomainError(t *testing.T) {
	wsURL, cleanup := mockCDPSetup(func(msg cdp.Message) *cdp.Message {
		switch msg.Method {
		case "Target.createTarget":
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{"targetId":"page-1"}`)}
		case "Target.attachToTarget":
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{"sessionId":"sess-1"}`)}
		case "Page.enable":
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "Page.enable failed"},
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	defer cleanup()

	b := New()
	err := b.connectToWSURL(nil, wsURL)
	if err == nil {
		t.Error("expected error for domain enable failure")
	}
	if b.client != nil {
		t.Error("client should be nil on error")
	}
	if b.session != nil {
		t.Error("session should be nil on error")
	}
}
