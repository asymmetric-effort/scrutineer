# Getting Started

## Installation

```bash
go install github.com/scrutineer/scrutineer/cmd/scrutineer@latest
```

Or download a pre-built binary from [Releases](https://github.com/scrutineer/scrutineer/releases).

## Quick Start

### 1. Create a config file

Create `scrutineer.yaml` in your project root:

```yaml
version: "0.0.1"

tests:
  - tests/api.test.yaml

parallelism: 4
timeout: 30s

reporters:
  - type: ansi

telemetry:
  enabled: true
  output: scrutineer.log
```

### 2. Write a test

Create `tests/api.test.yaml`:

```yaml
suite: "API Smoke Tests"
tags: [api, smoke]

tests:
  - name: "Health check returns 200"
    connector: http
    steps:
      - action: request
        method: GET
        path: /health
        assert:
          - status: 200

  - name: "Create user"
    connector: http
    steps:
      - action: request
        method: POST
        path: /users
        body:
          name: "Alice"
          email: "alice@example.com"
        assert:
          - status: 201
          - body.name: {equals: "Alice"}
        capture:
          user_id: body.id

      - action: request
        method: GET
        path: /users/${capture.user_id}
        assert:
          - status: 200
          - body.email: {equals: "alice@example.com"}
```

### 3. Run tests

```bash
scrutineer run
```

### Options

```bash
# Use a specific config file
scrutineer run --config path/to/config.yaml

# JSON output for CI
scrutineer run --format json

# Run tests with specific tags
scrutineer run --tags smoke

# Set parallelism
scrutineer run --parallelism 8

# View binary telemetry logs
scrutineer log-dump scrutineer.log
```

## Connectors

Scrutineer ships with these connectors:

| Connector | Name | Use Case |
|-----------|------|----------|
| CLI | `cli` | Test command-line programs |
| HTTP | `http` | Test REST APIs, web endpoints |
| SSH | `ssh` | Test remote systems |
| gRPC | `grpc` | Test gRPC services |
| Browser | `browser` | Test web UIs (Chromium, Firefox, WebKit) |

Each connector is configured in `scrutineer.yaml` under the `connectors` key and used in test files via the `connector` field.

## Next Steps

- [YAML Schema Reference](yaml-schema.md) — full test file syntax
- [Connectors Guide](connectors.md) — detailed connector configuration
- [TLV Format](tlv-format.md) — binary log format specification
