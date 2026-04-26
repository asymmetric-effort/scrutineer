// Package selector provides different element selection strategies for browser
// automation. Each strategy generates JavaScript expressions that can be
// evaluated in a page context via the CDP Runtime.evaluate command.
package selector

import (
	"fmt"
	"strings"
)

// CSSQueryOne returns a JS expression that finds a single element by CSS selector.
func CSSQueryOne(sel string) string {
	return fmt.Sprintf("document.querySelector(%s)", Quote(sel))
}

// CSSQueryAll returns a JS expression that finds all elements matching a CSS selector.
func CSSQueryAll(sel string) string {
	return fmt.Sprintf("Array.from(document.querySelectorAll(%s))", Quote(sel))
}

// Quote produces a JSON-safe JavaScript string literal.
func Quote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return `"` + s + `"`
}
