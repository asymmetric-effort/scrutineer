# Writing Tests

This guide walks you through writing tests with scrutineer, from your first test file to advanced patterns like parameterized tests and multi-step workflows with captures.

## How Tests Work

Tests in scrutineer are declarative YAML files. You describe what to test and what to assert -- the engine handles execution. Each test file is a **suite** containing one or more **tests**, and each test contains one or more **steps**. Steps are executed in order by the appropriate connector (HTTP, CLI, SSH, gRPC, or browser).

## Test File Structure

A complete test file has this structure:

```yaml
suite: "User API"                       # Suite name (required)
tags: [api, smoke]                      # Suite-level tags (optional)

fixtures:                               # Reusable data (optional)
  admin:
    username: "admin"
    password: "secret"

setup:                                  # Runs before all tests (optional)
  - connector: http
    action: request
    method: POST
    path: /test/reset

teardown:                               # Runs after all tests (optional)
  - connector: http
    action: request
    method: POST
    path: /test/cleanup

tests:                                  # Test cases (required, at least one)
  - name: "Get user list"
    connector: http
    tags: [smoke]
    steps:
      - action: request
        method: GET
        path: /users
        assert:
          - field: status
            operator: equal
            expected: 200
```

### Suite (Required)

The `suite` field is a human-readable name for the test file. It appears in test output and telemetry.

### Tags (Optional)

Tags let you filter which tests to run. Tags can be set at the suite level (applied to all tests) and at the individual test level. Use `--tags` on the command line to filter:

```bash
scrutineer run --tags smoke,api
```

### Fixtures (Optional)

Fixtures are reusable data defined once and referenced throughout the suite using `${fixture.path.to.value}` syntax:

```yaml
fixtures:
  user:
    name: "Alice"
    email: "alice@example.com"
  endpoints:
    base: "/api/v1"

tests:
  - name: "Create user"
    connector: http
    steps:
      - action: request
        method: POST
        path: ${fixture.endpoints.base}/users
        body:
          name: ${fixture.user.name}
          email: ${fixture.user.email}
```

### Setup and Teardown (Optional)

Setup steps run once before all tests in the suite. Teardown steps run once after all tests, even if tests fail. Both use the same step syntax as test steps.

Common uses:
- **Setup**: seed a database, create test users, start a service
- **Teardown**: clean up test data, reset state

### Tests (Required)

Each test has:
- `name` (required): a descriptive name
- `connector` (required at test or step level): which connector to use
- `tags` (optional): test-level tags for filtering
- `skip` (optional): set to `true` to skip the test
- `steps` (required): at least one step

## Writing Steps

A step is a single action executed by a connector. Every step requires an `action` field. All other fields (except `assert`, `capture`, `connector`, and `timeout`) are passed as parameters to the connector.

### Basic Step

```yaml
- action: request
  method: GET
  path: /health
```

### Step with Connector Override

If a step uses a different connector than the test-level default, specify it explicitly:

```yaml
tests:
  - name: "Full workflow"
    connector: http
    steps:
      - action: request                 # Uses http connector (test default)
        method: POST
        path: /users
        body:
          name: "Alice"

      - connector: cli                  # Override: uses cli connector
        action: exec
        command: "echo 'User created'"
```

### Step Timeout

Override the default timeout for a specific step:

```yaml
- action: request
  method: GET
  path: /slow-endpoint
  timeout: 60s
```

## Using Assertions

Assertions verify that step results match expectations. Each assertion is a map with three fields:

| Field      | Description                                    |
|------------|------------------------------------------------|
| `field`    | Dot-notation path into the result data         |
| `operator` | The comparison operator                        |
| `expected` | The expected value                             |

```yaml
assert:
  - field: status
    operator: equal
    expected: 200
  - field: body.name
    operator: equal
    expected: "Alice"
```

### Available Operators

**Equality:**
- `equal` / `eq` -- exact equality
- `not_equal` / `neq` -- not equal
- `deep_equal` -- deep structural equality for maps and slices

**String:**
- `contains` -- substring match
- `not_contains` -- substring absence
- `has_prefix` -- starts with
- `has_suffix` -- ends with
- `matches` -- regular expression match

**Numeric:**
- `greater_than` / `gt` -- strictly greater
- `less_than` / `lt` -- strictly less
- `greater_or_equal` / `gte` -- greater or equal
- `less_or_equal` / `lte` -- less or equal
- `in_range` -- within a range (requires `min` and `max` options)

**Collections:**
- `length` -- exact length of a string, slice, or map
- `empty` -- value is empty
- `not_empty` -- value is not empty
- `collection_not_empty` -- collection has at least one element

