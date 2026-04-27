# Scrutineer

An extensible test framework for automating tests against CLI programs and network-based applications. Built in Go with zero third-party dependencies.

## Project

- Repository: `~/git/scrutineer/`
- Language: Go 1.26+ (standard library only)
- Module system: Go workspaces + Go modules
- License: MIT (all consumed tools, binaries, and assets must also be MIT or MIT-compatible)
- Current version: 0.0.1-dev

## Core Rules

### ZERO Third-Party Dependencies

This is a strict, non-negotiable rule. Scrutineer may use ONLY packages maintained as part of Go itself (the standard library). No `golang.org/x/` packages. No external modules. Any feature not provided by the Go standard library must be implemented from scratch within this project.

**Exceptions**:
- `govulncheck` (`golang.org/x/vuln`) — permitted as a build pipeline tool only. It is never imported as a library — it runs as an external CLI during pre-commit checks. This exception exists because there is no adequate stdlib alternative for SAST vulnerability scanning.
- `golang.org/x/crypto/ssh` — permitted as a library dependency. Maintained by the Go team. Used for SSH connectivity (test connector and distributed load testing C2). No adequate stdlib alternative exists.
- `google.golang.org/grpc` — permitted as a library dependency. Maintained by Google (which controls Go). Used for gRPC connectivity and all four RPC modes.
- `google.golang.org/protobuf` — permitted as a library dependency. Maintained by Google. Used for protocol buffer serialization/deserialization and `.proto` file handling.

### Object-Oriented Design

All components use Go's struct + interface patterns to achieve OO design. Interfaces define contracts; structs provide implementations.

### Declarative Tests

Tests are defined in YAML. Users describe *what* to assert, not *how* to execute. Test definitions specify expected states and outcomes rather than step-by-step procedures. Go is used exclusively for building the framework, connectors, and tooling — never for writing tests. The YAML schema is the user-facing API; connectors and integrations are the developer-facing API.

### Modular Architecture

Core test logic (assertions, lifecycle, reporting, coverage) is abstracted from connectors (CLI, HTTP, TLS, SSH, etc.). Adding a new protocol means implementing a connector interface, not modifying the core. Third-party connectors implement Go interfaces and compile into the binary — no runtime plugin loading.

## Build Pipeline

### Pre-Push Checks (Git Hook)

A git pre-push hook (`.githooks/pre-push`) enforces all checks locally before any code is pushed. The hook is version-controlled and activated via `git config core.hooksPath .githooks`.

Before any push, the following must pass:

1. `gofmt` — all files must be formatted
2. `go vet ./...` — static analysis / linting (all modules)
3. `govulncheck ./...` — SAST vulnerability scanning (all modules, skipped if not installed)
4. `go test -race -coverprofile=coverage.out ./...` — tests with race detection (all modules)
5. Coverage gate: **minimum 98% coverage** (target 100%)

Setup: `git config core.hooksPath .githooks`

### CI/CD (GitHub)

- **GitHub Dependabot**: enabled for GitHub Actions workflow version updates (not Go modules — there are none)
- **CodeQL**: enabled for static analysis and security scanning
- Both configured via repository settings and `.github/` workflow files

### Docker

- Docker may be used for testing the project (e.g. standing up test services)
- All Docker images must be built from `ubuntu:latest` as the base — no other images pulled from the internet
- Test service containers (SMTP, IMAP, HTTP servers, etc.) are built from scratch on top of the Ubuntu base

### Test Requirements

- **Minimum 98% code coverage**, ideally 100%
- Every feature must have:
  - Happy path unit tests
  - Sad path unit tests (error conditions, edge cases, invalid input)
  - Integration tests (component interaction)
  - End-to-end tests (full workflow validation)
- **Fuzz testing**: supported as a first-class test type using Go's built-in fuzzing

## Target Capabilities (v0.0.1 — Playwright Feature Parity)

**Deferred to post-0.0.1**: video recording of test runs, HTTP/3 (QUIC), SMTP, IMAP, POP.

