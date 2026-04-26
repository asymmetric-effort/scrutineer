# Connectors Guide

## Overview

Connectors are the bridge between scrutineer's declarative test engine and the systems under test. Each connector implements a uniform interface: Setup, Execute, Teardown.

## HTTP Connector

Tests HTTP/1.1 and HTTP/2 endpoints with full TLS support.

### Configuration

```yaml
connectors:
  http:
    base_url: "https://api.example.com"
    default_headers:
      Authorization: "Bearer ${env.TOKEN}"
    timeout: "10s"
    tls_skip_verify: false       # for self-signed certs
    tls_ca_file: "ca.pem"       # custom CA
    tls_cert_file: "client.pem" # mTLS client cert
    tls_key_file: "client-key.pem"
```

### Features
- REST API testing (GET, POST, PUT, DELETE, PATCH)
- JSON and XML response validation
- Authentication (Bearer, Basic, API key)
- GraphQL queries, mutations, subscriptions, introspection
- TLS 1.2 and TLS 1.3
- Self-signed certificate support
- mTLS (mutual TLS)

## CLI Connector

Tests command-line programs via process execution.

### Configuration

```yaml
connectors:
  cli:
    work_dir: "/path/to/project"
    env:
      APP_ENV: "test"
```

### Features
- stdin/stdout/stderr capture
- Exit code verification
- Filesystem side-effect validation
- Process timeout management

## SSH Connector

Tests remote systems via SSH.

### Configuration

```yaml
connectors:
  ssh:
    host: "server.example.com"
    port: 22
    user: "deploy"
    key_file: "~/.ssh/id_ed25519"
    host_key_check: false
```

### Features
- Key-based and password authentication
- Remote command execution
- SSH tunneling / port forwarding

## gRPC Connector

Tests gRPC services with protobuf support.

### Configuration

```yaml
connectors:
  grpc:
    endpoint: "localhost:50051"
    plaintext: true              # no TLS
    tls: true                    # enable TLS
    tls_skip_verify: false
    tls_ca_file: "ca.pem"
    schema: "service.proto"      # optional .proto file
```

### Features
- Unary RPCs
- Server streaming, client streaming, bidirectional streaming
- Schema via .proto files or gRPC server reflection
- Metadata and trailer assertions
- gRPC status code validation

## Browser Connector

Tests web UIs via headless browsers using the Chrome DevTools Protocol.

### Configuration

```yaml
connectors:
  browser:
    browser: "chromium"          # chromium, firefox, webkit
    headless: true

browsers:
  chromium: true
  firefox: false
  webkit: false
```

### Setup

```bash
# Download browser binaries
scrutineer browsers install
```

### Features
- Page navigation
- Element selection (CSS, XPath, text, ARIA role)
- Click, type, fill, select interactions
- Wait for selectors and navigation
- Screenshot capture
- JavaScript evaluation
- Network request interception (planned)
- Multi-tab support (planned)

## Writing Custom Connectors

Implement the `connector.Connector` interface:

```go
type Connector interface {
    Name() string
    Setup(ctx context.Context, config map[string]any) error
    Execute(ctx context.Context, step Step) (*Result, error)
    Teardown(ctx context.Context) error
}
```

Register in `cmd/scrutineer/main.go`:

```go
registry.Register("myconnector", func() connector.Connector {
    return myconnector.New()
})
```
