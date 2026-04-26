# Exit Code Reference

Scrutineer uses numeric exit codes to communicate test results and error conditions to the calling process. These are defined as constants in `core/exitcode/exitcode.go`.

Exit codes are designed for reliable use in CI/CD pipelines and shell scripts.

---

## Exit Codes

### 0 -- OK

**Constant:** `exitcode.OK`

All tests passed and all post-run checks (such as coverage gates) succeeded.

**When it occurs:**
- Every test assertion in every suite passed.
- The coverage gate threshold was met (or coverage gating is disabled).
- No connection errors, parse errors, or internal errors occurred.

**CI usage:**
```bash
scrutineer run
if [ $? -eq 0 ]; then
  echo "All tests passed"
fi
```

---

### 1 -- Test Failure

**Constant:** `exitcode.TestFailure`

One or more test assertions failed, or the coverage gate was not met.

**When it occurs:**
- At least one assertion in a test step evaluated to failure.
- All tests passed, but measured coverage was below the configured `coverage.threshold`. In this case, a coverage error is printed to stderr.

**CI usage:**
```bash
scrutineer run
if [ $? -eq 1 ]; then
  echo "Tests failed -- check output for details"
  exit 1
fi
```

---

### 2 -- Connection Error

**Constant:** `exitcode.ConnectionError`

A connection or network error prevented tests from reaching the target system.

**When it occurs:**
- An HTTP connector could not establish a TCP connection.
- An SSH connector failed to authenticate or connect.
- A gRPC dial failed.
- DNS resolution failed.
- TLS handshake failed.
- Any other transport-layer error that prevented the test from executing.

**CI usage:**
```bash
scrutineer run
rc=$?
if [ $rc -eq 2 ]; then
  echo "Target unreachable -- is the service running?"
  exit 1
fi
```

This exit code is distinct from test failures because it indicates an infrastructure problem rather than a test logic problem. Retrying after a connection error may succeed without code changes.

---

### 3 -- Configuration Error

**Constant:** `exitcode.ConfigError`

A configuration or YAML parse error prevented scrutineer from running.

**When it occurs:**
- The `scrutineer.yaml` file was not found or could not be read.
- The config file or a test suite file contained invalid YAML.
- Validation failed (e.g., no tests listed, invalid reporter type, negative parallelism).
- No test files were found in the manifest.
- CLI flags could not be parsed.
- An unknown command or subcommand was provided.
- A required argument was missing (e.g., `scrutineer log-dump` without a file path).

**CI usage:**
```bash
scrutineer run
rc=$?
if [ $rc -eq 3 ]; then
  echo "Configuration error -- check scrutineer.yaml syntax"
  exit 1
fi
```

---

### 4 -- Internal Error

**Constant:** `exitcode.InternalError`

An unexpected framework or internal error occurred.

**When it occurs:**
- The telemetry log file could not be created or written.
- An unexpected panic was caught internally.
- A record in a telemetry log file was corrupted during `log-dump`.
- Any other error not attributable to test logic, configuration, or connectivity.

**CI usage:**
```bash
scrutineer run
rc=$?
if [ $rc -eq 4 ]; then
  echo "Internal error -- this may be a bug in scrutineer"
  exit 1
fi
```

---

## CI/CD Script Pattern

A comprehensive shell script handling all exit codes:

```bash
#!/bin/bash
set -euo pipefail

scrutineer run --format json --config ci/scrutineer.yaml
rc=$?

case $rc in
  0)
    echo "All tests passed"
    ;;
  1)
    echo "Test failures detected"
    exit 1
    ;;
  2)
    echo "Connection error -- target service may be down"
    exit 2
    ;;
  3)
    echo "Configuration error -- check YAML files"
    exit 3
    ;;
  4)
    echo "Internal framework error"
    exit 4
    ;;
  *)
    echo "Unexpected exit code: $rc"
    exit $rc
    ;;
esac
```

Note: with `set -e`, the script exits on the first non-zero exit code. To capture the exit code without exiting, use `scrutineer run || rc=$?` or disable `set -e` before the call.

---

## Programmatic Access

The `exitcode.String()` function converts a code to a human-readable description:

```go
import "github.com/scrutineer/scrutineer/core/exitcode"

desc := exitcode.String(1) // "one or more test assertions failed"
```

| Code | `exitcode.String()` output |
|------|---------------------------|
| 0 | `"all tests passed"` |
| 1 | `"one or more test assertions failed"` |
| 2 | `"connection or network error"` |
| 3 | `"configuration or YAML parse error"` |
| 4 | `"framework or internal error"` |
| other | `"unknown exit code"` |
