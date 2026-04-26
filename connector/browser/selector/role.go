package selector

import "fmt"

// RoleQueryOne returns a JS expression that finds the first element with the
// given ARIA role attribute.
func RoleQueryOne(role string) string {
	return fmt.Sprintf(`document.querySelector('[role=%s]')`, Quote(role))
}

// RoleQueryAll returns a JS expression that finds all elements with the given
// ARIA role attribute.
func RoleQueryAll(role string) string {
	return fmt.Sprintf(`Array.from(document.querySelectorAll('[role=%s]'))`, Quote(role))
}

// RoleWithNameQueryOne returns a JS expression that finds the first element
// with the given ARIA role and accessible name.
func RoleWithNameQueryOne(role, name string) string {
	return fmt.Sprintf(
		`(function() { var els = document.querySelectorAll('[role=%s]'); for (var i = 0; i < els.length; i++) { var el = els[i]; if (el.getAttribute('aria-label') === %s || el.textContent.trim() === %s) return el; } return null; })()`,
		Quote(role), Quote(name), Quote(name),
	)
}
