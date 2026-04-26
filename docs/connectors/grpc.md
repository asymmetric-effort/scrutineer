# gRPC Connector

The gRPC connector supports all four RPC modes (unary, server streaming, client streaming, and bidirectional streaming) using dynamic message construction via protocol buffer reflection. No compiled protobuf stubs are required. It is identified as `grpc` in YAML test definitions.

Source: `connector/grpc/`

Dependencies: `google.golang.org/grpc`, `google.golang.org/protobuf`

## Setup Configuration

The `Setup` method establishes a gRPC client connection to the specified endpoint.

| Key | Type | Required | Default | Description |
|-----|------|----------|---------|-------------|
| `endpoint` | `string` | Yes | -- | The gRPC server address (e.g., `"localhost:50051"`). |
| `tls` | `bool` | No | `false` | Enable TLS transport. When `false` (and `plaintext` is not set), insecure credentials are used. |
| `plaintext` | `bool` | No | `false` | Explicitly use plaintext (insecure) transport. Takes precedence over `tls`. |
| `tls_skip_verify` | `bool` | No | `false` | Skip TLS certificate verification (for self-signed certificates). |
| `tls_ca_file` | `string` | No | -- | Path to a PEM-encoded CA certificate file for custom certificate authorities. |
| `schema` | `string` | No | -- | Path to a `.proto` file (reserved for future use). Currently, the connector uses gRPC server reflection exclusively. |

### Transport Configuration

The transport is determined as follows:

- If `plaintext` is `true` OR `tls` is `false`: insecure credentials (`insecure.NewCredentials()`).
- If `tls` is `true` and `plaintext` is `false`: TLS credentials are built with:
  - Minimum TLS version: **TLS 1.2**
  - Optional `InsecureSkipVerify` via `tls_skip_verify`
  - Optional custom CA pool via `tls_ca_file`

### Setup Examples

#### Plaintext connection

```yaml
connector: grpc
config:
  endpoint: localhost:50051
  plaintext: true
```

#### TLS connection

```yaml
connector: grpc
config:
  endpoint: api.example.com:443
  tls: true
```

#### TLS with self-signed certificate

```yaml
connector: grpc
config:
  endpoint: localhost:50051
  tls: true
  tls_skip_verify: true
```

#### TLS with custom CA

```yaml
connector: grpc
config:
  endpoint: internal-grpc.corp:443
  tls: true
  tls_ca_file: /etc/certs/internal-ca.pem
```

## Service Discovery: gRPC Server Reflection

The connector uses **gRPC server reflection** (v1) to discover services and methods at runtime. When an action is executed:

1. A `ServerReflection.ServerReflectionInfo` stream is opened.
2. A `FileContainingSymbol` request is sent for the service name.
3. The returned file descriptors are parsed.
4. Transitive dependencies are resolved iteratively by sending `FileByFilename` requests for any missing imports.
5. A protobuf file descriptor registry is built from all collected descriptors.
6. The target method's input and output message descriptors are resolved.

The reflection client also supports `listServices` to enumerate all available services.

### Method Resolution

Methods are resolved by matching both the fully qualified service name (e.g., `mypackage.MyService`) and the simple service name (e.g., `MyService`). The method name must match exactly.

### Dynamic Message Construction

Request messages are built dynamically from YAML maps using `dynamicpb.Message`. Field mapping supports:

- **Field name matching**: by protobuf field name, with fallback to JSON name.
- **Scalar types**: bool, int32, int64, uint32, uint64, float, double, string, bytes, enum.
- **Enum values**: by numeric value (int/float64) or by name (string).
- **Nested messages**: represented as nested maps.
- **Repeated fields**: represented as lists (`[]any`).
- **Map fields**: represented as `map[string]any`.

Type coercion handles common YAML/JSON type mismatches (e.g., `float64` from JSON parsed as `int32` for protobuf).

## Actions

The gRPC connector supports four actions corresponding to the four RPC modes.

---

## Action: `unary`

A single request/response RPC.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `service` | `string` | Yes | Fully qualified service name (e.g., `mypackage.Greeter`). Simple name also accepted. |
| `method` | `string` | Yes | Method name (e.g., `SayHello`). |
| `message` | `map[string]any` | No | Request message fields. If omitted, an empty message is sent. |
| `metadata` | `map[string]any` | No | Outgoing gRPC metadata (headers). Values can be strings or lists of strings. |

### Result Data Keys

| Key | Type | Description |
|-----|------|-------------|
| `response` | `map[string]any` | The response message fields. Only present when the RPC succeeds (status OK). |
| `status_code` | `int` | Numeric gRPC status code (e.g., `0` for OK, `5` for NOT_FOUND). |
| `status_message` | `string` | Status message from the server. Empty on success. |
| `status_name` | `string` | Human-readable status code name (e.g., `"OK"`, `"NOT_FOUND"`). |
| `metadata` | `map[string]any` | Response headers (gRPC metadata). Single values are strings; multi-value keys are `[]string`. |
| `trailers` | `map[string]any` | Response trailers. Same format as `metadata`. |

### Example

```yaml
steps:
  - connector: grpc
    action: unary
    parameters:
      service: helloworld.Greeter
      method: SayHello
      message:
        name: "World"
      metadata:
        authorization: "Bearer token123"
    assert:
      - path: status_code
        equals: 0
      - path: status_name
        equals: "OK"
      - path: response.message
        equals: "Hello, World!"
```

---

## Action: `server_stream`

