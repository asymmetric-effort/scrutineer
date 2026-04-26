package reporter

// ANSI escape code constants for terminal coloring.
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	Gray   = "\033[90m"
)

// ColorEnabled controls whether Colorize applies ANSI escape codes.
// Set to false for non-TTY output.
var ColorEnabled = true

// Colorize wraps text with the given ANSI color code and a reset suffix.
// If ColorEnabled is false, the text is returned unmodified.
func Colorize(text, color string) string {
	if !ColorEnabled {
		return text
	}
	return color + text + Reset
}
