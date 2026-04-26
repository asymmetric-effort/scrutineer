package browser

import (
	"encoding/json"
	"fmt"

	"github.com/scrutineer/scrutineer/connector/browser/cdp"
)

// pageNavigate navigates the page to the given URL.
func pageNavigate(session *cdp.Session, url string) (map[string]any, error) {
	result, err := session.Send("Page.navigate", map[string]any{
		"url": url,
	})
	if err != nil {
		return nil, fmt.Errorf("browser: navigate: %w", err)
	}

	// Check for navigation error in the response.
	if errText, ok := result["errorText"]; ok {
		if s, ok := errText.(string); ok && s != "" {
			return nil, fmt.Errorf("browser: navigate error: %s", s)
		}
	}

	return map[string]any{
		"url":     url,
		"frameId": result["frameId"],
	}, nil
}

// pageEvaluate evaluates a JavaScript expression in the page context.
func pageEvaluate(session *cdp.Session, expression string, returnByValue bool) (any, error) {
	params := map[string]any{
		"expression":    expression,
		"returnByValue": returnByValue,
		"awaitPromise":  true,
	}

	result, err := session.Send("Runtime.evaluate", params)
	if err != nil {
		return nil, fmt.Errorf("browser: evaluate: %w", err)
	}

	// Check for exception.
	if exInfo, ok := result["exceptionDetails"]; ok {
		if details, ok := exInfo.(map[string]any); ok {
			text := "unknown error"
			if ex, ok := details["text"].(string); ok {
				text = ex
			}
			return nil, fmt.Errorf("browser: evaluate exception: %s", text)
		}
	}

	remoteObj, ok := result["result"].(map[string]any)
	if !ok {
		return nil, nil
	}

	if returnByValue {
		return remoteObj["value"], nil
	}

	return remoteObj, nil
}

// pageScreenshot captures a screenshot and returns base64-encoded data.
func pageScreenshot(session *cdp.Session, format string, quality int, fullPage bool) (string, error) {
	if format == "" {
		format = "png"
	}

	params := map[string]any{
		"format": format,
	}

	if format == "jpeg" && quality > 0 {
		params["quality"] = quality
	}

	if fullPage {
		// Get the full page dimensions.
		metrics, err := session.Send("Page.getLayoutMetrics", nil)
		if err == nil {
			if contentSize, ok := metrics["contentSize"].(map[string]any); ok {
				width, _ := toFloat64(contentSize["width"])
				height, _ := toFloat64(contentSize["height"])
				if width > 0 && height > 0 {
					params["clip"] = map[string]any{
						"x":      0,
						"y":      0,
						"width":  width,
						"height": height,
						"scale":  1,
					}
				}
			}
		}
	}

	result, err := session.Send("Page.captureScreenshot", params)
	if err != nil {
		return "", fmt.Errorf("browser: screenshot: %w", err)
	}

	data, ok := result["data"].(string)
	if !ok {
		return "", fmt.Errorf("browser: screenshot: no data in response")
	}

	return data, nil
}

// toFloat64 converts a JSON number value to float64.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
