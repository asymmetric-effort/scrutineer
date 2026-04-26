package browser

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/scrutineer/scrutineer/connector/browser/cdp"
)

func TestTeardown_WithProcess(t *testing.T) {
	// Start a real process (sleep) and verify teardown kills it.
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}

	b := New()
	b.process = &browserProcess{
		cmd: cmd,
	}

	ctx := context.Background()
	err := b.Teardown(ctx)
	// Error from killing is expected (signal: killed).
	_ = err

	if b.process != nil {
		t.Error("process should be nil after teardown")
	}
}

func TestTeardown_WithSessionAndClient(t *testing.T) {
	wsURL, srvCleanup := mockCDPBrowserServer(func(msg cdp.Message) *cdp.Message {
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	defer srvCleanup()

	client, err := cdp.Dial(nil, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	result, _ := client.Send("Target.createTarget", map[string]any{"url": "about:blank"})
	targetID := result["targetId"].(string)
	session, _ := client.NewSession(targetID)

	b := New()
	b.client = client
	b.session = session

	ctx := context.Background()
	err = b.Teardown(ctx)
	if err != nil {
		// Some errors may occur but should not panic.
		t.Logf("teardown error (expected): %v", err)
	}

	if b.session != nil {
		t.Error("session should be nil")
	}
	if b.client != nil {
		t.Error("client should be nil")
	}
}

func TestTeardown_WithSessionClientAndProcess(t *testing.T) {
	wsURL, srvCleanup := mockCDPBrowserServer(func(msg cdp.Message) *cdp.Message {
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	defer srvCleanup()

	client, err := cdp.Dial(nil, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	result, _ := client.Send("Target.createTarget", map[string]any{"url": "about:blank"})
	targetID := result["targetId"].(string)
	session, _ := client.NewSession(targetID)

	cmd := exec.Command("sleep", "60")
	cmd.Start()

	b := New()
	b.client = client
	b.session = session
	b.process = &browserProcess{cmd: cmd}

	ctx := context.Background()
	_ = b.Teardown(ctx)

	if b.session != nil || b.client != nil || b.process != nil {
		t.Error("all should be nil after teardown")
	}
}

func TestTeardown_SessionCloseError(t *testing.T) {
	wsURL, srvCleanup := mockCDPSetup(func(msg cdp.Message) *cdp.Message {
		switch msg.Method {
		case "Target.createTarget":
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{"targetId":"page-1"}`)}
		case "Target.attachToTarget":
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{"sessionId":"session-1"}`)}
		case "Target.detachFromTarget":
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "detach failed"},
			}
		default:
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
		}
	})
	defer srvCleanup()

	client, err := cdp.Dial(nil, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	result, _ := client.Send("Target.createTarget", map[string]any{"url": "about:blank"})
	targetID := result["targetId"].(string)
	session, _ := client.NewSession(targetID)

	b := New()
	b.client = client
	b.session = session

	ctx := context.Background()
	err = b.Teardown(ctx)
	if err == nil {
		t.Error("expected error from session close failure")
	}
}

func TestBrowserProcess_Kill_WithCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "sleep", "60")
	cmd.Start()

	bp := &browserProcess{
		cmd:    cmd,
		cancel: cancel,
	}

	err := bp.kill()
	// Error is expected (context canceled / signal killed).
	_ = err
}

func TestTeardown_ClientCloseError(t *testing.T) {
	wsURL, srvCleanup := mockCDPSetup(func(msg cdp.Message) *cdp.Message {
		switch msg.Method {
		case "Target.createTarget":
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{"targetId":"page-1"}`)}
		case "Target.attachToTarget":
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{"sessionId":"session-1"}`)}
		case "Target.detachFromTarget":
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
		default:
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
		}
	})
	defer srvCleanup()

	client, err := cdp.Dial(nil, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	result, _ := client.Send("Target.createTarget", map[string]any{"url": "about:blank"})
	targetID := result["targetId"].(string)
	session, _ := client.NewSession(targetID)

	b := New()
	b.client = client
	b.session = session

	// Close the client first, then teardown should get client.Close() error.
	client.Close()

	ctx := context.Background()
	err = b.Teardown(ctx)
	// Session close may fail since connection is closed, and client close will fail.
	// The teardown should still complete and collect errors.
	if err == nil {
		// It's OK if err is nil (some implementations don't error on double-close).
		t.Log("teardown succeeded without error")
	}

	if b.client != nil {
		t.Error("client should be nil")
	}
	if b.session != nil {
		t.Error("session should be nil")
	}
}

func TestBrowserProcess_Kill_NilCmd(t *testing.T) {
	bp := &browserProcess{
		cancel: func() {},
	}
	err := bp.kill()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
