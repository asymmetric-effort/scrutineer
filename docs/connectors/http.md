# HTTP Connector

The HTTP connector sends HTTP requests, captures responses, and supports TLS configuration and multiple authentication methods. It is identified as `http` in YAML test definitions.

Source: `connector/http/`

## Setup Configuration

The `Setup` method accepts the following configuration keys:

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `base_url` | `string` | No | Base URL prepended to the `path` in every request. For example, `https://api.example.com`. |
| `default_headers` | `map[string]string` | No | Headers applied to every request. Step-specific headers override these. Also accepts `map[string]any` where values are converted via `fmt.Sprintf("%v", val)`. |
| `timeout` | `string` | No | HTTP client timeout as a Go duration string (e.g., `"30s"`, `"5m"`, `"500ms"`). Parsed with `time.ParseDuration`. This is the overall client-level timeout, not a per-step timeout. |
| `tls_skip_verify` | `bool` | No | When `true`, disables TLS certificate verification. Use for self-signed certificates. Default: `false`. |
| `tls_ca_file` | `string` | No | Path to a PEM-encoded CA certificate file. The CA is added to the trust pool for verifying server certificates. |
| `tls_cert_file` | `string` | No | Path to a PEM-encoded client certificate for mutual TLS (mTLS). Must be provided together with `tls_key_file`. |
| `tls_key_file` | `string` | No | Path to a PEM-encoded client private key for mTLS. Must be provided together with `tls_cert_file`. |

### TLS Configuration

