package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/scrutineer/scrutineer/core/config"
	"github.com/scrutineer/scrutineer/core/connector"
	"github.com/scrutineer/scrutineer/core/coverage"
	"github.com/scrutineer/scrutineer/core/engine"
	"github.com/scrutineer/scrutineer/core/exitcode"
	"github.com/scrutineer/scrutineer/core/reporter"
	"github.com/scrutineer/scrutineer/core/schema"
	"github.com/scrutineer/scrutineer/core/telemetry"
)

func cmdRun(registry *connector.Registry, args []string) int {
	flags, err := config.ParseFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %s\n", err)
		return exitcode.ConfigError
	}

	// Load config
	var cfg *schema.Config
	if flags.ConfigFile != "" {
		cfg, err = config.LoadFromFile(flags.ConfigFile)
	} else {
		cfg, err = config.Load(".")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
		return exitcode.ConfigError
	}
	cfg = config.Merge(cfg, flags)

	// Load test suites
	suites, loadErr := loadSuites(cfg.Tests)
	if loadErr != nil {
		fmt.Fprintf(os.Stderr, "Error loading tests: %s\n", loadErr)
		return exitcode.ConfigError
	}
	if len(suites) == 0 {
		fmt.Fprintln(os.Stderr, "No test files found in manifest")
		return exitcode.ConfigError
	}

	// Set up reporter
	rep := buildReporter(cfg)

	// Set up telemetry writer
	var telWriter telemetry.RecordWriter
	if cfg.Telemetry.Enabled {
		telWriter, err = openTelemetryWriter(cfg.Telemetry.Output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening telemetry log: %s\n", err)
			return exitcode.InternalError
		}
		defer telWriter.Close()
	}

	// Set up coverage tracker
	tracker := coverage.NewTracker()

	// Build engine
	eng := engine.New(
		engine.WithRegistry(registry),
		engine.WithReporter(rep),
		engine.WithTelemetry(telWriter),
		engine.WithCoverage(tracker),
		engine.WithParallelism(cfg.Parallelism),
		engine.WithConnectorConfigs(cfg.Connectors),
	)

	// Run with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	results := eng.Run(ctx, suites)
	code := eng.ExitCode(results)

	// Flush reporter output
	rep.Flush(os.Stdout)

	// Check coverage gate
	if cfg.Coverage.Threshold > 0 {
		gate := &coverage.Gate{Threshold: cfg.Coverage.Threshold}
		if gateErr := gate.Check(tracker); gateErr != nil {
			fmt.Fprintf(os.Stderr, "\n%s\n", gateErr)
			if code == exitcode.OK {
				code = exitcode.TestFailure
			}
		}
	}

	return code
}

func loadSuites(paths []string) ([]schema.TestSuite, error) {
	var suites []schema.TestSuite
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		suite, err := schema.ParseSuite(data)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		suites = append(suites, *suite)
	}
	return suites, nil
}

func buildReporter(cfg *schema.Config) reporter.Reporter {
	for _, rc := range cfg.Reporters {
		if rc.Type == "json" {
			return reporter.NewJSONReporter()
		}
	}
	return reporter.NewANSIReporter()
}

func openTelemetryWriter(path string) (telemetry.RecordWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return telemetry.NewWriter(f), nil
}

// NewANSIReporter and NewJSONReporter may need to be exported from the reporter package.
// For now, we check if they exist; if not, we'll add constructor functions.
var _ io.Writer = os.Stdout // compile-time check
