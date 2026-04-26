# GraphQL Support

The GraphQL package provides query execution, mutation support, subscriptions via WebSocket, and schema introspection. It is implemented as a standalone library (`connector/http/graphql/`) that operates on top of a standard `net/http` client, and can be used independently or in conjunction with the HTTP connector.

Source: `connector/http/graphql/`

## Overview

GraphQL operations in scrutineer are not invoked as a separate connector action. Instead, the `graphql` package provides Go-level functions (`Execute`, `Subscribe`, `Introspect`) that accept an `http.Client`, an endpoint URL, a `Request` struct, and optional headers. Test YAML can invoke GraphQL through the HTTP connector or through a dedicated GraphQL test step (depending on engine wiring).

## Request Structure

All GraphQL operations use the `Request` type:

| Field | JSON Key | Type | Required | Description |
|-------|----------|------|----------|-------------|
| `Query` | `query` | `string` | Yes | The GraphQL query or mutation string. |
| `Variables` | `variables` | `map[string]any` | No | Variables for the query. Omitted from the JSON payload when nil. |
| `OperationName` | `operationName` | `string` | No | The operation name when the query document contains multiple operations. Omitted from the JSON payload when empty. |

### Building Requests from YAML Parameters

The `BuildRequest` function creates a `Request` from a step parameter map. It accepts:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | `string` | Yes | The GraphQL query or mutation string. Must not be empty. |
| `variables` | `map[string]any` | No | Variables for the query. |
| `operation_name` | `string` | No | The operation name when the query contains multiple operations. |

## Response Structure

All queries and mutations return a `Response`:

| Field | Type | Description |
|-------|------|-------------|
| `Data` | `any` | The `data` field from the GraphQL response. Typically a `map[string]any` after JSON decoding. |
| `Errors` | `[]GraphQLError` | The `errors` array from the GraphQL response. Empty when no errors occurred. |

### GraphQLError Structure

| Field | Type | Description |
|-------|------|-------------|
| `Message` | `string` | Human-readable error message. |
| `Locations` | `[]Location` | Source locations in the query where the error originated. Each has `Line` (int) and `Column` (int). |
| `Path` | `[]any` | Path to the field that caused the error (strings for field names, ints for list indices). |
| `Extensions` | `map[string]any` | Optional extension data from the server. |

---

## Queries and Mutations

Queries and mutations are executed identically -- both use `Execute`, which sends an HTTP POST request with `Content-Type: application/json` and `Accept: application/json` headers.

### Execution Flow

1. The `Request` is JSON-serialized as the POST body.
2. `Content-Type: application/json` and `Accept: application/json` headers are set.
3. Additional headers from the `headers` map are applied.
4. The response body is JSON-decoded into a `Response`.

If a nil `http.Client` is passed, `http.DefaultClient` is used.

### Query Example

```yaml
steps:
  - connector: http
    action: request
    parameters:
      method: POST
      path: /graphql
      headers:
        Content-Type: application/json
      body:
        query: |
          query GetUser($id: ID!) {
            user(id: $id) {
              id
              name
              email
            }
          }
        variables:
          id: "42"
        operation_name: GetUser
```

### Standalone Query YAML (via GraphQL parameters)

```yaml
steps:
  - connector: graphql
    action: query
    parameters:
      query: |
        {
          users {
            id
            name
          }
        }
    assert:
      - path: data.users
        not_empty: true
```

### Mutation Example

```yaml
steps:
  - connector: graphql
    action: query
    parameters:
      query: |
        mutation CreateUser($input: CreateUserInput!) {
          createUser(input: $input) {
            id
            name
            createdAt
          }
        }
      variables:
        input:
          name: "Jane Doe"
          email: "jane@example.com"
      operation_name: CreateUser
    assert:
      - path: data.createUser.id
        not_empty: true
```

### Query with Variables

```yaml
steps:
  - connector: graphql
    action: query
    parameters:
      query: |
        query SearchUsers($term: String!, $limit: Int) {
          searchUsers(term: $term, limit: $limit) {
            id
            name
          }
        }
      variables:
        term: "admin"
        limit: 10
```

### Error Handling

GraphQL responses can contain both `data` and `errors`. A response is not necessarily a failure just because it contains errors -- partial data is valid in GraphQL. The `Errors` field is populated from the `errors` array in the JSON response.

```yaml
steps:
  - connector: graphql
    action: query
    parameters:
      query: |
        {
          user(id: "nonexistent") {
            id
            name
          }
        }
    assert:
      - path: errors[0].message
        contains: "not found"
      - path: data.user
        equals: null
```

---

## Introspection

The `Introspect` function sends the standard GraphQL introspection query to discover the server's schema.

### Introspection Query

The built-in introspection query (`IntrospectionQuery` constant) fetches:

- `queryType.name` -- the root query type name
- `mutationType.name` -- the root mutation type name (if present)
- `subscriptionType.name` -- the root subscription type name (if present)
- `types[]` -- all types in the schema, each with:
  - `name` -- type name
  - `kind` -- type kind (OBJECT, SCALAR, ENUM, etc.)
  - `fields(includeDeprecated: true)[]` -- fields on the type, each with:
    - `name` -- field name
    - `type.name`, `type.kind`, `type.ofType.name`, `type.ofType.kind` -- field type info

### Schema Result

