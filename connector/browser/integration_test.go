package browser

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/scrutineer/scrutineer/connector/browser/cdp"
	"github.com/scrutineer/scrutineer/core/connector"
)

func TestIntegration_Navigate(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Page.navigate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"frameId":"frame-1","loaderId":"loader-1"}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	result, err := b.Execute(ctx, connector.Step{
		Action:     "navigate",
		Parameters: map[string]any{"url": "http://example.com"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Data["url"] != "http://example.com" {
		t.Errorf("url = %v, want http://example.com", result.Data["url"])
	}
	if result.Meta["action"] != "navigate" {
		t.Errorf("action = %v, want navigate", result.Meta["action"])
	}
	if result.Elapsed <= 0 {
		t.Error("elapsed should be > 0")
	}
}

func TestIntegration_Navigate_Error(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Page.navigate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"frameId":"frame-1","errorText":"net::ERR_CONNECTION_REFUSED"}`),
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
		Action:     "navigate",
		Parameters: map[string]any{"url": "http://invalid.example"},
	})
	if err == nil {
		t.Error("expected error for navigation error")
	}
}

func TestIntegration_Evaluate(t *testing.T) {
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

	ctx := context.Background()
	result, err := b.Execute(ctx, connector.Step{
		Action:     "evaluate",
		Parameters: map[string]any{"expression": "1 + 1"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Data["value"] != float64(42) {
		t.Errorf("value = %v, want 42", result.Data["value"])
	}
}

func TestIntegration_Evaluate_Exception(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID: msg.ID,
				Result: json.RawMessage(`{
					"result":{"type":"object","subtype":"error"},
					"exceptionDetails":{"text":"ReferenceError: foo is not defined"}
				}`),
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
		Action:     "evaluate",
		Parameters: map[string]any{"expression": "foo()"},
	})
	if err == nil {
		t.Error("expected error for evaluation exception")
	}
}

func TestIntegration_Screenshot(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Page.captureScreenshot" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"data":"iVBORw0KGgoAAAANSUhEUg=="}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	result, err := b.Execute(ctx, connector.Step{
		Action:     "screenshot",
		Parameters: map[string]any{},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Data["data"] == nil || result.Data["data"] == "" {
		t.Error("screenshot data should not be empty")
	}
}

func TestIntegration_Screenshot_WithFile(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Page.captureScreenshot" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"data":"iVBORw0KGgoAAAANSUhEUg=="}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	tmpFile := t.TempDir() + "/screenshot.png"
	ctx := context.Background()
	result, err := b.Execute(ctx, connector.Step{
		Action: "screenshot",
		Parameters: map[string]any{
			"path": tmpFile,
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Data["path"] != tmpFile {
		t.Errorf("path = %v, want %s", result.Data["path"], tmpFile)
	}
}

func TestIntegration_Screenshot_Error(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Page.captureScreenshot" {
			return &cdp.Message{
				ID:    msg.ID,
				Error: &cdp.ErrorInfo{Code: -32000, Message: "screenshot failed"},
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
		Action:     "screenshot",
		Parameters: map[string]any{},
	})
	if err == nil {
		t.Error("expected error from screenshot failure")
	}
}

func TestIntegration_Screenshot_FullPage(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		switch msg.Method {
		case "Page.getLayoutMetrics":
			return &cdp.Message{
				ID: msg.ID,
				Result: json.RawMessage(`{
					"contentSize":{"width":1920,"height":3000},
					"layoutViewport":{"pageX":0,"pageY":0,"clientWidth":1920,"clientHeight":1080}
				}`),
			}
		case "Page.captureScreenshot":
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"data":"iVBORw0KGgoAAAANSUhEUg=="}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	result, err := b.Execute(ctx, connector.Step{
		Action: "screenshot",
		Parameters: map[string]any{
			"full_page": true,
			"format":    "jpeg",
			"quality":   float64(80),
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Data["data"] == nil {
		t.Error("data should not be nil")
	}
}

func TestIntegration_Click(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		switch msg.Method {
		case "Runtime.evaluate":
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"object","value":{"x":100,"y":200}}}`),
			}
		case "Input.dispatchMouseEvent":
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	_, err = b.Execute(ctx, connector.Step{
		Action:     "click",
		Parameters: map[string]any{"selector": "#btn"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestIntegration_Click_ElementNotFound(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"object","subtype":"null","value":null}}`),
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
		Action:     "click",
		Parameters: map[string]any{"selector": "#nonexistent"},
	})
	if err == nil {
		t.Error("expected error for nonexistent element")
	}
}

func TestIntegration_Type(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		switch msg.Method {
		case "Runtime.evaluate":
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"object","objectId":"obj-1"}}`),
			}
		case "DOM.focus":
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
		case "Input.dispatchKeyEvent":
			return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	_, err = b.Execute(ctx, connector.Step{
		Action:     "type",
		Parameters: map[string]any{"selector": "#input", "text": "hello"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestIntegration_Fill(t *testing.T) {
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
	_, err = b.Execute(ctx, connector.Step{
		Action:     "fill",
		Parameters: map[string]any{"selector": "#input", "value": "test value"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestIntegration_Fill_NotFound(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"boolean","value":false}}`),
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
		Action:     "fill",
		Parameters: map[string]any{"selector": "#missing", "value": "test"},
	})
	if err == nil {
		t.Error("expected error for fill on missing element")
	}
}

func TestIntegration_Select(t *testing.T) {
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
	_, err = b.Execute(ctx, connector.Step{
		Action:     "select",
		Parameters: map[string]any{"selector": "select#country", "value": "us"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestIntegration_Select_NotSelect(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"boolean","value":false}}`),
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
		Action:     "select",
		Parameters: map[string]any{"selector": "#div", "value": "us"},
	})
	if err == nil {
		t.Error("expected error for select on non-select element")
	}
}

func TestIntegration_GetText(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"string","value":"Hello World"}}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	result, err := b.Execute(ctx, connector.Step{
		Action:     "get_text",
		Parameters: map[string]any{"selector": "#heading"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Data["text"] != "Hello World" {
		t.Errorf("text = %v, want Hello World", result.Data["text"])
	}
}

func TestIntegration_GetText_NotFound(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"object","subtype":"null","value":null}}`),
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
		Action:     "get_text",
		Parameters: map[string]any{"selector": "#missing"},
	})
	if err == nil {
		t.Error("expected error for missing element")
	}
}

func TestIntegration_GetAttribute(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"string","value":"http://example.com"}}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	result, err := b.Execute(ctx, connector.Step{
		Action: "get_attribute",
		Parameters: map[string]any{
			"selector":  "a#link",
			"attribute": "href",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Data["value"] != "http://example.com" {
		t.Errorf("value = %v, want http://example.com", result.Data["value"])
	}
}

func TestIntegration_GetAttribute_NotFound(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"object","subtype":"null","value":null}}`),
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
		Action: "get_attribute",
		Parameters: map[string]any{
			"selector":  "#missing",
			"attribute": "href",
		},
	})
	if err == nil {
		t.Error("expected error for missing element")
	}
}

