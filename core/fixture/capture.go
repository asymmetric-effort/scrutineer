package fixture

import (
	"fmt"
	"strings"
)

// Extract extracts a value from a result data map using a dot-notation path.
// e.g. "body.user.id" extracts data["body"]["user"]["id"]
func Extract(data map[string]any, path string) (any, error) {
	if data == nil {
		return nil, fmt.Errorf("cannot extract from nil map")
	}

	parts := strings.Split(path, ".")
	var current any = data

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot navigate through non-map value at %q", part)
		}
		current, ok = m[part]
		if !ok {
			return nil, fmt.Errorf("key %q not found", part)
		}
	}

	return current, nil
}
