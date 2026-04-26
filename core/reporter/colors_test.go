package reporter

import "testing"

func TestColorize_Enabled(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = true }()

	got := Colorize("hello", Red)
	want := Red + "hello" + Reset
	if got != want {
		t.Errorf("Colorize with color enabled: got %q, want %q", got, want)
	}
}

func TestColorize_Disabled(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	got := Colorize("hello", Red)
	if got != "hello" {
		t.Errorf("Colorize with color disabled: got %q, want %q", got, "hello")
	}
}

func TestColorize_AllColors(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = true }()

	colors := []struct {
		name  string
		value string
	}{
		{"Reset", Reset},
		{"Bold", Bold},
		{"Red", Red},
		{"Green", Green},
		{"Yellow", Yellow},
		{"Blue", Blue},
		{"Cyan", Cyan},
		{"Gray", Gray},
	}

	for _, c := range colors {
		t.Run(c.name, func(t *testing.T) {
			got := Colorize("text", c.value)
			want := c.value + "text" + Reset
			if got != want {
				t.Errorf("Colorize(%q, %s): got %q, want %q", "text", c.name, got, want)
			}
		})
	}
}

func TestColorize_EmptyText(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = true }()

	got := Colorize("", Green)
	want := Green + "" + Reset
	if got != want {
		t.Errorf("Colorize empty text: got %q, want %q", got, want)
	}
}

func TestColorize_DisabledEmptyText(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	got := Colorize("", Green)
	if got != "" {
		t.Errorf("Colorize disabled empty text: got %q, want %q", got, "")
	}
}

func TestColorConstants(t *testing.T) {
	// Verify the escape sequences are correct.
	if Reset != "\033[0m" {
		t.Errorf("Reset: got %q", Reset)
	}
	if Bold != "\033[1m" {
		t.Errorf("Bold: got %q", Bold)
	}
	if Red != "\033[31m" {
		t.Errorf("Red: got %q", Red)
	}
	if Green != "\033[32m" {
		t.Errorf("Green: got %q", Green)
	}
	if Yellow != "\033[33m" {
		t.Errorf("Yellow: got %q", Yellow)
	}
	if Blue != "\033[34m" {
		t.Errorf("Blue: got %q", Blue)
	}
	if Cyan != "\033[36m" {
		t.Errorf("Cyan: got %q", Cyan)
	}
	if Gray != "\033[90m" {
		t.Errorf("Gray: got %q", Gray)
	}
}
