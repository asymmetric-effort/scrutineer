# CLI Reference

The `scrutineer` binary is the single entry point for all framework operations. It is built from `cmd/scrutineer/main.go`.

## Synopsis

```
scrutineer <command> [options]
```

If no command is provided, the usage help is printed and the process exits with code `0`.

---

## Commands

### scrutineer run

Run tests from the scrutineer.yaml manifest.

```
scrutineer run [options]
```

This is the primary command. It loads configuration, parses test suites, executes tests (with optional parallelism), reports results, and checks coverage gates.

#### Execution Flow

1. Parse CLI flags.
2. Load `scrutineer.yaml` (or the file specified by `--config`).
3. Merge CLI flag overrides into the loaded config.
4. Load and parse all test suite files listed in the `tests` manifest.
5. Initialize the reporter (ANSI or JSON).
6. Open the telemetry writer (if enabled).
7. Initialize the coverage tracker.
8. Build the test engine with the configured registry, reporter, telemetry writer, coverage tracker, and parallelism.
9. Run all suites with signal handling (Ctrl+C / SIGINT triggers graceful shutdown).
10. Flush reporter output to stdout.
11. Check the coverage gate; downgrade exit code to `1` if coverage is below threshold.

#### Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config <file>` | string | `"scrutineer.yaml"` | Path to configuration file. |
| `--parallelism <n>` | int | (from config) | Number of parallel test workers. |
| `--timeout <dur>` | string | (from config) | Default test timeout (Go duration syntax). |
| `--format <type>` | string | (from config) | Output format: `ansi` or `json`. |
| `--verbose` | bool | `false` | Enable verbose output (activates telemetry). |
| `--tags <tags>` | string | (from config) | Comma-separated list of test tags to filter. |

#### Examples

Run with defaults:
```bash
scrutineer run
```

Run with a custom config file and JSON output:
```bash
scrutineer run --config ci/scrutineer.yaml --format json
```

Run 8 tests in parallel with a 60-second timeout:
```bash
scrutineer run --parallelism 8 --timeout 60s
```

Run only tests tagged "smoke":
```bash
scrutineer run --tags smoke
```

Run with verbose telemetry:
```bash
scrutineer run --verbose
```

#### Exit Codes

See the [Exit Codes Reference](exit-codes.md) for the full list. Summary:

| Code | Meaning |
|------|---------|
| 0 | All tests passed and coverage gate met. |
| 1 | One or more test assertions failed, or coverage below threshold. |
| 2 | Connection/network error. |
| 3 | Configuration or YAML parse error (bad config, bad test file, no tests found). |
| 4 | Internal framework error (e.g., cannot open telemetry log). |

---

### scrutineer log-dump

Dump a binary telemetry log file to stdout in human-readable format.

```
scrutineer log-dump <file>
```

Reads the TLV binary log file, decodes each record, and prints one line per event.

#### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<file>` | Yes | Path to the `.log` binary telemetry file. |

#### Output Format

Each line has the format:

```
[<RFC3339Nano timestamp>] <EventType> <key>=<value>... <detail>
```

Example output:
```
[2026-04-26T10:30:00.123456789Z] TestStart suite=API test=CreateUser
[2026-04-26T10:30:00.234567890Z] Request url=https://api.example.com/users method=POST
[2026-04-26T10:30:00.345678901Z] Response status=201
[2026-04-26T10:30:00.345679000Z] Assertion passed=true
[2026-04-26T10:30:00.345680000Z] TestPass suite=API test=CreateUser
```

Tags are printed as `key=value` pairs. If the detail blob is valid JSON, it is printed as-is; otherwise it is printed as a raw string.

#### Examples

```bash
scrutineer log-dump scrutineer.log
```

```bash
scrutineer log-dump results/test-run.log | grep TestFail
```

#### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Log dumped successfully. |
| 3 | Missing file argument or file not found. |
| 4 | Error reading/decoding a record. |

---

### scrutineer browsers

Manage browser installations for browser-automation tests.

```
scrutineer browsers <subcommand>
```

If no subcommand is provided, usage help is printed and the process exits with code `3`.

#### Subcommands

##### scrutineer browsers install

Download browser binaries (Chromium, Firefox, WebKit) from the Playwright CDN.

```
scrutineer browsers install
```

Downloads Playwright's open-source patched browser builds for use with scrutineer's CDP-based browser automation. Currently a placeholder (Phase 7).

##### scrutineer browsers list

List installed browser binaries.

```
scrutineer browsers list
```

Shows which browsers are available locally. Currently a placeholder (Phase 7).

##### scrutineer browsers help

Print the browsers subcommand usage.

```
scrutineer browsers help
```

#### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Command completed successfully. |
| 3 | Unknown subcommand or missing subcommand. |

---

### scrutineer version

Print version and build information.

```
scrutineer version
```

Output:
```
scrutineer 0.0.1-dev
```

The version string is set at compile time via the `version` variable in `main.go`. The `printVersion()` function also prints Go version and OS/architecture:

```
scrutineer 0.0.1-dev
  go:       go1.26
  os/arch:  linux/amd64
```

Note: the `version` command in `main()` uses `fmt.Printf` directly (single-line output), while `printVersion()` provides the detailed multi-line format.

#### Exit Code

Always exits with code `0`.

---

### scrutineer help

Print usage information.

```
scrutineer help
scrutineer --help
scrutineer -h
```

All three forms are equivalent. Prints the full command listing with available options.

#### Exit Code

Always exits with code `0`.

---

## Unknown Commands

If an unrecognized command is provided, scrutineer prints an error to stderr, shows the usage help, and exits with code `3` (ConfigError).

```bash
$ scrutineer foo
unknown command: foo

scrutineer -- extensible test framework
...
```

---

## Signal Handling

The `run` command sets up signal handling via `signal.NotifyContext` for `os.Interrupt` (Ctrl+C / SIGINT). When interrupted, the context is cancelled, allowing in-flight tests to complete their current step before the engine shuts down gracefully.

---

## Registered Connectors

The CLI binary registers the following connectors at startup:

| Name | Package | Description |
|------|---------|-------------|
| `cli` | `connector/cli` | CLI process execution and assertion |
| `http` | `connector/http` | HTTP/REST/GraphQL requests |
| `ssh` | `connector/ssh` | SSH remote command execution |
| `grpc` | `connector/grpc` | gRPC unary and streaming RPCs |
| `browser` | `connector/browser` | Browser automation via CDP |