func TestIntegration_WaitForSelector_Found(t *testing.T) {
	callCount := 0
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			callCount++
			if callCount >= 2 {
				return &cdp.Message{
					ID:     msg.ID,
					Result: json.RawMessage(`{"result":{"type":"boolean","value":true}}`),
				}
			}
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"boolean","value":false}}`),
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
		Action:     "wait_for_selector",
		Parameters: map[string]any{"selector": "#delayed"},
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestIntegration_WaitForSelector_DefaultTimeout(t *testing.T) {
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
	// No Timeout set - should use default.
	_, err = b.Execute(ctx, connector.Step{
		Action:     "wait_for_selector",
		Parameters: map[string]any{"selector": "#quick"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestIntegration_WaitForSelector_Timeout(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"boolean","value":false}}`),
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
		Action:     "wait_for_selector",
		Parameters: map[string]any{"selector": "#never"},
		Timeout:    300 * time.Millisecond,
	})
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestIntegration_FindElement(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"object","objectId":"obj-abc"}}`),
			}
		}
		return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer cleanup()

	objID, err := findElement(b.session, "css", "#test")
	if err != nil {
		t.Fatalf("findElement: %v", err)
	}
	if objID != "obj-abc" {
		t.Errorf("objectId = %q, want %q", objID, "obj-abc")
	}
}

func TestIntegration_FindElement_NotFound(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
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

	_, err = findElement(b.session, "css", "#missing")
	if err == nil {
		t.Error("expected error for missing element")
	}
}

func TestIntegration_FindElement_NoObjectID(t *testing.T) {
	b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
		if msg.Method == "Runtime.evaluate" {
			return &cdp.Message{
				ID:     msg.ID,
				Result: json.RawMessage(`{"result":{"type":"string","value":"not an element"}}`),
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
		t.Error("expected error when no objectId")
	}
}

func TestIntegration_Teardown(t *testing.T) {
	wsURL, srvCleanup := mockCDPBrowserServer(nil)
	defer srvCleanup()

	b := New()
	client, err := cdp.Dial(nil, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	b.client = client

	result, _ := client.Send("Target.createTarget", map[string]any{"url": "about:blank"})
	targetID := result["targetId"].(string)
	session, _ := client.NewSession(targetID)
	b.session = session

	ctx := context.Background()
	err = b.Teardown(ctx)
	// May have some errors due to connection closing, but should not panic.
	_ = err

	// Verify state is cleaned up.
	if b.session != nil {
		t.Error("session should be nil after teardown")
	}
	if b.client != nil {
		t.Error("client should be nil after teardown")
	}
}

func TestIntegration_SelectorTypes(t *testing.T) {
	selectorTypes := []struct {
		selType string
		value   string
	}{
		{"css", "#btn"},
		{"xpath", "//button"},
		{"text", "Click me"},
		{"role", "button"},
	}

	for _, st := range selectorTypes {
		t.Run(st.selType, func(t *testing.T) {
			b, cleanup, err := setupConnectorWithMock(func(msg cdp.Message) *cdp.Message {
				if msg.Method == "Runtime.evaluate" {
					return &cdp.Message{
						ID:     msg.ID,
						Result: json.RawMessage(`{"result":{"type":"string","value":"found"}}`),
					}
				}
				return &cdp.Message{ID: msg.ID, Result: json.RawMessage(`{}`)}
			})
			if err != nil {
				t.Fatalf("setup: %v", err)
			}
			defer cleanup()

			ctx := context.Background()
			result, err := b.Execute(ctx, connector.Step{
				Action: "get_text",
				Parameters: map[string]any{
					"selector":      st.value,
					"selector_type": st.selType,
				},
			})
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}

			if result.Data["text"] != "found" {
				t.Errorf("text = %v, want found", result.Data["text"])
			}
		})
	}
}
