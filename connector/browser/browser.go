// Package browser implements a browser automation connector using the Chrome
// DevTools Protocol (CDP). It provides headless browser control for Chromium,
// Firefox, and WebKit, with element selection, page navigation, screenshots,
// and JavaScript evaluation capabilities.
package browser

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/scrutineer/scrutineer/connector/browser/cdp"
	"github.com/scrutineer/scrutineer/connector/browser/install"
	"github.com/scrutineer/scrutineer/core/connector"
)

// BrowserConnector implements the connector.Connector interface for browser
// automation via CDP.
type BrowserConnector struct {
	browserType string
	headless    bool
	extraArgs   []string
	process     *browserProcess
	client      *cdp.Client
	session     *cdp.Session
	manager     *install.Manager
}

// New creates a new BrowserConnector.
func New() *BrowserConnector {
	return &BrowserConnector{
		browserType: "chromium",
		headless:    true,
		manager:     install.NewManager(install.DefaultBaseDir()),
	}
}

// Name returns the connector identifier.
func (b *BrowserConnector) Name() string {
	return "browser"
}

// parseConfig applies configuration values to the connector.
func (b *BrowserConnector) parseConfig(config map[string]any) error {
	if bt, ok := config["browser"].(string); ok {
		switch bt {
		case "chromium", "firefox", "webkit":
			b.browserType = bt
		default:
			return fmt.Errorf("browser: unsupported browser type: %s", bt)
		}
	}

	if hl, ok := config["headless"].(bool); ok {
		b.headless = hl
	}

	if args, ok := config["args"].([]string); ok {
		b.extraArgs = args
	}
	// Also handle []any from JSON/YAML parsing.
	if args, ok := config["args"].([]any); ok {
		for _, a := range args {
			if s, ok := a.(string); ok {
				b.extraArgs = append(b.extraArgs, s)
			}
		}
	}

	return nil
}

// connectToWSURL connects the CDP client to the given WebSocket URL, creates a
// page target, and enables the required CDP domains.
func (b *BrowserConnector) connectToWSURL(ctx context.Context, wsURL string) error {
	client, err := cdp.Dial(ctx, wsURL)
	if err != nil {
		return fmt.Errorf("browser: connect: %w", err)
	}
	b.client = client

	result, err := client.Send("Target.createTarget", map[string]any{
		"url": "about:blank",
	})
	if err != nil {
		_ = client.Close()
		b.client = nil
		return fmt.Errorf("browser: create target: %w", err)
	}

	targetID, ok := result["targetId"].(string)
	if !ok {
		_ = client.Close()
		b.client = nil
		return fmt.Errorf("browser: no targetId in response")
	}

	session, err := client.NewSession(targetID)
	if err != nil {
		_ = client.Close()
		b.client = nil
		return fmt.Errorf("browser: create session: %w", err)
	}
	b.session = session

	for _, domain := range []string{"Page", "Runtime", "DOM", "Network"} {
		if _, err := session.Send(domain+".enable", nil); err != nil {
			_ = client.Close()
			b.client = nil
			b.session = nil
			return fmt.Errorf("browser: enable %s: %w", domain, err)
		}
	}

	return nil
}

// Setup initializes the browser connector with the given configuration.
func (b *BrowserConnector) Setup(ctx context.Context, config map[string]any) error {
	if err := b.parseConfig(config); err != nil {
		return err
	}

	// Resolve browser path.
	browserPath, err := b.manager.BrowserPath(b.browserType)
	if err != nil {
		return fmt.Errorf("browser: resolve path: %w", err)
	}

	// Allow override via config.
	if p, ok := config["browser_path"].(string); ok {
		browserPath = p
	}

	// Launch browser.
	proc, err := launchBrowser(ctx, browserPath, b.browserType, b.headless, b.extraArgs)
	if err != nil {
		return fmt.Errorf("browser: launch: %w", err)
	}
	b.process = proc

	// Connect to the browser's WebSocket endpoint.
	if err := b.connectToWSURL(ctx, proc.wsURL); err != nil {
		_ = proc.kill()
		b.process = nil
		return err
	}

	return nil
}

// Execute runs a single browser action step.
func (b *BrowserConnector) Execute(ctx context.Context, step connector.Step) (*connector.Result, error) {
	if b.session == nil {
		return nil, fmt.Errorf("browser: not connected")
	}

	start := time.Now()
	data, err := b.dispatch(ctx, step)
	elapsed := time.Since(start)

	if err != nil {
		return nil, err
	}

	return &connector.Result{
		Data:    data,
		Elapsed: elapsed,
		Meta: map[string]string{
			"action": step.Action,
		},
	}, nil
}