A server-streaming RPC where the client sends a single request and receives a stream of responses.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `service` | `string` | Yes | Service name. |
| `method` | `string` | Yes | Method name. |
| `message` | `map[string]any` | No | Single request message. |
| `metadata` | `map[string]any` | No | Outgoing gRPC metadata. |

### Result Data Keys

| Key | Type | Description |
|-----|------|-------------|
| `responses` | `[]any` | List of response message maps received from the stream. |
| `status_code` | `int` | gRPC status code. |
| `status_message` | `string` | Status message. |
| `status_name` | `string` | Status code name. |
| `metadata` | `map[string]any` | Response headers. |
| `trailers` | `map[string]any` | Response trailers. |

The stream is read until EOF (the server closes its end). All received messages are collected into the `responses` list.

### Example

```yaml
steps:
  - connector: grpc
    action: server_stream
    parameters:
      service: mypackage.DataService
      method: StreamRecords
      message:
        query: "status=active"
        limit: 100
    assert:
      - path: status_code
        equals: 0
      - path: responses
        not_empty: true
```

---

## Action: `client_stream`

A client-streaming RPC where the client sends a stream of messages and receives a single response.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `service` | `string` | Yes | Service name. |
| `method` | `string` | Yes | Method name. |
| `messages` | `[]map[string]any` | Yes | List of request messages to send. Each entry is a map representing one message. |
| `metadata` | `map[string]any` | No | Outgoing gRPC metadata. |

### Result Data Keys

| Key | Type | Description |
|-----|------|-------------|
| `response` | `map[string]any` | The single response message. Only present on success. |
| `status_code` | `int` | gRPC status code. |
| `status_message` | `string` | Status message. |
| `status_name` | `string` | Status code name. |
| `metadata` | `map[string]any` | Response headers. |
| `trailers` | `map[string]any` | Response trailers. |

All messages are sent sequentially, then the send side is closed (`CloseSend`). The single response is then received.

### Example

```yaml
steps:
  - connector: grpc
    action: client_stream
    parameters:
      service: mypackage.UploadService
      method: UploadChunks
      messages:
        - chunk_data: "part1..."
          sequence: 1
        - chunk_data: "part2..."
          sequence: 2
        - chunk_data: "part3..."
          sequence: 3
      metadata:
        x-upload-id: "abc-123"
    assert:
      - path: status_code
        equals: 0
      - path: response.bytes_received
        greater_than: 0
```

---

## Action: `bidi_stream`

A bidirectional streaming RPC where both client and server send streams of messages.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `service` | `string` | Yes | Service name. |
| `method` | `string` | Yes | Method name. |
| `messages` | `[]map[string]any` | Yes | List of request messages to send. |
| `metadata` | `map[string]any` | No | Outgoing gRPC metadata. |

### Result Data Keys

| Key | Type | Description |
|-----|------|-------------|
| `responses` | `[]any` | List of all response messages received. |
| `status_code` | `int` | gRPC status code. |
| `status_message` | `string` | Status message. |
| `status_name` | `string` | Status code name. |
| `metadata` | `map[string]any` | Response headers. |
| `trailers` | `map[string]any` | Response trailers. |

All client messages are sent first, then the send side is closed. All server responses are then collected until EOF.

### Example

```yaml
steps:
  - connector: grpc
    action: bidi_stream
    parameters:
      service: mypackage.ChatService
      method: Chat
      messages:
        - text: "Hello"
          user: "alice"
        - text: "How are you?"
          user: "alice"
      metadata:
        x-session-id: "sess-456"
    assert:
      - path: status_code
        equals: 0
      - path: responses
        not_empty: true
```

---

## Metadata Handling

### Outgoing Metadata

The `metadata` parameter in any action is converted to gRPC outgoing metadata:

- **String values**: appended as a single metadata value for the key.
- **List values** (`[]any`): each element is converted to a string and appended.
- **Other types**: converted to string via `fmt.Sprintf("%v", val)`.

```yaml
metadata:
  authorization: "Bearer token"
  x-request-id: "req-123"
  x-tags:
    - "tag1"
    - "tag2"
```

### Response Metadata and Trailers

Response headers and trailers are converted to `map[string]any`:

- **Single-value keys**: stored as a plain `string`.
- **Multi-value keys**: stored as `[]string`.

## gRPC Status Codes

The connector maps all standard gRPC status codes:

| Code | Name | Numeric Value |
|------|------|---------------|
| OK | `OK` | 0 |
| Canceled | `CANCELLED` | 1 |
| Unknown | `UNKNOWN` | 2 |
| InvalidArgument | `INVALID_ARGUMENT` | 3 |
| DeadlineExceeded | `DEADLINE_EXCEEDED` | 4 |
| NotFound | `NOT_FOUND` | 5 |
| AlreadyExists | `ALREADY_EXISTS` | 6 |
| PermissionDenied | `PERMISSION_DENIED` | 7 |
| ResourceExhausted | `RESOURCE_EXHAUSTED` | 8 |
| FailedPrecondition | `FAILED_PRECONDITION` | 9 |
| Aborted | `ABORTED` | 10 |
| OutOfRange | `OUT_OF_RANGE` | 11 |
| Unimplemented | `UNIMPLEMENTED` | 12 |
| Internal | `INTERNAL` | 13 |
| Unavailable | `UNAVAILABLE` | 14 |
| DataLoss | `DATA_LOSS` | 15 |
| Unauthenticated | `UNAUTHENTICATED` | 16 |

When an error is not a gRPC status error, the code defaults to `UNKNOWN` and the message is taken from `err.Error()`.

## Teardown

The `Teardown` method closes the gRPC client connection (`conn.Close()`).
