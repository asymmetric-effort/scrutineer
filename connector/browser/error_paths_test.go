package browser

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/scrutineer/scrutineer/connector/browser/cdp"
	"github.com/scrutineer/scrutineer/core/connector"
)

func TestPageNavigate_CDPError(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Page.navigate" {
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "Cannot navigate"},
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	_, err = pageNavigate(b.session, "http://example.com")
	if err == nil {
		t.Error("expected error")
	}
}

func TestPageEvaluate_CDPError(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "Evaluation failed"},
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	_, err = pageEvaluate(b.session, "bad()", true)
	if err == nil {
		t.Error("expected error")
	}
}

func TestPageEvaluate_NoResult(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	val, err := pageEvaluate(b.session, "undefined", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestPageEvaluate_ReturnByRef(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"object","objectId":"obj-1"}}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	val, err := pageEvaluate(b.session, "document.body", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", val)
	}
	if m["objectId"] != "obj-1" {
		t.Errorf("objectId = %v, want obj-1", m["objectId"])
	}
}

func TestPageScreenshot_CDPError(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Page.captureScreenshot" {
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "Screenshot failed"},
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	_, err = pageScreenshot(b.session, "png", 0, false)
	if err == nil {
		t.Error("expected error")
	}
}

func TestPageScreenshot_NoData(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Page.captureScreenshot" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	_, err = pageScreenshot(b.session, "png", 0, false)
	if err == nil {
		t.Error("expected error for missing data")
	}
}

func TestClickElement_CDPError(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "Eval error"},
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	err = clickElement(b.session, "css", "#btn")
	if err == nil {
		t.Error("expected error")
	}
}

func TestClickElement_InvalidCoords(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"object","value":{"x":"not-a-number","y":"bad"}}}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	err = clickElement(b.session, "css", "#btn")
	if err == nil {
		t.Error("expected error for invalid coordinates")
	}
}

func TestClickElement_NonMapCoords(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"string","value":"not a map"}}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	err = clickElement(b.session, "css", "#btn")
	if err == nil {
		t.Error("expected error for non-map coordinates")
	}
}

func TestTypeText_FocusError(t *testing.T) {
	callCount := 0
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			callCount++
			// Return null to simulate element not found.
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"object","subtype":"null"}}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	err = typeText(b.session, "css", "#missing", "hello")
	if err == nil {
		t.Error("expected error for missing element")
	}
}