### CLI Testing
- stdin/stdout/stderr capture and assertion
- Exit code verification
- File system side-effect validation
- Process lifecycle management

### Network Protocol Testing
- **HTTPS**: HTTP/1.1, HTTP/2 (via Go stdlib `net/http`)
- **TLS**: TLS 1.2 and TLS 1.3 support (via Go stdlib `crypto/tls`)
- **SSH**: key-based auth, command execution, tunneling (via `golang.org/x/crypto/ssh`)
- **SMTP**: deferred to post-0.0.1
- **IMAP**: deferred to post-0.0.1
- **POP**: deferred to post-0.0.1
- Self-signed certificate support for all TLS-based protocols
- Extensible to additional protocols via connector interface

### Browser Automation
- Headless browser control for **Chromium**, **Firefox**, and **WebKit**
- Uses Playwright's open-source patched browser builds (Chromium, Firefox, WebKit) as external binaries — downloaded via `scrutineer browsers install`
- Communication via Chrome DevTools Protocol (CDP) and Playwright's equivalent wire protocols
- Wire protocol client implemented from scratch in Go (JSON-RPC over WebSocket, no Playwright/Selenium library dependency)
- Capabilities matching Playwright:
  - Page navigation, element selection (CSS, XPath, text, role selectors)
  - Click, type, fill, select, file upload, drag-and-drop
  - Wait for selectors, navigation, network idle, load state
  - Network request interception and mocking
  - Cookie and storage manipulation
  - Screenshot and PDF capture
  - Multi-tab / multi-context / incognito support
  - Emulation (viewport, geolocation, permissions, device descriptors)
  - JavaScript evaluation in page context
  - Frame and iframe traversal
  - Trace recording and HAR export
- Browser lifecycle management: `scrutineer browsers install` downloads known-good versions from vendor CDNs. Launch and cleanup handled automatically per test run.

### API Testing
- **RESTful APIs**: full CRUD, header/body/status assertions, JSON/XML response validation, auth (Bearer, Basic, API key), pagination, HATEOAS link following
- **GraphQL**: queries, mutations, subscriptions, variable injection, introspection, error/partial-response assertions (implemented from scratch — no GraphQL libraries)
- **Protobuf/gRPC**: protocol buffer serialization/deserialization, unary and streaming RPCs (client, server, bidirectional), metadata/trailer assertions, status code validation (via `google.golang.org/grpc` and `google.golang.org/protobuf`). Schema via `.proto` file parsing (primary) with gRPC server reflection as fallback.

### Core Features
- Assertion library (equality, contains, regex, JSON path, status codes, headers, timing, etc.)
- Test coverage measurement as a first-class feature
- ANSI color-coded test reporting (default for terminal)
- JSON output format (`--format json`) for machine-readable / CI consumption
- Test suites, setup/teardown hooks, parallel execution
- Selectors, locators, waiting/polling, network interception (Playwright parity)
- Page-object model support
- Fixtures and parameterized tests
- Response capture
- Auto-waiting and retry logic
- Trace and HAR-equivalent recording

### Load Testing
- Parallel test execution similar to Locust
- Distributed across multiple nodes via SSH command-and-control
- Configurable concurrency, ramp-up, and duration

### Benchmarking and Telemetry
- All tests capture timing data automatically (nanosecond precision)
- **Structured binary log format (TLV)**: custom type-length-value encoding with nanosecond timestamps. Each record contains:
  - Nanosecond timestamp (fixed-width)
  - Event type tag (e.g. test-start, test-pass, test-fail, assertion, request, response, error)
  - Event tags (key-value metadata — test name, connector, suite, etc.)
  - Detail blob (variable-length payload — request/response bodies, error messages, stack traces, timing breakdowns)
- Binary log tool (`scrutineer log-dump`) to parse and dump structured binary logs to stdout (like `cat`/`zcat` for scrutineer logs)
- TLV format documented for third-party tool interoperability
- Benchmark analysis from captured timing data

### Fuzz Testing
- Integrated fuzz test support leveraging Go's built-in `testing.F`
- Declarative fuzz target definition
- Corpus management

