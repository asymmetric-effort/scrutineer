// Package exitcode defines the process exit codes used by scrutineer.
package exitcode

const (
	// OK indicates all tests passed.
	OK = 0

	// TestFailure indicates one or more test assertions failed.
	TestFailure = 1

	// ConnectionError indicates a connection or network error
	// prevented tests from reaching the target.
	ConnectionError = 2

	// ConfigError indicates a configuration or YAML parse error.
	ConfigError = 3

	// InternalError indicates an unexpected framework or internal error.
	InternalError = 4
)

// String returns a human-readable description for an exit code.
func String(code int) string {
	switch code {
	case OK:
		return "all tests passed"
	case TestFailure:
		return "one or more test assertions failed"
	case ConnectionError:
		return "connection or network error"
	case ConfigError:
		return "configuration or YAML parse error"
	case InternalError:
		return "framework or internal error"
	default:
		return "unknown exit code"
	}
}
