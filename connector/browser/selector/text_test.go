package selector

import (
	"strings"
	"testing"
)

func TestTextQueryOne(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"simple", "Click me"},
		{"with quotes", `Say "hello"`},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TextQueryOne(tt.text)
			if !strings.Contains(got, "createTreeWalker") {
				t.Errorf("should contain createTreeWalker: %s", got)
			}
			if !strings.Contains(got, "SHOW_TEXT") {
				t.Errorf("should contain SHOW_TEXT: %s", got)
			}
			if !strings.Contains(got, "textContent.trim()") {
				t.Errorf("should contain textContent.trim(): %s", got)
			}
			if !strings.Contains(got, "parentElement") {
				t.Errorf("should contain parentElement: %s", got)
			}
		})
	}
}

func TestTextQueryAll(t *testing.T) {
	got := TextQueryAll("Submit")
	if !strings.Contains(got, "createTreeWalker") {
		t.Errorf("should contain createTreeWalker: %s", got)
	}
	if !strings.Contains(got, "var a = []") {
		t.Errorf("should collect into array: %s", got)
	}
	if !strings.Contains(got, "a.push") {
		t.Errorf("should push to array: %s", got)
	}
}

func TestTextContainsQueryOne(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"simple", "partial"},
		{"with special chars", "hello & world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TextContainsQueryOne(tt.text)
			if !strings.Contains(got, "includes(") {
				t.Errorf("should contain includes(): %s", got)
			}
			if !strings.Contains(got, "createTreeWalker") {
				t.Errorf("should contain createTreeWalker: %s", got)
			}
		})
	}
}

func TestTextQueryOne_EscapesQuotes(t *testing.T) {
	got := TextQueryOne(`It's a "test"`)
	if !strings.Contains(got, `\"test\"`) {
		t.Errorf("should escape double quotes: %s", got)
	}
}