**HTTP-specific:**
- `status_code` -- HTTP status code check
- `status_class` -- HTTP status class (e.g. "2xx", "4xx")
- `header_equals` -- header value equality (requires `header` option)
- `header_contains` -- header value substring (requires `header` option)
- `header_exists` -- header presence check

**JSON:**
- `json_path` -- extract and assert a JSON path value (requires `expected` option)

**Timing:**
- `response_time_below` -- response completed within duration

### Assertion Options

Some operators take additional options beyond `field`, `operator`, and `expected`:

```yaml
# Range check
- field: body.age
  operator: in_range
  expected: null
  min: 18
  max: 65

# Header check
- field: headers
  operator: header_equals
  expected: "application/json"
  header: "Content-Type"

# JSON path
- field: body
  operator: json_path
  expected: "$.user.name"
  expected: "Alice"
```

## Captures and Variable Interpolation

Captures extract values from step results and store them for use in later steps. This is how you chain multi-step workflows.

### Defining Captures

```yaml
- action: request
  method: POST
  path: /users
  body:
    name: "Alice"
  capture:
    user_id: body.id
    auth_token: body.token
```

The `capture` field is a map where keys are variable names and values are dot-notation paths into the result data.

### Using Captured Values

Reference captured values with `${capture.variable_name}`:

```yaml
- action: request
  method: GET
  path: /users/${capture.user_id}
  headers:
    Authorization: "Bearer ${capture.auth_token}"
  assert:
    - field: status
      operator: equal
      expected: 200
```

### Variable Sources

Three variable sources are available:

| Prefix     | Source                        | Example                       |
|------------|-------------------------------|-------------------------------|
| `fixture`  | Suite fixtures section        | `${fixture.user.email}`       |
| `capture`  | Captured from previous steps  | `${capture.user_id}`          |
| `env`      | Environment variables         | `${env.API_KEY}`              |

Variables are interpolated recursively in all string values within step parameters, including nested maps and lists.

### Escaping

To include a literal `${` in a value, escape with a backslash:

```yaml
body: "The syntax is \${variable}"
```

## Parameterized Tests

Parameterized tests let you run the same test logic with different inputs. Define parameter sets and scrutineer expands them into separate test executions.

Each parameter set has a `name` (used in the expanded test name) and `values` (a map of parameters). The expanded test name follows the pattern `"Original Name [parameter set name]"`.

This feature uses deep copying so each expanded test instance operates on independent data.

## Result Data by Connector

Each connector produces a specific set of result data keys that you can assert on and capture from.

### HTTP Connector

| Key           | Type               | Description                          |
|---------------|--------------------|--------------------------------------|
| `status`      | int                | HTTP status code                     |
| `status_text` | string             | Full status text (e.g. "200 OK")     |
| `headers`     | map[string][]string| Response headers                     |
| `body`        | any                | Parsed JSON body (or raw string)     |
| `body_raw`    | string             | Raw response body string             |
| `elapsed_ms`  | float64            | Request duration in milliseconds     |

### CLI Connector

| Key         | Type   | Description                    |
|-------------|--------|--------------------------------|
| `stdout`    | string | Standard output                |
| `stderr`    | string | Standard error                 |
| `exit_code` | int    | Process exit code              |
| `command`   | string | The command that was executed   |

### CLI Filesystem Action

| Key         | Type   | Description                    |
|-------------|--------|--------------------------------|
| `exists`    | bool   | Whether the path exists        |
| `size`      | int64  | File size in bytes             |
| `is_dir`    | bool   | Whether the path is a directory|
| `content`   | string | File content (if regular file) |
| `contains`  | bool   | Whether content has substring  |

### gRPC Connector

| Key              | Type              | Description                   |
|------------------|-------------------|-------------------------------|
| `status_code`    | int               | gRPC status code              |
| `status_message` | string            | gRPC status message           |
| `status_name`    | string            | gRPC status code name         |
| `response`       | map               | Response message (unary)      |
| `responses`      | []map             | Response messages (streaming) |
| `metadata`       | map[string][]string | Response metadata (headers) |
| `trailers`       | map[string][]string | Response trailers           |

### Browser Connector

Result data varies by action:

| Action              | Key     | Description                        |
|---------------------|---------|------------------------------------|
| `navigate`          | `url`   | Navigated URL                      |
| `evaluate`          | `value` | JavaScript return value            |
| `get_text`          | `text`  | Element inner text                 |
| `get_attribute`     | `value` | Attribute value                    |
| `screenshot`        | `data`  | Base64-encoded image data          |
| `screenshot`        | `path`  | File path (if saved to disk)       |

