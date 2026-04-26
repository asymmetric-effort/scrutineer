# Telemetry Reference

Scrutineer records all test events to a structured binary log using a custom Type-Length-Value (TLV) format with nanosecond-precision timestamps. The format is defined in `core/telemetry/` and is designed for fast sequential writing during test execution and compact storage.

---

## Enabling Telemetry

Telemetry is enabled by default. Configure it in `scrutineer.yaml`:

```yaml
telemetry:
  enabled: true
  output: "scrutineer.log"
```

Or enable via CLI:

```bash
scrutineer run --verbose
```

The `--verbose` flag sets `telemetry.enabled` to `true`.

---

## File Structure

A telemetry log file consists of a 16-byte file header followed by zero or more variable-length records.

```
[FileHeader: 16 bytes][Record 0][Record 1][Record 2]...
```

All multi-byte integers use **little-endian** byte order throughout the format.

---

## File Header (16 bytes)

The file header is written once at the beginning of the log by `WriteHeader()` in `core/telemetry/tlv.go`.

| Offset | Size | Type | Field | Description |
|--------|------|------|-------|-------------|
| 0 | 4 | `[4]byte` | Magic | Magic bytes: `SCTL` (0x53 0x43 0x54 0x4C) |
| 4 | 2 | `uint16 LE` | Version | Format version. Currently `1`. |
| 6 | 2 | `uint16 LE` | Flags | Reserved flags. Currently `0`. |
| 8 | 8 | `int64 LE` | CreatedAt | File creation timestamp (nanoseconds since Unix epoch). |

### Header Hex Example

A file created at timestamp `1745664600123456789` (2025-04-26T10:30:00.123456789Z):

```
Offset  Bytes
0x00    53 43 54 4C              SCTL magic
0x04    01 00                    version 1
0x06    00 00                    flags 0
0x08    15 3D 08 9A 39 38 3B 18  created_at (int64 LE)
```

### Header Errors

| Error | Condition |
|-------|-----------|
| `ErrInvalidMagic` | First 4 bytes are not `SCTL`. |
| `ErrUnsupportedVersion` | Version field is not `1`. |
| `ErrTruncatedHeader` | Fewer than 16 bytes available. |

---

## Record Format

Each record has a 20-byte fixed header followed by a variable-length tags section and a variable-length detail blob. Records are written by `EncodeRecord()` and read by `DecodeRecord()`.

### Record Header (20 bytes)

| Offset | Size | Type | Field | Description |
|--------|------|------|-------|-------------|
| 0 | 8 | `int64 LE` | Timestamp | Event timestamp (nanoseconds since Unix epoch). |
| 8 | 2 | `uint16 LE` | EventType | Event type code (see table below). |
| 10 | 2 | `uint16 LE` | TagCount | Number of key-value tag pairs. |
| 12 | 4 | `uint32 LE` | TagsLen | Total byte length of the tags section. |
| 16 | 4 | `uint32 LE` | DetailLen | Byte length of the detail blob. |

### Tags Section

Immediately follows the record header. Contains `TagCount` key-value pairs serialized sequentially. Keys are sorted alphabetically for deterministic output.

Each tag:

| Size | Type | Description |
|------|------|-------------|
| 2 | `uint16 LE` | Key length in bytes |
| variable | `bytes` | Key string (UTF-8) |
| 2 | `uint16 LE` | Value length in bytes |
| variable | `bytes` | Value string (UTF-8) |

Total tags section size equals the sum of `(2 + key_len + 2 + value_len)` for each tag, and must equal the `TagsLen` field in the record header.

### Detail Blob

Immediately follows the tags section. Raw bytes of length `DetailLen`. Content is event-type-specific (see below).

### Record Hex Example

A `TestStart` event (0x03) at timestamp `1745664600234567890` with tags `suite=API`, `test=CreateUser` and no detail blob:

```
Offset  Bytes                     Description
0x00    D2 40 08 9A 39 38 3B 18   timestamp (int64 LE)
0x08    03 00                     event type 0x0003 (TestStart)
0x0A    02 00                     tag count = 2
0x0C    1E 00 00 00               tags length = 30 bytes
0x10    00 00 00 00               detail length = 0
-- tags section (30 bytes) --
0x14    05 00                     key length = 5
0x16    73 75 69 74 65            "suite"
0x1B    03 00                     value length = 3
0x1D    41 50 49                  "API"
0x20    04 00                     key length = 4
0x22    74 65 73 74              "test"
0x26    0A 00                     value length = 10
0x28    43 72 65 61 74 65 55 73   "CreateUs"
0x30    65 72                     "er"
```

### Record Errors

| Error | Condition |
|-------|-----------|
| `io.EOF` | Clean end of file (no more records). |
| `ErrTruncatedRecord` | Record header or body is incomplete. |

---

## Event Types

Event types are `uint16` constants defined in `core/telemetry/events.go`.

| Value | Constant | Name | Description |
|-------|----------|------|-------------|
| `0x01` | `SuiteStart` | SuiteStart | A test suite begins execution. |
| `0x02` | `SuiteEnd` | SuiteEnd | A test suite finishes execution. |
| `0x03` | `TestStart` | TestStart | An individual test case begins. |
| `0x04` | `TestPass` | TestPass | A test case passed all assertions. |
| `0x05` | `TestFail` | TestFail | A test case had one or more assertion failures. |
| `0x06` | `TestSkip` | TestSkip | A test case was skipped. |
| `0x07` | `StepStart` | StepStart | A test step begins execution. |
| `0x08` | `StepEnd` | StepEnd | A test step finishes execution. |
| `0x09` | `Assertion` | Assertion | An assertion was evaluated. |
| `0x0A` | `Request` | Request | An outbound request was sent (HTTP, gRPC, SSH, etc.). |
| `0x0B` | `Response` | Response | A response was received. |
| `0x0C` | `Error` | Error | An error occurred during execution. |
| `0x0D` | `ConnectorSetup` | ConnectorSetup | A connector was initialized. |
| `0x0E` | `ConnectorTeardown` | ConnectorTeardown | A connector was torn down. |
| `0x0F` | `Metric` | Metric | A performance or timing metric was recorded. |

