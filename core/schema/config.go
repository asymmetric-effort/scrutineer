package schema

// Config represents the scrutineer.yaml project configuration.
type Config struct {
	Version     string                    `yaml:"version"`
	Tests       []string                  `yaml:"tests"`
	Parallelism int                       `yaml:"parallelism"`
	Timeout     string                    `yaml:"timeout"`
	Reporters   []ReporterConfig          `yaml:"reporters"`
	Coverage    CoverageConfig            `yaml:"coverage"`
	Browsers    BrowsersConfig            `yaml:"browsers"`
	Connectors  map[string]map[string]any `yaml:"connectors"`
	Telemetry   TelemetryConfig           `yaml:"telemetry"`
}

// ReporterConfig configures a single reporter output.
type ReporterConfig struct {
	Type   string `yaml:"type"`
	Output string `yaml:"output"`
}

// CoverageConfig configures code coverage thresholds.
type CoverageConfig struct {
	Threshold float64 `yaml:"threshold"`
}

// BrowsersConfig configures which browsers are enabled for browser testing.
type BrowsersConfig struct {
	Chromium bool `yaml:"chromium"`
	Firefox  bool `yaml:"firefox"`
	WebKit   bool `yaml:"webkit"`
}

// TelemetryConfig configures telemetry output.
type TelemetryConfig struct {
	Enabled bool   `yaml:"enabled"`
	Output  string `yaml:"output"`
}

// validReporterTypes lists accepted reporter type values.
var validReporterTypes = map[string]bool{
	"ansi": true,
	"json": true,
}