The connector enforces **TLS 1.2 as the minimum version** and supports TLS 1.3 (handled automatically by Go's `crypto/tls`). HTTP/2 is enabled automatically by Go's `net/http` when TLS is used.

**Self-signed certificates**: Set `tls_skip_verify: true` to skip server certificate verification.

**Custom CA**: Provide `tls_ca_file` pointing to a PEM certificate to add a custom certificate authority to the trust pool.

**Mutual TLS (mTLS)**: Provide both `tls_cert_file` and `tls_key_file`. These are loaded as a `tls.Certificate` via `tls.LoadX509KeyPair`. Providing only one of the two is an error.

### Setup Examples

#### Basic setup

```yaml
connector: http
config:
  base_url: https://api.example.com
  timeout: "30s"
  default_headers:
    Content-Type: application/json
    Accept: application/json
```

#### Self-signed certificate

```yaml
connector: http
config:
  base_url: https://localhost:8443
  tls_skip_verify: true
```

#### Custom CA certificate

```yaml
connector: http
config:
  base_url: https://internal.corp.example.com
  tls_ca_file: /etc/certs/internal-ca.pem
```

#### Mutual TLS (mTLS)

```yaml
connector: http
config:
  base_url: https://secure.example.com
  tls_cert_file: /etc/certs/client.pem
  tls_key_file: /etc/certs/client-key.pem
  tls_ca_file: /etc/certs/ca.pem
```

## Actions

The HTTP connector supports one action: `request`.

---

## Action: `request`

Sends an HTTP request and captures the response.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `method` | `string` | Yes | HTTP method (e.g., `GET`, `POST`, `PUT`, `DELETE`, `PATCH`). Automatically uppercased. |
| `path` | `string` | Yes | URL path appended to `base_url`. Can include a full URL if `base_url` is not set. |
| `headers` | `map[string]any` | No | Request headers. Override `default_headers`. Values are converted to strings via `fmt.Sprintf("%v", val)`. |
| `query` | `map[string]any` | No | URL query parameters. Each key-value pair is added to the URL query string. Values are converted to strings via `fmt.Sprintf("%v", val)`. |
| `body` | `string` or `map[string]any` | No | Request body. If a string, sent as-is. If a map (or any other non-string type), JSON-serialized with `json.Marshal`. |
| `auth` | `map` | No | Authentication configuration. See the Authentication section below. |

### Step Timeout

If `step.Timeout` is set, a context deadline is applied to the individual request. This is separate from the client-level `timeout` in setup configuration.

### Authentication

The `auth` parameter supports three authentication types, each with its own set of parameters:

#### Bearer Token

```yaml
auth:
  type: bearer
  token: "eyJhbGciOiJIUzI1NiIs..."
```

Sets the header: `Authorization: Bearer <token>`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | `string` | Yes | Must be `"bearer"`. |
| `token` | `string` | Yes | The bearer token value. |

#### Basic Authentication

```yaml
auth:
  type: basic
  username: admin
  password: secret123
```

Sets the header: `Authorization: Basic <base64(username:password)>`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | `string` | Yes | Must be `"basic"`. |
| `username` | `string` | Yes | The username. |
| `password` | `string` | Yes | The password. |

The username and password are concatenated with `:` and base64-encoded using `base64.StdEncoding`.

#### API Key

```yaml
auth:
  type: api_key
  header: X-API-Key
  key: "abc123def456"
```

Sets a custom header to the key value: `<header>: <key>`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | `string` | Yes | Must be `"api_key"`. |
| `header` | `string` | Yes | The header name to set (e.g., `X-API-Key`). |
| `key` | `string` | Yes | The API key value. |

### Result Data Keys

| Key | Type | Description |
|-----|------|-------------|
| `status` | `int` | HTTP status code (e.g., `200`, `404`, `500`). |
| `status_text` | `string` | Full HTTP status line (e.g., `"200 OK"`, `"404 Not Found"`). |
| `headers` | `map[string][]string` | Response headers. Each header name maps to a list of values (since HTTP headers can have multiple values). |
| `body` | `any` | Response body. If the body is valid JSON, it is parsed into a Go structure (map, list, string, number, etc.). Otherwise, it is returned as a raw string. |
| `body_raw` | `string` | Response body as a raw string, regardless of whether JSON parsing succeeded. |
| `elapsed_ms` | `float64` | Time taken for the request in milliseconds. |

Result metadata (`Meta`):
- `url`: the full URL that was requested
- `method`: the HTTP method used (uppercased)

### Request Building

1. The full URL is constructed as `base_url + path`.
2. If `query` is provided, query parameters are appended to the URL.
3. Default headers from setup are applied first.
4. Step-specific `headers` override any matching default headers.
5. Authentication headers are applied last via `applyAuth`.
6. The body is read and the request is executed using the configured `http.Client`.

### Response Parsing

The response body is always captured as a raw string (`body_raw`). Additionally, the connector attempts to parse the body as JSON. If parsing succeeds, the structured result is stored in `body`; otherwise, `body` contains the same raw string.

### Examples

#### GET request

```yaml
steps:
  - connector: http
    action: request
    parameters:
      method: GET
      path: /api/v1/users
    assert:
      - path: status
        equals: 200
```

#### GET with query parameters

```yaml
steps:
  - connector: http
    action: request
    parameters:
      method: GET
      path: /api/v1/users
      query:
        page: 1
        limit: 10
        active: true
    assert:
      - path: status
        equals: 200
```

#### POST with JSON body

```yaml
steps:
  - connector: http
    action: request
    parameters:
      method: POST
      path: /api/v1/users
      headers:
        Content-Type: application/json
      body:
        name: "Jane Doe"
        email: "jane@example.com"
        role: admin
    assert:
      - path: status
        equals: 201
      - path: body.id
        not_empty: true
```

#### PUT request

```yaml
steps:
  - connector: http
    action: request
    parameters:
      method: PUT
      path: /api/v1/users/42
      body:
        name: "Jane Smith"
        email: "jane.smith@example.com"
    assert:
      - path: status
        equals: 200
```

#### DELETE request

```yaml
steps:
  - connector: http
    action: request
    parameters:
      method: DELETE
      path: /api/v1/users/42
    assert:
      - path: status
        equals: 204
```

#### Bearer token authentication

```yaml
steps:
  - connector: http
    action: request
    parameters:
      method: GET
      path: /api/v1/profile
      auth:
        type: bearer
        token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    assert:
      - path: status
        equals: 200
```

#### Basic authentication

```yaml
steps:
  - connector: http
    action: request
    parameters:
      method: GET
      path: /api/v1/admin/dashboard
      auth:
        type: basic
        username: admin
        password: supersecret
    assert:
      - path: status
        equals: 200
```

#### API key authentication

```yaml
steps:
  - connector: http
    action: request
    parameters:
      method: GET
      path: /api/v1/data
      auth:
        type: api_key
        header: X-API-Key
        key: "abc123def456"
    assert:
      - path: status
        equals: 200
```

#### POST with string body

```yaml
steps:
  - connector: http
    action: request
    parameters:
      method: POST
      path: /api/v1/webhook
      headers:
        Content-Type: text/plain
      body: "raw text payload"
    assert:
      - path: status
        equals: 200
```

#### Request with step-level timeout

```yaml
steps:
  - connector: http
    action: request
    timeout: 5s
    parameters:
      method: GET
      path: /api/v1/slow-endpoint
    assert:
      - path: status
        equals: 200
```

#### Asserting response headers

```yaml
steps:
  - connector: http
    action: request
    parameters:
      method: GET
      path: /api/v1/resource
    assert:
      - path: headers.Content-Type[0]
        contains: "application/json"
```

#### Using body_raw for non-JSON responses

```yaml
steps:
  - connector: http
    action: request
    parameters:
      method: GET
      path: /health
    assert:
      - path: body_raw
        equals: "OK"
```

## Teardown

The `Teardown` method calls `client.CloseIdleConnections()` to release any idle HTTP keep-alive connections held by the transport.
