package selector

import (
	"strings"
	"testing"
)

func TestCSSQueryOne(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		want     string
	}{
		{"simple tag", "div", `document.querySelector("div")`},
		{"class", ".btn", `document.querySelector(".btn")`},
		{"id", "#main", `document.querySelector("#main")`},
		{"attribute", `input[type="text"]`, `document.querySelector("input[type=\"text\"]")`},
		{"descendant", "div > span", `document.querySelector("div > span")`},
		{"pseudo", "li:first-child", `document.querySelector("li:first-child")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CSSQueryOne(tt.selector)
			if got != tt.want {
				t.Errorf("CSSQueryOne(%q) = %q, want %q", tt.selector, got, tt.want)
			}
		})
	}
}

func TestCSSQueryAll(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		want     string
	}{
		{"simple", "div", `Array.from(document.querySelectorAll("div"))`},
		{"class", ".item", `Array.from(document.querySelectorAll(".item"))`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CSSQueryAll(tt.selector)
			if got != tt.want {
				t.Errorf("CSSQueryAll(%q) = %q, want %q", tt.selector, got, tt.want)
			}
		})
	}
}

func TestQuote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "hello", `"hello"`},
		{"with double quotes", `say "hi"`, `"say \"hi\""`},
		{"with backslash", `path\to`, `"path\\to"`},
		{"with newline", "line1\nline2", `"line1\nline2"`},
		{"with carriage return", "line1\rline2", `"line1\rline2"`},
		{"with tab", "col1\tcol2", `"col1\tcol2"`},
		{"empty", "", `""`},
		{"special chars", `<div class="x">`, `"<div class=\"x\">"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Quote(tt.input)
			if got != tt.want {
				t.Errorf("Quote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCSSQueryOne_SpecialCharacters(t *testing.T) {
	// Selector with special characters that need escaping.
	sel := `div[data-value="hello\nworld"]`
	result := CSSQueryOne(sel)
	if !strings.Contains(result, "querySelector") {
		t.Errorf("should contain querySelector: %s", result)
	}
}