The `Introspect` function returns a `Schema` struct:

| Field | Type | Description |
|-------|------|-------------|
| `QueryType` | `string` | Name of the root query type. |
| `MutationType` | `string` | Name of the root mutation type. Empty if not present. |
| `SubscriptionType` | `string` | Name of the root subscription type. Empty if not present. |
| `Types` | `[]SchemaType` | All types in the schema. |

Each `SchemaType`:

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Type name. |
| `Kind` | `string` | Type kind (`OBJECT`, `SCALAR`, `ENUM`, `INPUT_OBJECT`, `INTERFACE`, `UNION`, `LIST`, `NON_NULL`). |
| `Fields` | `[]SchemaField` | Fields on the type. |

Each `SchemaField`:

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Field name. |
| `Type` | `string` | Readable type string (e.g., `String`, `Int!`, `[User]`). Wrapper types (NON_NULL, LIST) are represented with `!` and `[]` notation. |

### Introspection Example

```yaml
steps:
  - connector: graphql
    action: introspect
    parameters:
      endpoint: https://api.example.com/graphql
    assert:
      - path: query_type
        equals: "Query"
      - path: types
        not_empty: true
```

### Introspection Errors

If the introspection query returns GraphQL errors, the first error message is reported. If the response data structure is unexpected (missing `__schema`), a descriptive error is returned.

---

## Subscriptions

Subscriptions use the **graphql-ws** WebSocket sub-protocol (`graphql-transport-ws`), as defined by the [graphql-ws specification](https://github.com/enisdenjo/graphql-ws).

### WebSocket Protocol Details

The subscription implementation builds WebSocket frames from scratch (no external library):

1. **Connection**: A raw TCP connection is opened to the endpoint. The endpoint URL supports `ws://`, `wss://`, `http://`, and `https://` schemes (`http`/`https` are used as-is; `ws`/`wss` are rewritten).

2. **WebSocket Upgrade**: An HTTP upgrade request is sent with:
   - `Upgrade: websocket`
   - `Connection: Upgrade`
   - `Sec-WebSocket-Key: <random base64>`
   - `Sec-WebSocket-Version: 13`
   - `Sec-WebSocket-Protocol: graphql-transport-ws`
   - Any additional headers from the `headers` map

3. **Handshake**: The server must respond with HTTP 101 Switching Protocols.

4. **Connection Init**: A `connection_init` message is sent.

5. **Connection Ack**: The client waits for a `connection_ack` message. Any other message type is an error.

6. **Subscribe**: A `subscribe` message is sent with:
   - `id`: `"1"` (fixed subscription ID)
   - `type`: `"subscribe"`
   - `payload`: the JSON-serialized GraphQL `Request`

### WebSocket Frame Handling

- **Client-to-server frames** are masked per RFC 6455 (using a random 4-byte mask from `crypto/rand`).
- **Server-to-client frames** are expected to be unmasked.
- **Supported opcodes**: text (0x1), close (0x8), ping (0x9), pong (0xA).
- **Ping frames** receive an automatic pong response.
- **Payload lengths** support all three RFC 6455 size formats: 7-bit (up to 125 bytes), 16-bit extended, and 64-bit extended.

### Receiving Subscription Events

The `Next` method blocks until the next event arrives. It processes messages as follows:

| Message Type | Behavior |
|-------------|----------|
| `next` | Payload is decoded as a `Response` and returned. |
| `error` | Payload is decoded as `[]GraphQLError` and returned in a `Response` with nil `Data`. |
| `complete` | Returns an error indicating the subscription was completed by the server. |
| Other types | Skipped (e.g., keepalive/ping messages). |

### Closing a Subscription

The `Close` method:

1. Sends a `complete` message with the subscription ID (best-effort).
2. Sends a WebSocket close frame with code 1000 ("normal closure").
3. Closes the TCP connection.

Close is idempotent -- calling it multiple times is safe.

### Subscription Example

```yaml
steps:
  - connector: graphql
    action: subscribe
    parameters:
      endpoint: ws://localhost:4000/graphql
      query: |
        subscription OnNewMessage($channel: String!) {
          newMessage(channel: $channel) {
            id
            text
            author
            timestamp
          }
        }
      variables:
        channel: "general"
    assert:
      - path: data.newMessage.text
        not_empty: true
```

### Subscription with Headers

```yaml
steps:
  - connector: graphql
    action: subscribe
    parameters:
      endpoint: wss://api.example.com/graphql
      headers:
        Authorization: "Bearer eyJhbG..."
      query: |
        subscription {
          orderUpdated {
            id
            status
          }
        }
```

### Context Cancellation

When the context is cancelled during `Next`, the TCP connection is force-closed to unblock the read goroutine, and `ctx.Err()` is returned.

---

## Returned Data Summary

### Queries and Mutations (via `Execute`)

The `Response` struct contains:

| Key | Type | Description |
|-----|------|-------------|
| `Data` | `any` | Parsed `data` field (typically `map[string]any`). |
| `Errors` | `[]GraphQLError` | Parsed `errors` array. Empty list when no errors. |

### Subscriptions (via `Next`)

Each call to `Next` returns a `Response` with the same structure as queries. For `error` message types, `Data` is nil and `Errors` is populated.

### Introspection (via `Introspect`)

Returns a `Schema` struct as described above, not a raw `Response`.
