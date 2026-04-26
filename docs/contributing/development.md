# Developer Guide

This guide covers setting up a development environment, running tests, and following the project's conventions.

## Prerequisites

- **Go 1.26+** -- required for workspace support and language features
- **Git** -- version control
- **Make** -- build automation
- **govulncheck** (optional but recommended) -- vulnerability scanning

Install govulncheck:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

## Repository Setup

### Clone

```bash
git clone https://github.com/scrutineer/scrutineer.git
cd scrutineer
```

### Configure Git Hooks

The project includes a pre-push hook that enforces code quality before pushing:

```bash
git config core.hooksPath .githooks
```

This activates the pre-push hook at `.githooks/pre-push`, which runs:

1. `gofmt` -- all files must be formatted
2. `go vet ./...` -- static analysis for all modules
3. `govulncheck ./...` -- vulnerability scanning (skipped if not installed)
4. `go test -race -coverprofile=coverage.out ./...` -- tests with race detection
5. Coverage gate -- minimum 98% coverage per module

### Go Workspace

The project uses Go workspaces (`go.work`) for multi-module development. The workspace file references all nine modules:

```
go 1.26.2

use (
    ./cmd/scrutineer
    ./connector/browser
    ./connector/cli
    ./connector/grpc
    ./connector/http
    ./connector/ssh
    ./core
    ./fuzz
    ./loadtest
)
```

With the workspace in place, cross-module imports resolve locally. You can edit any module and all dependent modules see the changes immediately.

## Running Tests

### All Modules

```bash
make test
```

This runs `go test -race -coverprofile=coverage.out ./...` in each module.

### Single Module

```bash
cd core && go test -race ./...
cd connector/http && go test -race ./...
```

### With Coverage Report

```bash
cd core && go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out   # opens in browser
```

### Coverage Gate

```bash
make coverage
```

This runs all tests and then checks that each module meets the 98% coverage threshold. If any module falls below 98%, the command fails with an error message like:

```
FAIL: connector/http coverage 96.5% < 98%
```

## Code Style

### Formatting

All code must be formatted with `gofmt`. Check formatting:

```bash
gofmt -l .
```

Format all files:

```bash
make fmt
```

### Static Analysis

```bash
make vet
```

This runs `go vet ./...` in every module.

### Vulnerability Scanning

```bash
make vuln
```

This runs `govulncheck ./...` in every module. Requires govulncheck to be installed.

### Full Pre-Commit Suite

Run everything the pre-push hook checks:

```bash
make precommit
```

This runs: `fmt`, `vet`, `vuln`, `test`, `coverage`.

## Coverage Requirements

The project requires a **minimum of 98% code coverage**, with 100% as the target. Every feature must have:

- **Happy path unit tests** -- normal operation
- **Sad path unit tests** -- error conditions, edge cases, invalid input
- **Integration tests** -- component interaction
- **End-to-end tests** -- full workflow validation

When adding new code, always add corresponding tests. The pre-push hook will reject pushes that drop below 98%.

## Module Boundaries

### Core Module Rules

The `core` module has **zero external dependencies**. It imports only Go standard library packages. This is a strict rule -- never add external imports to any package under `core/`.

### Connector Module Rules

Each connector is an independent module in `connector/<name>/`. Connectors:
- Import `core/connector` for the interface and types
- May import approved external dependencies (see below)
- Must not import other connectors
- Must not be imported by the core module

### Approved External Dependencies

| Dependency                      | Used By            | Reason                          |
|---------------------------------|--------------------|---------------------------------|
| `golang.org/x/crypto/ssh`      | `connector/ssh`    | SSH protocol implementation     |
| `google.golang.org/grpc`       | `connector/grpc`   | gRPC protocol implementation    |
| `google.golang.org/protobuf`   | `connector/grpc`   | Protocol buffer handling        |

All other external dependencies are prohibited. If you need functionality not in the standard library, implement it from scratch.

### Loadtest and Fuzz Modules

