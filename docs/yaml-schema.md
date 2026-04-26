# YAML Schema Reference

## Configuration File (`scrutineer.yaml`)

```yaml
version: "0.0.1"                    # scrutineer version

tests:                               # test file manifest (explicit)
  - tests/api.test.yaml
  - tests/cli.test.yaml

parallelism: 4                       # concurrent test execution (default: 1)
timeout: 30s                         # default step timeout

reporters:
  - type: ansi                       # terminal output (default)
  - type: json                       # machine-readable output
    output: results.json

coverage:
  threshold: 98.0                    # minimum coverage percentage

browsers:
  chromium: true
  firefox: false
  webkit: false

connectors:
  http:
    base_url: "https://api.example.com"
    default_headers:
      Authorization: "Bearer ${env.API_TOKEN}"
    timeout: "10s"
    tls_skip_verify: false
  ssh:
    host: "test-server.local"
    port: 22
    user: "testuser"
    key_file: "~/.ssh/id_ed25519"
  grpc:
    endpoint: "localhost:50051"
    plaintext: true

telemetry:
  enabled: true
  output: scrutineer.log
```

## Test File

### Top-Level Structure

```yaml
suite: "Suite Name"                  # required
tags: [tag1, tag2]                   # optional, for filtering

setup:                               # optional, runs before all tests
  - connector: http
    action: request
    method: POST
    path: /reset

teardown:                            # optional, runs after all tests
  - connector: http
    action: request
    method: POST
    path: /cleanup

fixtures:                            # optional, reusable data
  user:
    name: "Alice"
    email: "alice@example.com"

tests:                               # required, at least one test
  - name: "Test name"               # required
    connector: http                  # required
    tags: [smoke]                    # optional
    skip: false                      # optional
    steps:                           # required, at least one step
      - action: request
        # ... step-specific fields
```

### Variable Interpolation

```yaml
# Fixture reference
path: /users/${fixture.user.name}

# Capture reference (from a previous step)
path: /users/${capture.user_id}

# Environment variable
path: /api/${env.API_VERSION}/users
```

### Assertions

Assertions are evaluated against step results.

```yaml
assert:
  # Direct value assertions
  - status: 200
  - exit_code: 0

  # Operator assertions
  - body.name: {equals: "Alice"}
  - body.name: {not_equal: "Bob"}
  - body.name: {contains: "Ali"}
  - body.name: {has_prefix: "Al"}
  - body.name: {has_suffix: "ce"}
  - body.name: {matches: "^[A-Z][a-z]+$"}

  # Numeric assertions
  - body.age: {greater_than: 18}
  - body.age: {less_than: 100}
  - body.age: {in_range: [18, 65]}

  # Collection assertions
  - body.items: {length: 5}
  - body.items: {not_empty: true}

  # Existence assertions
  - body.id: {not_empty: true}

  # Timing assertions
  - elapsed: {less_than: 2s}

  # Header assertions (HTTP)
  - header.Content-Type: {contains: "application/json"}
```

### Captures

Extract values from step results for use in subsequent steps.

```yaml
capture:
  user_id: body.id
  token: body.auth.token
  count: body.total
```

### Step Actions by Connector

#### HTTP Connector

```yaml
- action: request
  method: GET|POST|PUT|DELETE|PATCH
  path: /endpoint
  headers:
    X-Custom: "value"
  query:
    page: "1"
    limit: "10"
  body:                              # string or map (auto-JSON)
    key: "value"
  auth:
    type: bearer|basic|api_key
    token: "..."                     # for bearer
    username: "..."                  # for basic
    password: "..."                  # for basic
    header: "X-API-Key"             # for api_key
    key: "..."                       # for api_key
```

#### CLI Connector

```yaml
- action: exec
  command: "echo hello"
  stdin: "input data"

- action: filesystem
  path: /tmp/output.txt
  exists: true
  contains: "expected content"
  size:
    greater_than: 0
    less_than: 1048576
```

#### SSH Connector

```yaml
- action: exec
  command: "uptime"
  stdin: "optional input"

- action: tunnel
  local_port: 8080
  remote_host: "localhost"
  remote_port: 5432
```

#### gRPC Connector

```yaml
- action: unary
  service: "myapp.UserService"
  method: "GetUser"
  message:
    id: 1
  metadata:
    authorization: "Bearer token"

- action: server_stream
  service: "myapp.EventService"
  method: "Subscribe"
  message:
    topic: "events"

- action: client_stream
  service: "myapp.UploadService"
  method: "Upload"
  messages:
    - {chunk: "part1"}
    - {chunk: "part2"}

- action: bidi_stream
  service: "myapp.ChatService"
  method: "Chat"
  messages:
    - {text: "hello"}
    - {text: "world"}
```

#### Browser Connector

```yaml
- action: navigate
  url: "https://example.com"

- action: click
  selector: "#submit-button"

- action: type
  selector: "#search-input"
  text: "search query"

- action: fill
  selector: "#email"
  value: "user@example.com"

- action: screenshot
  path: "screenshot.png"

- action: evaluate
  expression: "document.title"

- action: wait_for_selector
  selector: ".loaded"
  timeout: 5s

- action: get_text
  selector: ".message"

- action: get_attribute
  selector: "img"
  attribute: "src"
```
