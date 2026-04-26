package browser

import (
	"fmt"

	"github.com/scrutineer/scrutineer/connector/browser/cdp"
	"github.com/scrutineer/scrutineer/connector/browser/selector"
)

// resolveSelector returns the JS expression for finding an element based on
// the selector type and value.
func resolveSelector(selectorType, value string) string {
	switch selectorType {
	case "xpath":
		return selector.XPathQueryOne(value)
	case "text":
		return selector.TextQueryOne(value)
	case "role":
		return selector.RoleQueryOne(value)
	default:
		return selector.CSSQueryOne(value)
	}
}

// findElement evaluates a selector in the page and returns the remote object ID.
func findElement(session *cdp.Session, selectorType, selectorValue string) (string, error) {
	expr := resolveSelector(selectorType, selectorValue)

	result, err := session.Send("Runtime.evaluate", map[string]any{
		"expression":    expr,
		"returnByValue": false,
	})
	if err != nil {
		return "", fmt.Errorf("browser: find element: %w", err)
	}

	remoteObj, ok := result["result"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("browser: find element: no result")
	}

	subtype, _ := remoteObj["subtype"].(string)
	if subtype == "null" {
		return "", fmt.Errorf("browser: element not found: %s(%s)", selectorType, selectorValue)
	}

	objectID, ok := remoteObj["objectId"].(string)
	if !ok {
		return "", fmt.Errorf("browser: find element: no objectId")
	}

	return objectID, nil
}

// getElementText returns the innerText of an element found by selector.
func getElementText(session *cdp.Session, selectorType, selectorValue string) (string, error) {
	expr := resolveSelector(selectorType, selectorValue)
	jsExpr := fmt.Sprintf(`(function() { var el = %s; return el ? el.innerText : null; })()`, expr)

	val, err := pageEvaluate(session, jsExpr, true)
	if err != nil {
		return "", err
	}

	if val == nil {
		return "", fmt.Errorf("browser: element not found: %s(%s)", selectorType, selectorValue)
	}

	text, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("browser: innerText is not a string")
	}

	return text, nil
}

// getElementAttribute returns the value of an attribute on an element.
func getElementAttribute(session *cdp.Session, selectorType, selectorValue, attribute string) (string, error) {
	expr := resolveSelector(selectorType, selectorValue)
	jsExpr := fmt.Sprintf(
		`(function() { var el = %s; return el ? el.getAttribute(%s) : null; })()`,
		expr, selector.Quote(attribute),
	)

	val, err := pageEvaluate(session, jsExpr, true)
	if err != nil {
		return "", err
	}

	if val == nil {
		return "", fmt.Errorf("browser: element not found or attribute missing: %s(%s).%s",
			selectorType, selectorValue, attribute)
	}

	text, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("browser: attribute value is not a string")
	}

	return text, nil
}
