package selector

import "fmt"

// XPathQueryOne returns a JS expression that finds the first element matching an XPath.
func XPathQueryOne(expr string) string {
	return fmt.Sprintf(
		`document.evaluate(%s, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue`,
		Quote(expr),
	)
}

// XPathQueryAll returns a JS expression that finds all elements matching an XPath.
func XPathQueryAll(expr string) string {
	return fmt.Sprintf(
		`(function() { var r = document.evaluate(%s, document, null, XPathResult.ORDERED_NODE_SNAPSHOT_TYPE, null); var a = []; for (var i = 0; i < r.snapshotLength; i++) a.push(r.snapshotItem(i)); return a; })()`,
		Quote(expr),
	)
}
