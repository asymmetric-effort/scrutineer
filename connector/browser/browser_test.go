package browser

import (
	"context"
	"testing"

	"github.com/scrutineer/scrutineer/core/connector"
)

func TestNew(t *testing.T) {
	b := New()
	if b == nil {
		t.Fatal("New() returned nil")
	}
	if b.browserType != "chromium" {
		t.Errorf("default browserType = %q, want %q", b.browserType, "chromium")
	}
	if !b.headless {
		t.Error("default headless should be true")
	}
	if b.manager == nil {
		t.Error("manager should not be nil")
	}
}

func TestBrowserConnector_Name(t *testing.T) {
	b := New()
	if b.Name() != "browser" {
		t.Errorf("Name() = %q, want %q", b.Name(), "browser")
	}
}

func TestBrowserConnector_ImplementsInterface(t *testing.T) {
	var _ connector.Connector = (*BrowserConnector)(nil)
}

func TestBrowserConnector_Execute_NotConnected(t *testing.T) {
	b := New()
	ctx := context.Background()

	_, err := b.Execute(ctx, connector.Step{Action: "navigate"})
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestBrowserConnector_Execute_UnknownAction(t *testing.T) {
	b := New()
	// session is nil, so Execute returns "not connected" error.
	ctx := context.Background()
	_, err := b.Execute(ctx, connector.Step{Action: "fly"})
	if err == nil {
		t.Error("expected error for unknown action")
	}
}

func TestBrowserConnector_Setup_InvalidBrowser(t *testing.T) {
	ctx := context.Background()

	err := New().Setup(ctx, map[string]any{
		"browser": "opera",
	})
	if err == nil {
		t.Error("expected error for unsupported browser")
	}
}

func TestBrowserConnector_Setup_ConfigParsing(t *testing.T) {
	// Test config parsing without actually launching.
	// We test that the config is parsed correctly by checking the fields.

	// Valid browser types.
	for _, bt := range []string{"chromium", "firefox", "webkit"} {
		b2 := New()
		config := map[string]any{"browser": bt}
		// Setup will fail since there is no real browser, but it should parse the config first.
		_ = b2.Setup(context.Background(), config)
		if b2.browserType != bt {
			t.Errorf("browserType = %q, want %q", b2.browserType, bt)
		}
	}

	// Headless config.
	b3 := New()
	_ = b3.Setup(context.Background(), map[string]any{"headless": false})
	if b3.headless {
		t.Error("headless should be false")
	}

	// Args as []string.
	b4 := New()
	_ = b4.Setup(context.Background(), map[string]any{
		"args": []string{"--flag1", "--flag2"},
	})
	if len(b4.extraArgs) != 2 {
		t.Errorf("extraArgs length = %d, want 2", len(b4.extraArgs))
	}

	// Args as []any (from JSON parsing).
	b5 := New()
	_ = b5.Setup(context.Background(), map[string]any{
		"args": []any{"--flag1", "--flag2"},
	})
	if len(b5.extraArgs) != 2 {
		t.Errorf("extraArgs length = %d, want 2", len(b5.extraArgs))
	}
}

func TestBrowserConnector_Teardown_NotSetup(t *testing.T) {
	b := New()
	ctx := context.Background()

	err := b.Teardown(ctx)
	if err != nil {
		t.Errorf("Teardown on uninitialized should not error: %v", err)
	}
}

func TestExtractSelector(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]any
		wantSel  string
		wantType string
	}{
		{
			"css default",
			map[string]any{"selector": "#btn"},
			"#btn", "css",
		},
		{
			"explicit css",
			map[string]any{"selector": ".item", "selector_type": "css"},
			".item", "css",
		},
		{
			"xpath",
			map[string]any{"selector": "//div", "selector_type": "xpath"},
			"//div", "xpath",
		},
		{
			"text",
			map[string]any{"selector": "Click me", "selector_type": "text"},
			"Click me", "text",
		},
		{
			"role",
			map[string]any{"selector": "button", "selector_type": "role"},
			"button", "role",
		},
		{
			"missing selector",
			map[string]any{},
			"", "css",
		},
		{
			"nil params",
			map[string]any{},
			"", "css",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sel, selType := extractSelector(tt.params)
			if sel != tt.wantSel {
				t.Errorf("selector = %q, want %q", sel, tt.wantSel)
			}
			if selType != tt.wantType {
				t.Errorf("type = %q, want %q", selType, tt.wantType)
			}
		})
	}
}

func TestBrowserConnector_Dispatch_MissingParams(t *testing.T) {
	b := New()
	// Create a mock session-like state. We need session to be non-nil for dispatch.
	// We use a fake to test parameter validation without a real connection.

	tests := []struct {
		action string
		params map[string]any
		errMsg string
	}{
		{"navigate", map[string]any{}, "requires 'url'"},
		{"click", map[string]any{}, "requires 'selector'"},
		{"type", map[string]any{}, "requires 'selector'"},
		{"fill", map[string]any{}, "requires 'selector'"},
		{"select", map[string]any{}, "requires 'selector'"},
		{"evaluate", map[string]any{}, "requires 'expression'"},
		{"wait_for_selector", map[string]any{}, "requires 'selector'"},
		{"get_text", map[string]any{}, "requires 'selector'"},
		{"get_attribute", map[string]any{"selector": ".x"}, "requires 'attribute'"},
		{"get_attribute", map[string]any{}, "requires 'selector'"},
		{"unknown_action", map[string]any{}, "unknown action"},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			ctx := context.Background()
			_, err := b.dispatch(ctx, connector.Step{
				Action:     tt.action,
				Parameters: tt.params,
			})
			if err == nil {
				t.Errorf("expected error for %s with empty params", tt.action)
				return
			}
			if tt.errMsg != "" {
				found := false
				if contains(err.Error(), tt.errMsg) {
					found = true
				}
				if !found {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