Unknown event type values produce the string `Unknown(0xNN)` via the `EventType.String()` method.

---

## Tag Conventions

Tags are key-value string pairs attached to records for filtering and grouping. Common tag keys:

| Tag Key | Used With | Description |
|---------|-----------|-------------|
| `suite` | SuiteStart, SuiteEnd, TestStart, TestPass, TestFail, TestSkip | Suite name. |
| `test` | TestStart, TestPass, TestFail, TestSkip | Test case name. |
| `step` | StepStart, StepEnd | Step description or index. |
| `connector` | ConnectorSetup, ConnectorTeardown, Request, Response | Connector type (e.g., `http`, `cli`). |
| `assertion` | Assertion | Assertion operator name. |
| `passed` | Assertion | `"true"` or `"false"`. |
| `url` | Request, Response | Request URL (HTTP connector). |
| `method` | Request | HTTP method (GET, POST, etc.). |
| `status` | Response | HTTP status code as string. |
| `error` | Error | Short error description. |
| `metric` | Metric | Metric name (e.g., `response_time`). |

Tags are sorted by key when encoded, ensuring deterministic binary output and reproducible log files.

---

## Detail Blob Formats by Event Type

The detail blob is a variable-length byte payload. Its interpretation depends on the event type.

### SuiteStart / SuiteEnd

- Typically empty or JSON with suite metadata.
- Example: `{"suite":"API Tests","tests":5}`

### TestStart

- Typically empty or JSON with test metadata.
- Example: `{"name":"CreateUser","tags":["smoke","api"]}`

### TestPass

- Typically empty or JSON with timing data.
- Example: `{"duration_ns":123456789}`

### TestFail

- JSON with failure details: assertion errors, expected/actual values.
- Example: `{"assertion":"equal","expected":200,"actual":500,"message":"expected status 200, got 500"}`

### TestSkip

- Typically empty or JSON with skip reason.
- Example: `{"reason":"skip flag set"}`

### Assertion

- JSON with assertion details.
- Example: `{"operator":"status_code","expected":200,"actual":200,"passed":true}`

### Request

- JSON with request details: URL, method, headers, body (possibly truncated).
- Example: `{"method":"POST","url":"https://api.example.com/users","content_type":"application/json"}`

### Response

- JSON with response details: status code, headers, body (possibly truncated), timing.
- Example: `{"status":201,"content_type":"application/json","duration_ns":45678901}`

### Error

- Plain text error message or JSON with structured error info.
- Example: `connection refused: dial tcp 127.0.0.1:8080`

### ConnectorSetup / ConnectorTeardown

- JSON with connector configuration used.
- Example: `{"type":"http","base_url":"https://api.example.com"}`

### Metric

- JSON with metric name and value.
- Example: `{"name":"response_time","value_ns":45678901,"unit":"ns"}`

---

## Interfaces

### RecordWriter

Defined in `core/telemetry/telemetry.go`:

```go
type RecordWriter interface {
    Write(Record) error
    Close() error
}
```

The `Writer` implementation (`core/telemetry/writer.go`) writes the file header automatically on the first `Write()` call. Calling `Write()` after `Close()` returns `io.ErrClosedPipe`. If the underlying `io.Writer` implements `io.Closer`, it is closed when `Close()` is called.

### RecordReader

```go
type RecordReader interface {
    Next() (Record, error)
    Close() error
}
```

The `Reader` implementation (`core/telemetry/reader.go`) reads and validates the file header automatically on the first `Next()` call. Returns `io.EOF` when no more records are available.

### Record

```go
type Record struct {
    Timestamp int64
    EventType EventType
    Tags      map[string]string
    Detail    []byte
}
```

---

## Reading Logs

Use `scrutineer log-dump` to convert binary logs to human-readable text:

```bash
scrutineer log-dump scrutineer.log
```

Output format (one line per record):

```
[2026-04-26T10:30:00.123456789Z] TestStart suite=API test=CreateUser
[2026-04-26T10:30:00.234567890Z] Request url=https://api.example.com/users method=POST
[2026-04-26T10:30:00.345678901Z] Response status=201
[2026-04-26T10:30:00.345679000Z] Assertion passed=true
[2026-04-26T10:30:00.345680000Z] TestPass suite=API test=CreateUser
```

Timestamps are formatted as RFC3339Nano in UTC. Tags are printed as `key=value` pairs. The detail blob is appended as-is if present.

---

## Third-Party Integration

The TLV format is intentionally simple for third-party tool interoperability. A minimal reader in any language needs to:

1. Read and validate the 16-byte file header (check `SCTL` magic and version `1`).
2. In a loop, read the 20-byte record header.
3. Read `TagsLen` bytes and decode tags as repeated `(uint16 key_len, key_bytes, uint16 val_len, val_bytes)`.
4. Read `DetailLen` bytes for the detail blob.
5. Stop on EOF.

All integers are little-endian. No compression. No framing beyond the record headers. No checksums (the format prioritizes write speed).
