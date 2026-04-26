// Package main is the entry point for the scrutineer CLI.
package main

import (
	"fmt"
	"os"

	"github.com/scrutineer/scrutineer/core/connector"
	"github.com/scrutineer/scrutineer/core/exitcode"

	connBrowser "github.com/scrutineer/scrutineer/connector/browser"
	connCLI "github.com/scrutineer/scrutineer/connector/cli"
	connGRPC "github.com/scrutineer/scrutineer/connector/grpc"
	connHTTP "github.com/scrutineer/scrutineer/connector/http"
	connSSH "github.com/scrutineer/scrutineer/connector/ssh"
)

var version = "0.0.1-dev"

func main() {
	registry := connector.NewRegistry()
	registerConnectors(registry)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(exitcode.OK)
	}

	var code int
	switch os.Args[1] {
	case "run":
		code = cmdRun(registry, os.Args[2:])
	case "log-dump":
		code = cmdLogDump(os.Args[2:])
	case "browsers":
		code = cmdBrowsers(os.Args[2:])
	case "version":
		fmt.Printf("scrutineer %s\n", version)
		code = exitcode.OK
	case "help", "--help", "-h":
		printUsage()
		code = exitcode.OK
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		code = exitcode.ConfigError
	}

	os.Exit(code)
}

func registerConnectors(registry *connector.Registry) {
	registry.Register("cli", func() connector.Connector { return connCLI.New() })
	registry.Register("http", func() connector.Connector { return connHTTP.New() })
	registry.Register("ssh", func() connector.Connector { return connSSH.New() })
	registry.Register("grpc", func() connector.Connector { return connGRPC.New() })
	registry.Register("browser", func() connector.Connector { return connBrowser.New() })
}

func printUsage() {
	fmt.Println(`scrutineer — extensible test framework

Usage:
  scrutineer <command> [options]

Commands:
  run              Run tests from scrutineer.yaml manifest
  log-dump <file>  Dump binary telemetry log to stdout
  browsers         Manage browser installations
  version          Print version information
  help             Show this help

Run Options:
  --config <file>      Config file (default: scrutineer.yaml)
  --parallelism <n>    Number of parallel tests
  --timeout <dur>      Default test timeout
  --format <type>      Output format: ansi, json
  --verbose            Verbose output
  --tags <tags>        Filter tests by tags (comma-separated)`)
}
