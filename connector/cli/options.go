package cli

import (
	"fmt"
)

// paramString extracts a string parameter from a map, returning an error if
// the key is present but not a string.
func paramString(params map[string]any, key string) (string, bool, error) {
	v, ok := params[key]
	if !ok {
		return "", false, nil
	}
	s, ok := v.(string)
	if !ok {
		return "", false, fmt.Errorf("cli: parameter %q must be a string, got %T", key, v)
	}
	return s, true, nil
}

// paramBool extracts a bool parameter from a map.
func paramBool(params map[string]any, key string) (bool, bool, error) {
	v, ok := params[key]
	if !ok {
		return false, false, nil
	}
	b, ok := v.(bool)
	if !ok {
		return false, false, fmt.Errorf("cli: parameter %q must be a bool, got %T", key, v)
	}
	return b, true, nil
}

// paramMap extracts a map[string]any parameter from a map.
func paramMap(params map[string]any, key string) (map[string]any, bool, error) {
	v, ok := params[key]
	if !ok {
		return nil, false, nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, false, fmt.Errorf("cli: parameter %q must be a map, got %T", key, v)
	}
	return m, true, nil
}

// parseShellArgs splits a command string into arguments using shell-style
// quoting rules. It handles single quotes, double quotes, and backslash
// escapes within double quotes.
func parseShellArgs(command string) ([]string, error) {
	var args []string
	var current []byte
	inSingle := false
	inDouble := false

	for i := 0; i < len(command); i++ {
		ch := command[i]
		switch {
		case ch == '\\' && inDouble && i+1 < len(command):
			// Backslash escape inside double quotes.
			i++
			current = append(current, command[i])
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case (ch == ' ' || ch == '\t') && !inSingle && !inDouble:
			if len(current) > 0 {
				args = append(args, string(current))
				current = current[:0]
			}
		default:
			current = append(current, ch)
		}
	}

	if inSingle || inDouble {
		return nil, fmt.Errorf("cli: unterminated quote in command %q", command)
	}

	if len(current) > 0 {
		args = append(args, string(current))
	}

	return args, nil
}
