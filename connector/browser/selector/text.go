package selector

import "fmt"

// TextQueryOne returns a JS expression that finds the first element containing
// the exact visible text.
func TextQueryOne(text string) string {
	return fmt.Sprintf(
		`(function() { var tw = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT, null, false); while (tw.nextNode()) { if (tw.currentNode.textContent.trim() === %s) return tw.currentNode.parentElement; } return null; })()`,
		Quote(text),
	)
}

// TextQueryAll returns a JS expression that finds all elements containing
// the exact visible text.
func TextQueryAll(text string) string {
	return fmt.Sprintf(
		`(function() { var tw = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT, null, false); var a = []; while (tw.nextNode()) { if (tw.currentNode.textContent.trim() === %s) a.push(tw.currentNode.parentElement); } return a; })()`,
		Quote(text),
	)
}

// TextContainsQueryOne returns a JS expression that finds the first element
// whose visible text contains the given substring.
func TextContainsQueryOne(text string) string {
	return fmt.Sprintf(
		`(function() { var tw = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT, null, false); while (tw.nextNode()) { if (tw.currentNode.textContent.includes(%s)) return tw.currentNode.parentElement; } return null; })()`,
		Quote(text),
	)
}
