# Extending Scrutineer

Scrutineer is designed for extensibility. This guide covers writing custom connectors, custom assertions, and integrating them into the build.

## Writing a Custom Connector

### The Connector Interface

Every connector implements the `connector.Connector` interface from `core/connector`:

```go
type Connector interface {
    // Name returns the connector identifier used in YAML (e.g. "http", "cli").
    Name() string

    // Setup initializes the connector with the given configuration.
    Setup(ctx context.Context, config map[string]any) error

    // Execute runs a single test step and returns the result.
    Execute(ctx context.Context, step Step) (*Result, error)

    // Teardown cleans up resources. Always called, even after failures.
    Teardown(ctx context.Context) error
}
```

### Supporting Types

```go
// Step represents a single action within a test.
type Step struct {
    Action     string
    Parameters map[string]any
    Timeout    time.Duration
}

// Result holds the output of a single step execution.
type Result struct {
    Data    map[string]any    // keyed output (e.g. "body", "stdout", "status_code")
    Elapsed time.Duration     // time taken for this step
    Meta    map[string]string // metadata for telemetry
}

// Factory creates a new Connector instance.
type Factory func() Connector
```

### Step-by-Step: Building a Redis Connector

Let us walk through building a hypothetical Redis connector.

#### 1. Create the Module

Create a new directory and Go module:

```
connector/redis/
  go.mod
  redis.go
```

The `go.mod` declares the module and its dependencies:

```
module github.com/scrutineer/scrutineer/connector/redis

go 1.26

require github.com/scrutineer/scrutineer/core v0.0.1
```

Add the module to `go.work`:

```
use (
    ./connector/redis
    // ... existing modules
)
```

#### 2. Implement the Connector

```go
package redis

import (
    "context"
    "fmt"
    "time"

    "github.com/scrutineer/scrutineer/core/connector"
)

// RedisConnector implements connector.Connector for Redis testing.
type RedisConnector struct {
    addr     string
    password string
    db       int
    // conn would be your Redis connection
}

// Compile-time interface check.
var _ connector.Connector = (*RedisConnector)(nil)

// New creates a new RedisConnector.
func New() *RedisConnector {
    return &RedisConnector{}
}

// Name returns the connector identifier.
func (c *RedisConnector) Name() string {
    return "redis"
}
```

#### 3. Implement Setup

Parse configuration from the map. Configuration values come from the `connectors` section of `scrutineer.yaml` or from step-level parameters.

```go
func (c *RedisConnector) Setup(ctx context.Context, config map[string]any) error {
    // Extract required configuration.
    addr, ok := config["addr"].(string)
    if !ok || addr == "" {
        return fmt.Errorf("redis: addr is required")
    }
    c.addr = addr

    // Extract optional configuration.
    if pw, ok := config["password"].(string); ok {
        c.password = pw
    }
    if db, ok := config["db"]; ok {
        switch n := db.(type) {
        case int:
            c.db = n
        case float64:
            c.db = int(n)
        default:
            return fmt.Errorf("redis: db must be an integer")
        }
    }

    // Establish connection (implementation-specific).
    // c.conn = ...

    return nil
}
```

**Important**: Config values from YAML are parsed as `map[string]any`. Numbers may arrive as `int`, `int64`, or `float64` depending on the YAML parser. Always handle multiple numeric types.

#### 4. Implement Execute

Route actions to handler methods. Return results via `connector.Result`.

```go
func (c *RedisConnector) Execute(ctx context.Context, step connector.Step) (*connector.Result, error) {
    start := time.Now()
    var data map[string]any
    var err error

    switch step.Action {
    case "get":
        data, err = c.executeGet(ctx, step)
    case "set":
        data, err = c.executeSet(ctx, step)
    case "del":
        data, err = c.executeDel(ctx, step)
    default:
        return nil, fmt.Errorf("redis: unsupported action %q", step.Action)
    }

    if err != nil {
        return nil, err
    }

    return &connector.Result{
        Data:    data,
        Elapsed: time.Since(start),
        Meta: map[string]string{
            "connector": "redis",
            "action":    step.Action,
        },
    }, nil
}

func (c *RedisConnector) executeGet(ctx context.Context, step connector.Step) (map[string]any, error) {
    key, ok := step.Parameters["key"].(string)
    if !ok {
        return nil, fmt.Errorf("redis: get requires 'key' parameter")
    }

    // value := c.conn.Get(ctx, key)
    value := "example" // placeholder

    return map[string]any{
        "key":   key,
        "value": value,
    }, nil
}
```

#### 5. Implement Teardown

Always clean up resources, even on error:

```go
func (c *RedisConnector) Teardown(ctx context.Context) error {
    // if c.conn != nil {
    //     return c.conn.Close()
    // }
    return nil
}
```

#### 6. Construct Result Data

The `Result.Data` map is what users interact with in assertions and captures. Design it carefully:

- Use clear, consistent key names
- Include the raw data users need for assertions
- Include timing information where relevant
- Use types that work well with assertions (strings, ints, maps)

```go
// Good: clear keys, useful for assertions
data := map[string]any{
    "key":    key,
    "value":  value,
    "exists": true,
    "ttl":    300,
}

// Bad: opaque or deeply nested structure
data := map[string]any{
    "result": map[string]any{
        "data": map[string]any{
            "raw_bytes": []byte{...},
        },
    },
}
```

### Registration

Register the connector in `cmd/scrutineer/main.go`:

```go
import connRedis "github.com/scrutineer/scrutineer/connector/redis"

func registerConnectors(registry *connector.Registry) {
    registry.Register("cli", func() connector.Connector { return connCLI.New() })
    registry.Register("http", func() connector.Connector { return connHTTP.New() })
    registry.Register("ssh", func() connector.Connector { return connSSH.New() })
    registry.Register("grpc", func() connector.Connector { return connGRPC.New() })
    registry.Register("browser", func() connector.Connector { return connBrowser.New() })
    registry.Register("redis", func() connector.Connector { return connRedis.New() })
}
```

The `Registry` uses a factory pattern. Each call to `Registry.Get(name)` creates a fresh connector instance, so connectors do not need to handle concurrent use across tests.

### The Connector Registry Pattern

The registry maps connector names (strings used in YAML) to factory functions:

```go
type Registry struct {
    factories map[string]Factory
}

func (r *Registry) Register(name string, f Factory) error
func (r *Registry) Get(name string) (Connector, error)
func (r *Registry) Names() []string
```

- `Register` adds a factory. Returns an error if the name is already registered (prevents silent overwrites).
- `Get` creates a new instance from the factory. Returns an error if the name is not registered.
- `Names` returns all registered names in sorted order.

This pattern decouples the engine from specific connector implementations. The engine works entirely through the `Connector` interface.

### Using the Connector in YAML

Once registered, the connector is available in test files:

```yaml
# scrutineer.yaml
connectors:
  redis:
    addr: "localhost:6379"
    password: "secret"
    db: 0

# tests/redis.test.yaml
suite: "Redis Operations"

tests:
  - name: "Set and get a key"
    connector: redis
    steps:
      - action: set
        key: "test_key"
        value: "hello"
        assert:
          - field: ok
            operator: equal
            expected: true

      - action: get
        key: "test_key"
        assert:
          - field: value
            operator: equal
            expected: "hello"
```

## Writing Custom Assertions

### The Assertion Interface

```go
type Assertion interface {
    Name() string
    Evaluate(actual any) error
}
```

`Evaluate` receives the actual value (extracted from the result data at the field path specified in the YAML) and returns nil if the assertion passes, or an `*AssertionError` if it fails.

### Implementing an Assertion

```go
// BetweenAssertion checks that a numeric value falls between two bounds (exclusive).
type BetweenAssertion struct {
    Lower float64
    Upper float64
}

func (a *BetweenAssertion) Name() string {
    return "between"
}

func (a *BetweenAssertion) Evaluate(actual any) error {
    val, err := toFloat64(actual)
    if err != nil {
        return &AssertionError{
            Assertion: "between",
            Expected:  fmt.Sprintf("(%v, %v)", a.Lower, a.Upper),
            Actual:    actual,
            Message:   fmt.Sprintf("cannot convert to number: %v", err),
        }
    }

    if val <= a.Lower || val >= a.Upper {
        return &AssertionError{
            Assertion: "between",
            Expected:  fmt.Sprintf("(%v, %v)", a.Lower, a.Upper),
            Actual:    val,
            Message:   fmt.Sprintf("value %v is not between %v and %v", val, a.Lower, a.Upper),
        }
    }

    return nil
}
```

### Registering Custom Assertions

Add a case to the `DefaultBuilder.Build` method in `core/assertion/assertion.go`:

```go
case "between":
    lower, _ := options["lower"].(float64)
    upper, _ := options["upper"].(float64)
    return &BetweenAssertion{Lower: lower, Upper: upper}, nil
```

### Using in YAML

```yaml
assert:
  - field: body.score
    operator: between
    expected: null
    lower: 0.0
    upper: 100.0
```

### Error Messages

Assertion errors should be clear and actionable. Use the `AssertionError` struct:

```go
type AssertionError struct {
    Assertion string  // operator name
    Expected  any     // what was expected
    Actual    any     // what was received
    Message   string  // human-friendly explanation
    Path      string  // optional dot-notation path
}
```

The error message format:

```
assertion "between" failed: value 105.3 is not between 0 and 100 (expected: (0, 100), actual: 105.3)
```

With a path:

```
assertion "equal" failed at path "body.user.name": values are not equal (expected: Alice, actual: Bob)
```

## Build Integration

### Adding to the Makefile

Add your module to the `MODULES` list in the Makefile:

```makefile
MODULES := core connector/cli connector/http connector/ssh connector/grpc connector/browser connector/redis loadtest fuzz cmd/scrutineer
```

This ensures your module is included in `make fmt`, `make vet`, `make vuln`, `make test`, and `make coverage`.

### Adding to go.work

```
use (
    ./cmd/scrutineer
    ./connector/browser
    ./connector/cli
    ./connector/grpc
    ./connector/http
    ./connector/redis
    ./connector/ssh
    ./core
    ./fuzz
    ./loadtest
)
```

### Adding to Pre-Push Hook

The pre-push hook in `.githooks/pre-push` reads the `MODULES` variable. If you add the module to the Makefile's `MODULES`, the hook will automatically include it. If you maintain the hook's list separately, update it:

```bash
MODULES="core connector/cli connector/http connector/ssh connector/grpc connector/browser connector/redis loadtest fuzz cmd/scrutineer"
```

### Test Requirements

Every connector must have:
- Unit tests for all public methods (happy path and error cases)
- Coverage of at least 98%
- Race-safe code (tested via `go test -race`)

### Dependency Rules

- If your connector uses only the Go standard library, no additional dependencies are needed
- If you need an external dependency, document the justification and ensure the dependency is maintained by a trusted party (Go team, Google, etc.)
- Never add dependencies to the `core` module

## Next Steps

- [Architecture Overview](overview.md) -- system architecture and data flow
- [Developer Guide](../contributing/development.md) -- development setup and conventions
- [YAML Schema Reference](../yaml-schema.md) -- test file format
