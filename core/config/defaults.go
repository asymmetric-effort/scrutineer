package config

import "github.com/scrutineer/scrutineer/core/schema"

// Defaults returns a Config populated with sensible default values.
func Defaults() *schema.Config {
	return &schema.Config{
		Parallelism: 1,
		Timeout:     "30s",
		Reporters: []schema.ReporterConfig{
			{Type: "ansi"},
		},
		Coverage: schema.CoverageConfig{
			Threshold: 98.0,
		},
		Browsers: schema.BrowsersConfig{
			Chromium: false,
			Firefox:  false,
			WebKit:   false,
		},
		Telemetry: schema.TelemetryConfig{
			Enabled: true,
			Output:  "scrutineer.log",
		},
	}
}
