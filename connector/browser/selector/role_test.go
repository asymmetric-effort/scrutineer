package selector

import (
	"strings"
	"testing"
)

func TestRoleQueryOne(t *testing.T) {
	tests := []struct {
		name string
		role string
	}{
		{"button", "button"},
		{"navigation", "navigation"},
		{"dialog", "dialog"},
		{"alert", "alert"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoleQueryOne(tt.role)
			if !strings.Contains(got, "querySelector") {
				t.Errorf("should contain querySelector: %s", got)
			}
			if !strings.Contains(got, "[role=") {
				t.Errorf("should contain role attribute selector: %s", got)
			}
		})
	}
}

func TestRoleQueryAll(t *testing.T) {
	got := RoleQueryAll("listitem")
	if !strings.Contains(got, "querySelectorAll") {
		t.Errorf("should contain querySelectorAll: %s", got)
	}
	if !strings.Contains(got, "Array.from") {
		t.Errorf("should wrap in Array.from: %s", got)
	}
}

func TestRoleWithNameQueryOne(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		accName string
	}{
		{"button submit", "button", "Submit"},
		{"link home", "link", "Home"},
		{"tab settings", "tab", "Settings"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoleWithNameQueryOne(tt.role, tt.accName)
			if !strings.Contains(got, "querySelectorAll") {
				t.Errorf("should contain querySelectorAll: %s", got)
			}
			if !strings.Contains(got, "aria-label") {
				t.Errorf("should check aria-label: %s", got)
			}
			if !strings.Contains(got, "textContent.trim()") {
				t.Errorf("should check textContent: %s", got)
			}
		})
	}
}

func TestRoleWithNameQueryOne_EscapesName(t *testing.T) {
	got := RoleWithNameQueryOne("button", `Click "here"`)
	if !strings.Contains(got, `\"here\"`) {
		t.Errorf("should escape quotes in name: %s", got)
	}
}
