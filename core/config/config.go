package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/scrutineer/scrutineer/core/schema"
	"github.com/scrutineer/scrutineer/core/yaml"
)

const configFileName = "scrutineer.yaml"

// Load reads scrutineer.yaml from the given directory and merges with defaults.
// Returns an error if the file doesn't exist or is invalid.
func Load(dir string) (*schema.Config, error) {
	path := filepath.Join(dir, configFileName)
	return LoadFromFile(path)
}

// LoadFromFile reads a specific config file path and merges with defaults.
func LoadFromFile(path string) (*schema.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg := Defaults()

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return cfg, nil
}

// Merge merges CLI flag overrides into a loaded config.
// Non-zero flag values override config file values.
func Merge(cfg *schema.Config, flags *Flags) *schema.Config {
	if flags.Parallelism != 0 {
		cfg.Parallelism = flags.Parallelism
	}

	if flags.Timeout != "" {
		cfg.Timeout = flags.Timeout
	}

	if flags.Format != "" {
		cfg.Reporters = []schema.ReporterConfig{
			{Type: flags.Format},
		}
	}

	if flags.Verbose {
		cfg.Telemetry.Enabled = true
	}

	if len(flags.Tags) > 0 {
		cfg.Tests = flags.Tags
	}

	return cfg
}