// dispatch routes a step to the appropriate handler.
func (b *BrowserConnector) dispatch(ctx context.Context, step connector.Step) (map[string]any, error) {
	p := step.Parameters

	switch step.Action {
	case "navigate":
		url, _ := p["url"].(string)
		if url == "" {
			return nil, fmt.Errorf("browser: navigate requires 'url' parameter")
		}
		return pageNavigate(b.session, url)

	case "click":
		sel, selType := extractSelector(p)
		if sel == "" {
			return nil, fmt.Errorf("browser: click requires 'selector' parameter")
		}
		err := clickElement(b.session, selType, sel)
		return nil, err

	case "type":
		sel, selType := extractSelector(p)
		text, _ := p["text"].(string)
		if sel == "" {
			return nil, fmt.Errorf("browser: type requires 'selector' parameter")
		}
		err := typeText(b.session, selType, sel, text)
		return nil, err

	case "fill":
		sel, selType := extractSelector(p)
		value, _ := p["value"].(string)
		if sel == "" {
			return nil, fmt.Errorf("browser: fill requires 'selector' parameter")
		}
		err := fillElement(b.session, selType, sel, value)
		return nil, err

	case "select":
		sel, selType := extractSelector(p)
		value, _ := p["value"].(string)
		if sel == "" {
			return nil, fmt.Errorf("browser: select requires 'selector' parameter")
		}
		err := selectOption(b.session, selType, sel, value)
		return nil, err

	case "screenshot":
		format, _ := p["format"].(string)
		quality := 0
		if q, ok := toFloat64(p["quality"]); ok {
			quality = int(q)
		}
		fullPage, _ := p["full_page"].(bool)

		data, err := pageScreenshot(b.session, format, quality, fullPage)
		if err != nil {
			return nil, err
		}

		result := map[string]any{
			"data": data,
		}

		// Optionally write to file.
		if path, ok := p["path"].(string); ok && path != "" {
			decoded, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				return nil, fmt.Errorf("browser: decode screenshot: %w", err)
			}
			if err := os.WriteFile(path, decoded, 0644); err != nil {
				return nil, fmt.Errorf("browser: write screenshot: %w", err)
			}
			result["path"] = path
		}

		return result, nil

	case "evaluate":
		expression, _ := p["expression"].(string)
		if expression == "" {
			return nil, fmt.Errorf("browser: evaluate requires 'expression' parameter")
		}
		val, err := pageEvaluate(b.session, expression, true)
		if err != nil {
			return nil, err
		}
		return map[string]any{"value": val}, nil

	case "wait_for_selector":
		sel, selType := extractSelector(p)
		if sel == "" {
			return nil, fmt.Errorf("browser: wait_for_selector requires 'selector' parameter")
		}
		timeout := step.Timeout
		if timeout == 0 {
			timeout = defaultWaitTimeout
		}
		err := waitForSelector(ctx, b.session, selType, sel, timeout)
		return nil, err

	case "get_text":
		sel, selType := extractSelector(p)
		if sel == "" {
			return nil, fmt.Errorf("browser: get_text requires 'selector' parameter")
		}
		text, err := getElementText(b.session, selType, sel)
		if err != nil {
			return nil, err
		}
		return map[string]any{"text": text}, nil

	case "get_attribute":
		sel, selType := extractSelector(p)
		attr, _ := p["attribute"].(string)
		if sel == "" {
			return nil, fmt.Errorf("browser: get_attribute requires 'selector' parameter")
		}
		if attr == "" {
			return nil, fmt.Errorf("browser: get_attribute requires 'attribute' parameter")
		}
		val, err := getElementAttribute(b.session, selType, sel, attr)
		if err != nil {
			return nil, err
		}
		return map[string]any{"value": val}, nil

	default:
		return nil, fmt.Errorf("browser: unknown action: %s", step.Action)
	}
}

// extractSelector extracts the selector value and type from parameters.
func extractSelector(params map[string]any) (string, string) {
	sel, _ := params["selector"].(string)
	selType, _ := params["selector_type"].(string)
	if selType == "" {
		selType = "css"
	}
	return sel, selType
}

// Teardown cleans up the browser process and connections.
func (b *BrowserConnector) Teardown(_ context.Context) error {
	var errs []error

	if b.session != nil {
		if err := b.session.Close(); err != nil {
			errs = append(errs, err)
		}
		b.session = nil
	}

	if b.client != nil {
		if err := b.client.Close(); err != nil {
			errs = append(errs, err)
		}
		b.client = nil
	}

	if b.process != nil {
		if err := b.process.kill(); err != nil {
			errs = append(errs, err)
		}
		b.process = nil
	}

	if len(errs) > 0 {
		return fmt.Errorf("browser: teardown errors: %v", errs)
	}
	return nil
}

// Verify interface compliance.
var _ connector.Connector = (*BrowserConnector)(nil)
