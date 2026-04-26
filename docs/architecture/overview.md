# Architecture Overview

Scrutineer is an extensible test framework built in Go with zero third-party dependencies in the core module. It uses a modular architecture where protocol-specific connectors are decoupled from the test engine, assertion library, and reporting system.

## Module Structure

The project is organized as a Go workspace with nine independent modules:

```
scrutineer/
  go.work                       # workspace definition

  core/                         # Module: github.com/scrutineer/scrutineer/core
    connector/                  # Connector interface, Registry, Result types
    engine/                     # Engine, Runner, Pool, TestContext
    schema/                     # Config, TestSuite, Test, TestStep, validation
    yaml/                       # YAML parser (implemented from scratch)
    assertion/                  # Assertion interface + 20+ implementations
    reporter/                   # Reporter interface, ANSI + JSON implementations
    coverage/                   # Coverage Tracker, Gate, model types
    fixture/                    # Fixture Store, captures, variable interpolation
    retry/                      # Retry, timeout, polling utilities
    telemetry/                  # TLV binary log format, reader, writer
    config/                     # Config file loading, defaults, CLI flag merging
    exitcode/                   # Exit code constants

  connector/
    cli/                        # Module: CLI connector (exec, filesystem)
    http/                       # Module: HTTP/REST connector (request, auth, TLS)
    ssh/                        # Module: SSH connector (exec, tunnel)
    grpc/                       # Module: gRPC connector (unary, streaming)
    browser/                    # Module: Browser automation (CDP)

  loadtest/                     # Module: Distributed load testing
  fuzz/                         # Module: Fuzz testing

  cmd/scrutineer/               # Module: CLI entry point
```

## Dependency Graph

Dependencies flow inward toward the core module. Connectors depend on core, but core never depends on connectors. The CLI binary ties everything together.

```
cmd/scrutineer
  |-- core (engine, schema, config, reporter, assertion, ...)
  |-- connector/cli
  |-- connector/http
  |-- connector/ssh    (depends on golang.org/x/crypto/ssh)
  |-- connector/grpc   (depends on google.golang.org/grpc, google.golang.org/protobuf)
  |-- connector/browser
  |-- loadtest         (depends on connector/ssh for distributed execution)
  |-- fuzz             (depends on core/connector for execution)
```

**External dependency rules:**
- The `core` module has zero external dependencies -- only Go standard library
- `connector/ssh` depends on `golang.org/x/crypto/ssh` (maintained by the Go team)
- `connector/grpc` depends on `google.golang.org/grpc` and `google.golang.org/protobuf` (maintained by Google)
- All other modules use only the standard library
- No other third-party dependencies are permitted

## Core vs Connectors

### Core Module

The core module provides the framework's backbone: parsing YAML, validating schemas, running tests, evaluating assertions, tracking coverage, and reporting results. It defines the `Connector` interface but does not implement any protocol-specific logic.

Key abstractions in core:

| Package      | Responsibility                                              |
|-------------|-------------------------------------------------------------|
| `connector` | Defines `Connector` interface, `Registry`, `Step`, `Result` |
| `engine`    | Orchestrates suite/test execution with parallel support      |
| `schema`    | Go types for Config, TestSuite, Test, TestStep               |
| `assertion` | `Assertion` interface + `Builder` pattern for all operators  |
| `reporter`  | `Reporter` interface + ANSI and JSON implementations         |
| `coverage`  | Thread-safe `Tracker` + `Gate` threshold enforcement         |
| `fixture`   | `Store` for fixtures, captures, and variable interpolation   |
| `yaml`      | Custom YAML parser (from scratch, no dependencies)           |
| `config`    | File loading, defaults, CLI flag merging                     |
| `telemetry` | TLV binary log format for nanosecond-precision events        |

### Connector Modules

Each connector implements the `Connector` interface from `core/connector`:

```go
type Connector interface {
    Name() string
    Setup(ctx context.Context, config map[string]any) error
    Execute(ctx context.Context, step Step) (*Result, error)
    Teardown(ctx context.Context) error
}
```

Connectors are independent modules that can be compiled separately. They register themselves with the `Registry` at startup.

| Connector | Actions                                      | External Deps         |
|-----------|----------------------------------------------|-----------------------|
| `cli`     | `exec`, `filesystem`                         | None (stdlib)         |
| `http`    | `request`                                    | None (stdlib)         |
| `ssh`     | `exec`, `tunnel`                             | `x/crypto/ssh`        |
| `grpc`    | `unary`, `server_stream`, `client_stream`, `bidi_stream` | `grpc`, `protobuf` |
| `browser` | `navigate`, `click`, `type`, `fill`, `select`, `screenshot`, `evaluate`, `wait_for_selector`, `get_text`, `get_attribute` | None (stdlib) |

## How the Engine Orchestrates Test Execution

The engine is the central coordinator. Here is the execution flow:

### 1. Configuration Loading

```
scrutineer.yaml --> config.Load() --> schema.Config
```

The config file is parsed using the custom YAML parser. CLI flags are merged over config file values via `config.Merge()`. Defaults are applied for any unset values.

### 2. Suite Loading

```
test file paths --> os.ReadFile() --> schema.ParseSuite() --> []schema.TestSuite
```

Each test file in the manifest is read, parsed from YAML into raw maps, then converted to typed `TestSuite` structs. The `ParseTestStep` function separates known fields (connector, action, assert, capture, timeout) from connector-specific parameters, which go into the `Parameters` map.

### 3. Engine Construction

```go
eng := engine.New(
    engine.WithRegistry(registry),     // connector factories
    engine.WithReporter(rep),          // ANSI or JSON reporter
    engine.WithTelemetry(telWriter),   // binary log writer
    engine.WithCoverage(tracker),      // coverage tracker
    engine.WithParallelism(cfg.Parallelism),
)
```

