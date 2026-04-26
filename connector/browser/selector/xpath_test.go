package selector

import (
	"strings"
	"testing"
)

func TestXPathQueryOne(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"simple", "//div"},
		{"with predicate", "//input[@type='text']"},
		{"text content", "//span[text()='Hello']"},
		{"descendant", "//div//span"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := XPathQueryOne(tt.expr)
			if !strings.Contains(got, "document.evaluate") {
				t.Errorf("should contain document.evaluate: %s", got)
			}
			if !strings.Contains(got, "FIRST_ORDERED_NODE_TYPE") {
				t.Errorf("should contain FIRST_ORDERED_NODE_TYPE: %s", got)
			}
			if !strings.Contains(got, "singleNodeValue") {
				t.Errorf("should contain singleNodeValue: %s", got)
			}
		})
	}
}

func TestXPathQueryAll(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"simple", "//li"},
		{"with attribute", "//a[@href]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := XPathQueryAll(tt.expr)
			if !strings.Contains(got, "document.evaluate") {
				t.Errorf("should contain document.evaluate: %s", got)
			}
			if !strings.Contains(got, "ORDERED_NODE_SNAPSHOT_TYPE") {
				t.Errorf("should contain ORDERED_NODE_SNAPSHOT_TYPE: %s", got)
			}
			if !strings.Contains(got, "snapshotLength") {
				t.Errorf("should contain snapshotLength: %s", got)
			}
		})
	}
}

func TestXPathQueryOne_QuotesExpression(t *testing.T) {
	expr := `//div[@class="test"]`
	got := XPathQueryOne(expr)
	// The expression should be properly quoted.
	if !strings.Contains(got, `\"test\"`) {
		t.Errorf("should escape quotes in expression: %s", got)
	}
}
