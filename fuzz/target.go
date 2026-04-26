package fuzz

import "fmt"

// Validate checks that a Target is well-formed.
// Name, connector, and action are required. At least one fuzz field must be
// specified, and every fuzz field must exist in the parameters map.
func (t *Target) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("name is required")
	}
	if t.Connector == "" {
		return fmt.Errorf("connector is required")
	}
	if t.Action == "" {
		return fmt.Errorf("action is required")
	}
	if len(t.FuzzFields) == 0 {
		return fmt.Errorf("at least one fuzz field is required")
	}
	for _, field := range t.FuzzFields {
		if _, ok := t.Parameters[field]; !ok {
			return fmt.Errorf("fuzz field %q not found in parameters", field)
		}
	}
	return nil
}