The engine uses the functional options pattern. All dependencies are injected at construction time.

### 4. Suite Execution

```
Engine.Run(ctx, suites)
  for each suite (in parallel via Pool):
    Engine.runSuite(ctx, suite)
      reporter.OnSuiteStart()
      coverage.RegisterSuite()
      for each test:
        if test.Skip: skip
        TestContext = NewTestContext(suite, test, fixtures)
        reporter.OnTestStart()
        Runner.RunTest(ctx, tctx, test, config)
      reporter.OnSuiteEnd()
```

The `Pool` provides bounded concurrency. With `parallelism: 4`, up to 4 suites execute simultaneously using a semaphore pattern (buffered channel).

### 5. Test Execution (Runner)

For each test, the Runner executes steps sequentially:

```
Runner.RunTest(ctx, tctx, test, config)
  for each step:
    Runner.runStep(ctx, tctx, step, index, config)
      1. Resolve connector name (step-level or test-level)
      2. Get connector from Registry
      3. Connector.Setup(ctx, config)
      4. Interpolate variables in step parameters
      5. Connector.Execute(ctx, step)
      6. Process captures (extract values from result)
      7. Evaluate assertions (build + evaluate)
      8. Connector.Teardown(ctx)       [always, via defer]
```

If any step fails, the test stops and subsequent steps are not executed.

## The Connector Lifecycle

Every step follows the same lifecycle:

### Setup

The connector initializes itself from the configuration map. For HTTP, this means building an `http.Client` with TLS settings. For SSH, this means dialing the remote host. For browser, this means launching a browser process and connecting via WebSocket.

```go
func (c *HTTPConnector) Setup(ctx context.Context, config map[string]any) error {
    // Parse base_url, headers, timeout, TLS settings
    // Build http.Client
}
```

### Execute

The connector performs the requested action and returns a `Result` containing:
- `Data` (map[string]any): keyed output values for assertions and captures
- `Elapsed` (time.Duration): step timing
- `Meta` (map[string]string): metadata for telemetry

```go
func (c *HTTPConnector) Execute(ctx context.Context, step connector.Step) (*connector.Result, error) {
    switch step.Action {
    case "request":
        return c.doRequest(ctx, step)
    }
}
```

### Teardown

The connector cleans up resources. This is always called via `defer`, even if Setup or Execute fails.

```go
func (c *HTTPConnector) Teardown(ctx context.Context) error {
    c.client.CloseIdleConnections()
    return nil
}
```

## YAML Flow Through the System

The complete data flow from YAML on disk to test results:

```
scrutineer.yaml          tests/api.test.yaml
      |                         |
      v                         v
  yaml.Unmarshal()          yaml.Unmarshal()
      |                         |
      v                         v
  schema.Config             raw map[string]any
      |                         |
      v                         v
  config.Merge(flags)       schema.ParseSuite()
      |                     ParseTestStep() for each step
      |                     ValidateSuite()
      |                         |
      v                         v
  Engine construction       []schema.TestSuite
      |                         |
      +----------+--------------+
                 |
                 v
          Engine.Run(ctx, suites)
                 |
                 v
          Runner.runStep()
                 |
                 v
   +-------------+-------------+
   |             |             |
   v             v             v
Interpolate   Connector     Assertions
Variables     .Execute()    .Evaluate()
   |             |             |
   v             v             v
fixture.Store  Result{Data}  AssertionError
   |             |          or nil
   |             v             |
   |         Captures          |
   |         fixture.Extract() |
   |             |             |
   v             v             v
          Reporter.OnStepResult()
          Reporter.OnTestEnd()
          Reporter.OnSuiteEnd()
                 |
                 v
          Reporter.Flush(os.Stdout)
                 |
                 v
          Coverage Gate check
                 |
                 v
          Exit code (0-4)
```

### Parse Phase

The custom YAML parser (`core/yaml`) tokenizes and parses YAML into Go maps and slices. `schema.ParseSuite` then:
1. Unmarshals into a raw `map[string]any` for flexible step parameter extraction
2. Unmarshals into a typed `TestSuite` for strongly-typed fields
3. Re-parses steps from the raw data using `ParseTestStep`, which separates known fields from connector-specific parameters
4. Validates the suite structure

### Engine Phase

The engine creates a `TestContext` per test, which holds a `fixture.Store` containing the suite's fixtures and a captures map. As steps execute, the store accumulates captured values.

### Connector Phase

Parameters are interpolated (replacing `${fixture.x}`, `${capture.y}`, `${env.Z}` with resolved values) before being passed to the connector. The connector returns a `Result` with a `Data` map.

### Assertion Phase

Each assertion map from the YAML is processed by the `DefaultBuilder`, which creates the appropriate `Assertion` implementation based on the `operator` field. The assertion's `Evaluate` method compares the actual value (extracted from `Result.Data` at the `field` path) against the expected value.

### Reporter Phase

Events are delivered in lifecycle order to the reporter. The ANSI reporter accumulates output with color codes for terminal display. The JSON reporter builds a structured object. Both flush their output at the end.

## Thread Safety

- The `Engine` uses a `Pool` (bounded concurrency via goroutines and semaphore) for parallel suite execution
- `SuiteResult` writes are protected by a `sync.Mutex` in `Engine.Run`
- The `coverage.Tracker` is protected by a `sync.Mutex`
- The `loadtest.Metrics` collector is protected by a `sync.Mutex`
- Within a single test, steps execute sequentially (no concurrency)

## Next Steps

- [Extending Scrutineer](extending.md) -- writing custom connectors and assertions
- [YAML Schema Reference](../yaml-schema.md) -- complete schema documentation
- [TLV Format](../tlv-format.md) -- binary telemetry log specification
