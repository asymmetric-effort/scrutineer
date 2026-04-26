# Configuration Reference

Scrutineer is configured through a `scrutineer.yaml` file at the project root. This file controls test discovery, execution settings, reporter output, coverage thresholds, browser configuration, connector defaults, and telemetry.

The configuration is defined by `schema.Config` in `core/schema/config.go`, loaded by `config.Load()` in `core/config/config.go`, and merged with CLI flag overrides by `config.Merge()`.

---

## Loading Order

1. Defaults are applied (`core/config/defaults.go`).
2. `scrutineer.yaml` is parsed and merged on top of defaults.
3. CLI flags override any values set in the file.

The config file path defaults to `./scrutineer.yaml`. Override with `--config <path>`.

---

## Top-Level Fields

### version

**Type:** `string`
**Default:** `""` (empty)
**Required:** No

The configuration format version. Reserved for future use to support schema migrations.

```yaml
version: "1"
```

---

### tests

**Type:** `[]string` (list of file paths)
**Default:** `[]` (empty)
**Required:** Yes (validation fails if empty)

Explicit list of test files to run. Scrutineer uses a manifest-based approach -- test files are not auto-discovered. Paths are relative to the working directory.

```yaml
tests:
  - tests/api/users.yaml
  - tests/api/products.yaml
  - tests/cli/smoke.yaml
```

**CLI override:** `--tags <tags>` replaces the test list with the provided tags (comma-separated). This is a filtering mechanism, not an append.

---

### parallelism

**Type:** `int`
**Default:** `1`
**Required:** No

Number of parallel test workers. Controls how many tests execute concurrently.

```yaml
parallelism: 4
```

**CLI override:** `--parallelism <n>`

**Validation:** Must be >= 0. A value of 0 in the config file means "use the default" (1). Negative values cause a validation error.

---

### timeout

**Type:** `string` (Go duration format)
**Default:** `"30s"`
**Required:** No

Default timeout for individual test steps. Uses Go's `time.ParseDuration` syntax: `"100ms"`, `"5s"`, `"1m30s"`, `"2h"`.

```yaml
timeout: "10s"
```

**CLI override:** `--timeout <duration>`

---

### reporters

**Type:** `[]ReporterConfig`
**Default:** `[{type: "ansi"}]`
**Required:** No

List of reporter configurations. Each reporter has a `type` and optional `output` path.

```yaml
reporters:
  - type: ansi
  - type: json
    output: results.json
```

#### ReporterConfig fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `type` | `string` | `"ansi"` | Reporter type. Valid values: `"ansi"`, `"json"`. |
| `output` | `string` | `""` | Output file path. Empty means stdout. |

**Valid reporter types:** `ansi`, `json`. Invalid types cause a validation error.

**CLI override:** `--format <type>` replaces the entire reporters list with a single reporter of the given type.

**ANSI reporter:** Color-coded terminal output designed for human readability.

**JSON reporter:** Machine-readable JSON output for CI/CD pipeline consumption.

---

### coverage

**Type:** `CoverageConfig`
**Default:** `{threshold: 98.0}`
**Required:** No

Coverage gate configuration. After test execution, scrutineer checks whether measured coverage meets the threshold. If it does not, the exit code is set to `1` (TestFailure) even if all tests passed.

```yaml
coverage:
  threshold: 95.0
```

#### CoverageConfig fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `threshold` | `float64` | `98.0` | Minimum required coverage percentage (0.0 - 100.0). Set to `0` to disable the coverage gate. |

---

### browsers

**Type:** `BrowsersConfig`
**Default:** `{chromium: false, firefox: false, webkit: false}`
**Required:** No

Enables browser engines for browser-automation tests. Each browser must be installed first via `scrutineer browsers install`.

```yaml
browsers:
  chromium: true
  firefox: false
  webkit: false
```

#### BrowsersConfig fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `chromium` | `bool` | `false` | Enable Chromium browser for tests. |
| `firefox` | `bool` | `false` | Enable Firefox browser for tests. |
| `webkit` | `bool` | `false` | Enable WebKit browser for tests. |

---

### connectors

**Type:** `map[string]map[string]any`
**Default:** `nil`
**Required:** No

Default configuration for connectors, keyed by connector name. Values are passed to the connector as default parameters. Individual test steps can override these.

```yaml
connectors:
  http:
    base_url: "https://api.example.com"
    timeout: "10s"
    headers:
      Authorization: "Bearer ${env.API_TOKEN}"
  cli:
    working_dir: "/app"
  ssh:
    host: "test-server.example.com"
    user: "deploy"
    key_file: "~/.ssh/id_ed25519"
  grpc:
    address: "localhost:50051"
    tls: true
```

The available connector names are: `cli`, `http`, `ssh`, `grpc`, `browser`. The exact keys within each connector's map depend on the connector implementation.

---

### telemetry

**Type:** `TelemetryConfig`
**Default:** `{enabled: true, output: "scrutineer.log"}`
**Required:** No

Controls the TLV binary telemetry log. When enabled, all test events (suite start/end, test start/pass/fail, assertions, requests, responses) are written to a binary log file.

```yaml
telemetry:
  enabled: true
  output: "results/test-run.log"
```

#### TelemetryConfig fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | `bool` | `true` | Enable or disable telemetry recording. |
| `output` | `string` | `"scrutineer.log"` | Path to the output binary log file. |

**CLI override:** `--verbose` sets `telemetry.enabled` to `true`.

---

## CLI Flags

All CLI flags are parsed by `config.ParseFlags()` in `core/config/flags.go`. Non-zero flag values override the corresponding config file values via `config.Merge()`.

| Flag | Type | Default | Overrides |
|------|------|---------|-----------|
| `--config <file>` | `string` | `""` | Config file path (default: `./scrutineer.yaml`) |
| `--parallelism <n>` | `int` | `0` | `parallelism` (0 means "don't override") |
| `--timeout <dur>` | `string` | `""` | `timeout` |
| `--format <type>` | `string` | `""` | `reporters` (replaces entire list) |
| `--verbose` | `bool` | `false` | `telemetry.enabled` (sets to `true`) |
| `--tags <tags>` | `string` | `""` | `tests` (replaces list; comma-separated) |

---

## Complete Example

```yaml
version: "1"

tests:
  - tests/api/users.yaml
  - tests/api/auth.yaml
  - tests/cli/deploy.yaml
  - tests/browser/login.yaml

parallelism: 4
timeout: "15s"

reporters:
  - type: ansi
  - type: json
    output: test-results.json

coverage:
  threshold: 95.0

browsers:
  chromium: true

connectors:
  http:
    base_url: "https://staging.example.com/api"
    headers:
      Accept: "application/json"
  cli:
    working_dir: "/app"

telemetry:
  enabled: true
  output: "logs/scrutineer.log"
```

---

## Validation

Configuration is validated by `schema.ValidateConfig()` in `core/schema/validate.go`. The following rules are enforced:

- At least one test path must be specified in `tests`.
- `parallelism` must not be negative.
- All reporter `type` values must be one of the valid types (`ansi`, `json`).

Validation errors are collected and reported together, not one at a time.
