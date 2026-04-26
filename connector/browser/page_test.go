package browser

import (
	"encoding/json"
	"testing"
)

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  float64
		ok    bool
	}{
		{"float64", float64(3.14), 3.14, true},
		{"int", int(42), 42.0, true},
		{"int64", int64(100), 100.0, true},
		{"json number", json.Number("12.5"), 12.5, true},
		{"string", "not a number", 0, false},
		{"nil", nil, 0, false},
		{"bool", true, 0, false},
		{"invalid json number", json.Number("not_a_number"), 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toFloat64(tt.input)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("value = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestResolveSelector(t *testing.T) {
	tests := []struct {
		selType string
		value   string
		contain string
	}{
		{"css", "#id", "querySelector"},
		{"xpath", "//div", "document.evaluate"},
		{"text", "Click", "createTreeWalker"},
		{"role", "button", "querySelector"},
		{"", ".class", "querySelector"},    // default to CSS
		{"unknown", ".x", "querySelector"}, // unknown defaults to CSS
	}

	for _, tt := range tests {
		t.Run(tt.selType, func(t *testing.T) {
			result := resolveSelector(tt.selType, tt.value)
			if !searchString(result, tt.contain) {
				t.Errorf("resolveSelector(%q, %q) = %q, should contain %q",
					tt.selType, tt.value, result, tt.contain)
			}
		})
	}
}