## Test Organization Best Practices

### File Naming

Use descriptive names with a `.test.yaml` suffix:

```
tests/
  api-users.test.yaml
  api-auth.test.yaml
  cli-commands.test.yaml
  browser-login.test.yaml
```

### Suite Naming

Use clear, hierarchical names:

```yaml
suite: "API / Users / CRUD"
suite: "CLI / File Operations"
suite: "Browser / Authentication"
```

### Test Naming

Name tests after the behavior being verified:

```yaml
# Good
- name: "Returns 404 for non-existent user"
- name: "Creates user with valid input"
- name: "Rejects duplicate email"

# Avoid
- name: "Test 1"
- name: "GET /users"
```

### One Concern Per Test

Each test should verify one behavior. If you find a test with many unrelated assertions, split it:

```yaml
# Prefer this: separate tests for separate concerns
- name: "Returns correct user data"
  steps:
    - action: request
      method: GET
      path: /users/1
      assert:
        - field: body.name
          operator: equal
          expected: "Alice"

- name: "Returns correct content type"
  steps:
    - action: request
      method: GET
      path: /users/1
      assert:
        - field: headers
          operator: header_contains
          expected: "application/json"
          header: "Content-Type"
```

### Use Tags for Organization

```yaml
# Manifest in scrutineer.yaml
tests:
  - tests/api-users.test.yaml
  - tests/api-auth.test.yaml
  - tests/browser-login.test.yaml

# Run subsets
scrutineer run --tags smoke          # just smoke tests
scrutineer run --tags api            # just API tests
scrutineer run --tags browser        # just browser tests
```

## Walkthrough: Building a REST API Test Suite

Let us build a complete test suite for a user management API step by step.

### Step 1: Create the Project Config

Create `scrutineer.yaml` in your project root:

```yaml
version: "0.0.1"
tests:
  - tests/users.test.yaml
parallelism: 1
timeout: 10s
reporters:
  - type: ansi
connectors:
  http:
    base_url: "http://localhost:8080"
    default_headers:
      Content-Type: "application/json"
```

### Step 2: Write the Test Suite

Create `tests/users.test.yaml`:

```yaml
suite: "User Management API"
tags: [api, users]

fixtures:
  new_user:
    name: "Alice Smith"
    email: "alice@example.com"

tests:
  - name: "Create a new user"
    connector: http
    tags: [smoke]
    steps:
      - action: request
        method: POST
        path: /api/users
        body:
          name: ${fixture.new_user.name}
          email: ${fixture.new_user.email}
        assert:
          - field: status
            operator: equal
            expected: 201
          - field: body.name
            operator: equal
            expected: "Alice Smith"
          - field: body.id
            operator: not_empty
        capture:
          user_id: body.id

  - name: "Retrieve the created user"
    connector: http
    steps:
      - action: request
        method: POST
        path: /api/users
        body:
          name: ${fixture.new_user.name}
          email: ${fixture.new_user.email}
        capture:
          user_id: body.id

      - action: request
        method: GET
        path: /api/users/${capture.user_id}
        assert:
          - field: status
            operator: equal
            expected: 200
          - field: body.name
            operator: equal
            expected: "Alice Smith"
          - field: body.email
            operator: equal
            expected: "alice@example.com"

  - name: "Delete returns 204"
    connector: http
    steps:
      - action: request
        method: POST
        path: /api/users
        body:
          name: "Temp User"
          email: "temp@example.com"
        capture:
          user_id: body.id

      - action: request
        method: DELETE
        path: /api/users/${capture.user_id}
        assert:
          - field: status
            operator: equal
            expected: 204

  - name: "Get non-existent user returns 404"
    connector: http
    tags: [smoke]
    steps:
      - action: request
        method: GET
        path: /api/users/999999
        assert:
          - field: status
            operator: equal
            expected: 404
```

### Step 3: Run the Tests

```bash
scrutineer run
```

Or with options:

```bash
# Run only smoke tests
scrutineer run --tags smoke

# JSON output for CI
scrutineer run --format json

# Verbose with telemetry
scrutineer run --verbose
```

## Next Steps

- [Load Testing](load-testing.md) -- run load tests against your API
- [Fuzz Testing](fuzz-testing.md) -- find edge cases with fuzz testing
- [Browser Testing](browser-testing.md) -- test web UIs
- [CI Integration](ci-integration.md) -- automate tests in your pipeline
