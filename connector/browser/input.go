package browser

import (
	"fmt"

	"github.com/scrutineer/scrutineer/connector/browser/cdp"
	"github.com/scrutineer/scrutineer/connector/browser/selector"
)

// clickElement finds an element by selector and dispatches a click event.
func clickElement(session *cdp.Session, selectorType, selectorValue string) error {
	// Get the element's bounding box center for click coordinates.
	expr := resolveSelector(selectorType, selectorValue)
	jsExpr := fmt.Sprintf(`(function() {
		var el = %s;
		if (!el) return null;
		var rect = el.getBoundingClientRect();
		return {x: rect.x + rect.width / 2, y: rect.y + rect.height / 2};
	})()`, expr)

	val, err := pageEvaluate(session, jsExpr, true)
	if err != nil {
		return fmt.Errorf("browser: click: %w", err)
	}
	if val == nil {
		return fmt.Errorf("browser: click: element not found: %s(%s)", selectorType, selectorValue)
	}

	coords, ok := val.(map[string]any)
	if !ok {
		return fmt.Errorf("browser: click: invalid coordinates")
	}

	x, xOk := toFloat64(coords["x"])
	y, yOk := toFloat64(coords["y"])
	if !xOk || !yOk {
		return fmt.Errorf("browser: click: invalid coordinate values")
	}

	// Dispatch mouse events: move, press, release.
	for _, action := range []string{"mouseMoved", "mousePressed", "mouseReleased"} {
		params := map[string]any{
			"type": action,
			"x":    x,
			"y":    y,
		}
		if action == "mousePressed" || action == "mouseReleased" {
			params["button"] = "left"
			params["clickCount"] = 1
		}
		if _, err := session.Send("Input.dispatchMouseEvent", params); err != nil {
			return fmt.Errorf("browser: click %s: %w", action, err)
		}
	}

	return nil
}

// typeText focuses an element and dispatches key events for each character.
func typeText(session *cdp.Session, selectorType, selectorValue, text string) error {
	// Focus the element first.
	if err := focusElement(session, selectorType, selectorValue); err != nil {
		return err
	}

	// Type each character via Input.dispatchKeyEvent.
	for _, ch := range text {
		charStr := string(ch)
		for _, evType := range []string{"keyDown", "keyUp"} {
			params := map[string]any{
				"type": evType,
				"text": charStr,
			}
			if _, err := session.Send("Input.dispatchKeyEvent", params); err != nil {
				return fmt.Errorf("browser: type key %q: %w", charStr, err)
			}
		}
	}

	return nil
}

// fillElement clears an input element and sets its value directly.
func fillElement(session *cdp.Session, selectorType, selectorValue, value string) error {
	expr := resolveSelector(selectorType, selectorValue)
	jsExpr := fmt.Sprintf(`(function() {
		var el = %s;
		if (!el) return false;
		el.focus();
		el.value = %s;
		el.dispatchEvent(new Event('input', {bubbles: true}));
		el.dispatchEvent(new Event('change', {bubbles: true}));
		return true;
	})()`, expr, selector.Quote(value))

	val, err := pageEvaluate(session, jsExpr, true)
	if err != nil {
		return fmt.Errorf("browser: fill: %w", err)
	}

	if val != true {
		return fmt.Errorf("browser: fill: element not found: %s(%s)", selectorType, selectorValue)
	}

	return nil
}

// selectOption selects an option in a <select> element by value.
func selectOption(session *cdp.Session, selectorType, selectorValue, optionValue string) error {
	expr := resolveSelector(selectorType, selectorValue)
	jsExpr := fmt.Sprintf(`(function() {
		var el = %s;
		if (!el || el.tagName !== 'SELECT') return false;
		el.value = %s;
		el.dispatchEvent(new Event('input', {bubbles: true}));
		el.dispatchEvent(new Event('change', {bubbles: true}));
		return true;
	})()`, expr, selector.Quote(optionValue))

	val, err := pageEvaluate(session, jsExpr, true)
	if err != nil {
		return fmt.Errorf("browser: select: %w", err)
	}

	if val != true {
		return fmt.Errorf("browser: select: element not found or not a select: %s(%s)",
			selectorType, selectorValue)
	}

	return nil
}

// focusElement focuses an element found by selector.
func focusElement(session *cdp.Session, selectorType, selectorValue string) error {
	objectID, err := findElement(session, selectorType, selectorValue)
	if err != nil {
		return err
	}

	_, err = session.Send("DOM.focus", map[string]any{
		"objectId": objectID,
	})
	if err != nil {
		return fmt.Errorf("browser: focus: %w", err)
	}

	return nil
}