- `loadtest` depends on `core/connector` and `connector/ssh` (for distributed testing)
- `fuzz` depends on `core/connector` (for executing fuzz inputs)
- Neither should depend on specific connector implementations beyond their stated needs

## Adding a New Package

### Within an Existing Module

1. Create the directory under the module (e.g., `core/newpackage/`)
2. Add a `doc.go` or main `.go` file with the package declaration
3. Write implementation and tests
4. Ensure tests pass with race detection: `go test -race ./...`
5. Ensure coverage meets 98%

### As a New Module

1. Create the directory (e.g., `connector/newconn/`)
2. Initialize the module: `cd connector/newconn && go mod init github.com/scrutineer/scrutineer/connector/newconn`
3. Add the module to `go.work`:
   ```
   use (
       ./connector/newconn
       // ... existing modules
   )
   ```
4. Add the module to the `MODULES` list in `Makefile`
5. Update `.githooks/pre-push` if it maintains a separate module list
6. Implement the connector (see [Extending Scrutineer](../architecture/extending.md))
7. Register it in `cmd/scrutineer/main.go`
8. Write comprehensive tests

## Build Commands

| Command         | Description                                          |
|-----------------|------------------------------------------------------|
| `make build`    | Build the scrutineer binary into `bin/scrutineer`    |
| `make cross`    | Cross-compile for all 6 platform/arch combinations   |
| `make test`     | Run tests with race detection in all modules         |
| `make coverage` | Run tests and enforce 98% coverage gate              |
| `make fmt`      | Format all Go files with gofmt                       |
| `make vet`      | Run go vet on all modules                            |
| `make vuln`     | Run govulncheck on all modules                       |
| `make clean`    | Remove binaries and coverage files                   |
| `make precommit`| Run all checks (fmt, vet, vuln, test, coverage)      |
| `make all`      | Run fmt, vet, test, build                            |

## Running govulncheck

govulncheck scans for known vulnerabilities in your dependencies:

```bash
# Install (one-time)
go install golang.org/x/vuln/cmd/govulncheck@latest

# Run on all modules
make vuln

# Run on a single module
cd connector/grpc && govulncheck ./...
```

govulncheck is the only permitted `golang.org/x/` tool. It runs as an external CLI -- it is never imported as a library.

## Design Conventions

### Interface-First Design

Define interfaces before implementations. Use Go's struct + interface patterns for OO design:

```go
// Interface in core
type Reporter interface {
    OnSuiteStart(suite SuiteInfo)
    OnTestEnd(test TestInfo, result TestResult)
    Flush(w io.Writer) error
}

// Implementation
type ANSIReporter struct { ... }
var _ Reporter = (*ANSIReporter)(nil) // compile-time check
```

### Functional Options

Use the functional options pattern for configurable constructors:

```go
type Option func(*Engine)

func WithParallelism(n int) Option {
    return func(e *Engine) { e.parallelism = n }
}

func New(opts ...Option) *Engine { ... }
```

### Error Handling

Return clear, actionable errors. Wrap errors with context:

```go
return fmt.Errorf("redis: get key %q: %w", key, err)
```

Use sentinel errors or typed errors where callers need to inspect error types.

### Test File Organization

Place tests alongside the code they test:

```
core/assertion/
  assertion.go
  assertion_test.go
  contains.go
  contains_test.go
```

### Godoc

All exported types, functions, and interfaces must have documentation comments. The first sentence should be a complete sentence starting with the name of the thing being documented:

```go
// Registry maps connector names to their factories.
type Registry struct { ... }

// Register adds a connector factory under the given name.
// Returns an error if name is already registered.
func (r *Registry) Register(name string, f Factory) error { ... }
```

## Next Steps

- [Versioning Policy](versioning.md) -- release process and semver rules
- [Architecture Overview](../architecture/overview.md) -- system design
- [Extending Scrutineer](../architecture/extending.md) -- adding connectors and assertions