## Versioning

Semantic versioning (semver):
- **Patch** (0.0.x): bug fixes, config changes
- **Minor** (0.x.0): new features, minor enhancements
- **Major** (x.0.0): significant feature changes, manual determination

## Cross-Platform

Must build cleanly for all major targets without code changes:
- `linux/amd64`, `linux/arm64`
- `darwin/amd64`, `darwin/arm64`
- `windows/amd64`, `windows/arm64`

Use build tags sparingly and only where OS-level differences genuinely require them.

## Configuration

- Project configuration via `scrutineer.yaml` at the project root
- **Test discovery**: explicit manifest — test files are listed in `scrutineer.yaml`, not auto-discovered
- Covers: test file manifest, parallelism, timeouts, reporter format, browser settings, connector defaults, coverage thresholds
- CLI flags override config file values
- Same YAML format as test definitions — one parser, one syntax for users to learn

## Design Philosophy

- Familiar to users of Playwright or assertion-based test frameworks
- Low ceremony: simple tests should be simple to write
- Discoverable API: IDE autocomplete and Go doc should guide users
- Fail fast with clear, actionable error messages
- Test output is human-readable by default, machine-parseable optionally

## Exit Codes

- `0` — all tests passed
- `1` — one or more test assertions failed
- `2` — connection/network error (couldn't reach target)
- `3` — configuration/YAML parse error
- `4` — framework/internal error

Error output is human-friendly first: a clear summary of what went wrong and why, followed by detailed diagnostics (stack traces, request/response dumps, timing data) below.

## Documentation

- **Godoc**: all exported Go types, functions, and interfaces documented in source comments (connector/developer audience)
- **`docs/` directory**: user-facing markdown documentation — getting started guide, YAML schema reference, examples, connector usage

## Coding Standards

Organization coding standards are maintained at: http://coding-standards.asymmetric-effort.com/

All code in this project must comply with those standards. The conventions below are project-specific supplements.

## Development Conventions

- Go 1.26+ required.
- Go workspaces (`go.work`) for multi-module development.
- Keep commits atomic and well-described.
- Write tests alongside implementation code.
- Run full check suite before committing: `go vet`, `govulncheck`, `go test -race -cover`.
- Format with `gofmt`.
- No generated code without clear justification.

## Repository Structure

```
scrutineer/
├── go.work                      # Go workspace definition
├── CLAUDE.md                    # Project specification
├── TODO.md                      # Deferred features
├── LICENSE                      # MIT License
├── Makefile                     # Build, test, vet, vuln, cross-compile
├── .github/
│   └── workflows/
│       ├── ci.yaml              # CI pipeline (test, vet, vuln, codeql)
│       └── dependabot.yml       # GitHub Actions version updates
├── core/                        # Module: scrutineer/core (zero external deps)
│   ├── connector/               # Connector interface, Registry
│   ├── engine/                  # Engine, Runner, Suite, parallel execution
│   ├── yaml/                    # YAML parser (from scratch)
│   ├── schema/                  # Test/Suite/Config types, validation
│   ├── assertion/               # Assertion interface + implementations
│   ├── reporter/                # Reporter interface, ANSI + JSON
│   ├── coverage/                # Coverage tracker + gate
│   ├── fixture/                 # Fixtures, parameterized tests
│   ├── retry/                   # Auto-wait, retry, polling
│   ├── telemetry/               # TLV binary log format
│   ├── config/                  # scrutineer.yaml loading
│   └── exitcode/                # Exit code constants
├── connector/
│   ├── cli/                     # Module: CLI connector
│   ├── http/                    # Module: HTTP/REST/GraphQL connector
│   ├── ssh/                     # Module: SSH connector (x/crypto/ssh)
│   ├── grpc/                    # Module: gRPC connector (google packages)
│   └── browser/                 # Module: Browser automation (CDP)
├── loadtest/                    # Module: Distributed load testing
├── fuzz/                        # Module: Fuzz testing
├── cmd/scrutineer/              # Module: CLI binary entry point
└── docs/                        # User-facing documentation
```
