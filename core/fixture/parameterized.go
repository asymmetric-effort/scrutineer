package fixture

// ParameterSet defines a set of parameters for parameterized tests.
type ParameterSet struct {
	Name   string
	Values map[string]any
}

// ExpandedTest represents a test that has been expanded from a template
// with a specific set of parameters applied.
type ExpandedTest struct {
	Name   string
	Steps  []map[string]any
	Params map[string]any
}

// Expand takes a test template and a list of parameter sets,
// returns expanded tests with parameters interpolated.
// Each expanded test gets a name like "OriginalName [paramSet.Name]".
// If params is empty, no expanded tests are returned.
func Expand(name string, steps []map[string]any, params []ParameterSet) []ExpandedTest {
	if len(params) == 0 {
		return nil
	}

	result := make([]ExpandedTest, 0, len(params))

	for _, ps := range params {
		// Deep copy the steps so each expanded test has independent data.
		copiedSteps := deepCopySteps(steps)

		result = append(result, ExpandedTest{
			Name:   name + " [" + ps.Name + "]",
			Steps:  copiedSteps,
			Params: ps.Values,
		})
	}

	return result
}

// deepCopySteps creates a deep copy of a slice of step maps.
func deepCopySteps(steps []map[string]any) []map[string]any {
	if steps == nil {
		return nil
	}

	result := make([]map[string]any, len(steps))
	for i, step := range steps {
		result[i] = deepCopyMap(step)
	}
	return result
}

// deepCopyMap creates a deep copy of a map[string]any.
func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}

	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = deepCopyValue(v)
	}
	return result
}

// deepCopyValue creates a deep copy of a value.
func deepCopyValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return deepCopyMap(val)
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = deepCopyValue(item)
		}
		return result
	default:
		// Primitive types are safe to copy by value.
		return v
	}
}