func TestFocusElement_DOMFocusError(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"object","objectId":"obj-1"}}`),
			}
		}
		if msg.Method == "DOM.focus" {
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "Cannot focus"},
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	err = focusElement(b.session, "css", "#input")
	if err == nil {
		t.Error("expected error for focus failure")
	}
}

func TestGetElementText_NonStringValue(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"number","value":42}}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	_, err = getElementText(b.session, "css", "#num")
	if err == nil {
		t.Error("expected error for non-string text")
	}
}

func TestGetElementAttribute_NonStringValue(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"number","value":42}}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	_, err = getElementAttribute(b.session, "css", "#el", "data-count")
	if err == nil {
		t.Error("expected error for non-string attribute")
	}
}

func TestFindElement_CDPError(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "Eval error"},
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	_, err = findElement(b.session, "css", "#test")
	if err == nil {
		t.Error("expected error")
	}
}

func TestFindElement_NoResult(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			// Return with no result key.
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"notResult":"foo"}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	_, err = findElement(b.session, "css", "#test")
	if err == nil {
		t.Error("expected error for no result")
	}
}

func TestScreenshot_InvalidBase64_WriteFile(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Page.captureScreenshot" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"data":"!!!not-base64!!!"}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	_, err = b.Execute(ctx, connector.Step{
		Action: "screenshot",
		Parameters: map[string]any{
			"path": t.TempDir() + "/test.png",
		},
	})
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestScreenshot_InvalidPath(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Page.captureScreenshot" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"data":"aGVsbG8="}`), // "hello" in base64
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	_, err = b.Execute(ctx, connector.Step{
		Action: "screenshot",
		Parameters: map[string]any{
			"path": "/nonexistent/path/screenshot.png",
		},
	})
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestWaitForNavigation_Success(t *testing.T) {
	// Create a mock server that sends a Page.loadEventFired event after a request.
	wsURL, connCh, srvCleanup := mockWSServerForBrowser(t)
	defer srvCleanup()

	client, err := cdp.Dial(nil, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	serverConn := <-connCh

	ctx := context.Background()

	done := make(chan error, 1)
	go func() {
		done <- waitForNavigation(ctx, client, 2*time.Second)
	}()

	// Give waitForNavigation time to subscribe.
	time.Sleep(50 * time.Millisecond)

	// Send the event from the server.
	event := `{"method":"Page.loadEventFired","params":{"timestamp":1234.5}}`
	writeWSFrame(serverConn, []byte(event))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("waitForNavigation: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("test took too long")
	}
	serverConn.Close()
}

func TestWaitForNavigation_DefaultTimeout(t *testing.T) {
	wsURL, srvCleanup := mockCDPBrowserServer(nil)
	defer srvCleanup()

	client, err := cdp.Dial(nil, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Pass 0 timeout to use default; context will cancel first.
	err = waitForNavigation(ctx, client, 0)
	if err == nil {
		t.Error("expected error")
	}
}

func TestWaitForNavigation_Timeout(t *testing.T) {
	wsURL, srvCleanup := mockCDPBrowserServer(nil)
	defer srvCleanup()

	client, err := cdp.Dial(nil, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	err = waitForNavigation(ctx, client, 200*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestWaitForNavigation_ContextCancel(t *testing.T) {
	wsURL, srvCleanup := mockCDPBrowserServer(nil)
	defer srvCleanup()

	client, err := cdp.Dial(nil, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = waitForNavigation(ctx, client, 5*time.Second)
	if err == nil {
		t.Error("expected context canceled error")
	}
}

func TestClickElement_MouseEventError(t *testing.T) {
	callCount := 0
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"object","value":{"x":100,"y":200}}}`),
			}
		}
		if msg.Method == "Input.dispatchMouseEvent" {
			callCount++
			if callCount == 2 {
				return &cdp.Message{
					ID:    msg.ID,
					Error: &cdp.ErrorInfo{Code: -32000, Message: "Mouse event failed"},
				}
			}
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	err = clickElement(b.session, "css", "#btn")
	if err == nil {
		t.Error("expected error for mouse event failure")
	}
}

func TestTypeText_KeyEventError(t *testing.T) {
	keyEventCount := 0
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"object","objectId":"obj-1"}}`),
			}
		}
		if msg.Method == "DOM.focus" {
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
		}
		if msg.Method == "Input.dispatchKeyEvent" {
			keyEventCount++
			if keyEventCount == 3 {
				return &cdp.Message{
					ID:    msg.ID,
					Error: &cdp.ErrorInfo{Code: -32000, Message: "Key event failed"},
				}
			}
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	err = typeText(b.session, "css", "#input", "hi")
	if err == nil {
		t.Error("expected error for key event failure")
	}
}

func TestFillElement_CDPError(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "Eval error"},
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	err = fillElement(b.session, "css", "#input", "value")
	if err == nil {
		t.Error("expected error")
	}
}

func TestSelectOption_CDPError(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "Eval error"},
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	err = selectOption(b.session, "css", "#select", "value")
	if err == nil {
		t.Error("expected error")
	}
}

func TestGetElementText_CDPError(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "Eval error"},
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	_, err = getElementText(b.session, "css", "#el")
	if err == nil {
		t.Error("expected error")
	}
}

func TestGetElementAttribute_CDPError(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "Eval error"},
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	_, err = getElementAttribute(b.session, "css", "#el", "href")
	if err == nil {
		t.Error("expected error")
	}
}

func TestPageNavigate_Success(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Page.navigate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"frameId":"f1"}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	result, err := pageNavigate(b.session, "http://example.com")
	if err != nil {
		t.Fatalf("pageNavigate: %v", err)
	}
	if result["url"] != "http://example.com" {
		t.Errorf("url = %v", result["url"])
	}
}

func TestWaitForSelector_DefaultTimeout(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"boolean","value":true}}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	// Pass 0 timeout, should use default.
	err = waitForSelector(ctx, b.session, "css", "#test", 0)
	if err != nil {
		t.Fatalf("waitForSelector: %v", err)
	}
}
