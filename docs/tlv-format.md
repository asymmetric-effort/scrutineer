# TLV Binary Log Format

Scrutineer records test telemetry in a custom Type-Length-Value (TLV) binary format with nanosecond timestamps. This format is designed for fast writing during test execution and compact storage.

## File Header (16 bytes)

| Offset | Size | Type | Description |
|--------|------|------|-------------|
| 0 | 4 | bytes | Magic bytes: `SCTL` |
| 4 | 2 | uint16 LE | Format version (currently 1) |
| 6 | 2 | uint16 LE | Flags (reserved, currently 0) |
| 8 | 8 | int64 LE | Creation timestamp (nanoseconds since epoch) |

## Record Format (variable length)

Each record has a 20-byte fixed header followed by variable-length tags and detail blob.

### Record Header (20 bytes)

| Offset | Size | Type | Description |
|--------|------|------|-------------|
| 0 | 8 | int64 LE | Timestamp (nanoseconds since epoch) |
| 8 | 2 | uint16 LE | Event type |
| 10 | 2 | uint16 LE | Tag count |
| 12 | 4 | uint32 LE | Tags section length (bytes) |
| 16 | 4 | uint32 LE | Detail blob length (bytes) |

### Tags Section

Repeated `tag_count` times:

| Size | Type | Description |
|------|------|-------------|
| 2 | uint16 LE | Key length |
| variable | bytes | Key (UTF-8) |
| 2 | uint16 LE | Value length |
| variable | bytes | Value (UTF-8) |

### Detail Blob

Raw bytes of length `detail_blob_length`. Content is event-specific (JSON, error messages, request/response bodies, stack traces).

## Event Types

| Value | Name | Description |
|-------|------|-------------|
| 0x01 | SuiteStart | Test suite begins |
| 0x02 | SuiteEnd | Test suite ends |
| 0x03 | TestStart | Individual test begins |
| 0x04 | TestPass | Test passed |
| 0x05 | TestFail | Test failed |
| 0x06 | TestSkip | Test skipped |
| 0x07 | StepStart | Test step begins |
| 0x08 | StepEnd | Test step ends |
| 0x09 | Assertion | Assertion evaluated |
| 0x0A | Request | Outbound request sent |
| 0x0B | Response | Response received |
| 0x0C | Error | Error occurred |
| 0x0D | ConnectorSetup | Connector initialized |
| 0x0E | ConnectorTeardown | Connector torn down |
| 0x0F | Metric | Performance metric recorded |

## Reading Logs

Use the built-in log-dump tool:

```bash
scrutineer log-dump scrutineer.log
```

Output format:
```
[2026-04-26T10:30:00.123456789Z] TestStart suite=API test=CreateUser
[2026-04-26T10:30:00.234567890Z] Request url=https://api.example.com/users method=POST
[2026-04-26T10:30:00.345678901Z] Response status=201
[2026-04-26T10:30:00.345679000Z] Assertion passed=true
[2026-04-26T10:30:00.345680000Z] TestPass suite=API test=CreateUser
```

## Byte Order

All multi-byte integers are little-endian.

## Third-Party Integration

The format is simple enough for third-party tools to parse. A minimal reader needs only:
1. Validate the 4-byte magic header
2. Read 20-byte record headers in a loop
3. Skip or parse tags/detail based on lengths
