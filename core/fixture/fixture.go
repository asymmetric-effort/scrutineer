// Package fixture provides fixture management, variable interpolation,
// parameterized test expansion, and value extraction for the scrutineer
// test framework.
package fixture

import (
	"fmt"
	"os"
	"strings"

	"github.com/scrutineer/scrutineer/core/expression"
)

// Store holds fixtures and captured variables for variable interpolation.
type Store struct {
	fixtures     map[string]any       // from YAML fixtures section
	captures     map[string]any       // captured during test execution
	env          map[string]string    // environment variables
	exprRegistry *expression.Registry // optional expression function registry
}

// NewStore creates a new Store with the given fixtures.
// If fixtures is nil, an empty map is used.
func NewStore(fixtures map[string]any) *Store {
	if fixtures == nil {
		fixtures = make(map[string]any)
	}
	return &Store{
		fixtures: fixtures,
		captures: make(map[string]any),
		env:      make(map[string]string),
	}
}

// SetCapture stores a captured value from a test step result.
func (s *Store) SetCapture(key string, value any) {
	s.captures[key] = value
}

// GetCapture retrieves a captured value.
func (s *Store) GetCapture(key string) (any, bool) {
	v, ok := s.captures[key]
	return v, ok
}

// Resolve resolves a variable reference like "fixture.user.name" or
// "capture.user_id" or "env.API_KEY".
// Returns the resolved value and true, or nil and false if not found.
func (s *Store) Resolve(ref string) (any, bool) {
	parts := strings.SplitN(ref, ".", 2)
	if len(parts) < 2 {
		return nil, false
	}

	prefix := parts[0]
	rest := parts[1]

	switch prefix {
	case "fixture":
		return navigatePath(s.fixtures, rest)
	case "capture":
		return navigatePath(s.captures, rest)
	case "env":
		// For env, the rest is the env var name (no further nesting).
		val := os.Getenv(rest)
		if val == "" {
			// Check if the variable is actually set (could be empty string).
			_, found := os.LookupEnv(rest)
			if !found {
				return nil, false
			}
		}
		return val, true
	default:
		return nil, false
	}
}

// navigatePath traverses a map using a dot-notation path.
func navigatePath(data map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var current any = data

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}

	return current, true
}

// SetExpressionRegistry sets the expression function registry for fn: evaluation.
func (s *Store) SetExpressionRegistry(r *expression.Registry) {
	s.exprRegistry = r
}

// Interpolate replaces all ${...} expressions in a string with resolved values.
// Returns the interpolated string and any resolution errors.
// Escaped sequences like \${...} are treated as literals.
// Expressions with the fn: prefix are evaluated as function calls.
func (s *Store) Interpolate(input string) (string, error) {
	var result strings.Builder
	i := 0

	for i < len(input) {
		// Check for escaped dollar-brace.
		if i+1 < len(input) && input[i] == '\\' && input[i+1] == '$' {
			// Write literal $ and skip the backslash.
			result.WriteByte('$')
			i += 2
			continue
		}

		// Check for ${...} pattern.
		if i+1 < len(input) && input[i] == '$' && input[i+1] == '{' {
			// Find the closing brace.
			end := strings.Index(input[i:], "}")
			if end == -1 {
				// No closing brace; write literally.
				result.WriteByte(input[i])
				i++
				continue
			}

			ref := input[i+2 : i+end]

			// Check for fn: prefix — expression function call.
			if strings.HasPrefix(ref, "fn:") {
				exprStr := ref[3:] // strip "fn:" prefix
				val, err := s.evalExpression(exprStr)
				if err != nil {
					return "", fmt.Errorf("expression error in ${fn:%s}: %w", exprStr, err)
				}
				result.WriteString(fmt.Sprintf("%v", val))
				i += end + 1
				continue
			}

			val, ok := s.Resolve(ref)
			if !ok {
				return "", fmt.Errorf("unresolved variable: %s", ref)
			}

			result.WriteString(fmt.Sprintf("%v", val))
			i += end + 1
			continue
		}

		result.WriteByte(input[i])
		i++
	}

	return result.String(), nil
}

// evalExpression parses and evaluates an expression function string.
func (s *Store) evalExpression(input string) (any, error) {
	if s.exprRegistry == nil {
		return nil, fmt.Errorf("expression functions not available (no registry configured)")
	}
	return expression.EvalString(input, s.exprRegistry, s)
}

// InterpolateMap recursively interpolates all string values in a map.
// Non-string values pass through unchanged.
func (s *Store) InterpolateMap(input map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(input))

	for k, v := range input {
		resolved, err := s.interpolateValue(v)
		if err != nil {
			return nil, err
		}
		result[k] = resolved
	}

	return result, nil
}

// interpolateValue recursively interpolates a value.
func (s *Store) interpolateValue(v any) (any, error) {
	switch val := v.(type) {
	case string:
		return s.Interpolate(val)
	case map[string]any:
		return s.InterpolateMap(val)
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			resolved, err := s.interpolateValue(item)
			if err != nil {
				return nil, err
			}
			result[i] = resolved
		}
		return result, nil
	default:
		return v, nil
	}
}
