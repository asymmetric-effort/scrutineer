# CI/CD Integration Guide

This guide covers integrating scrutineer into your CI/CD pipeline, including GitHub Actions setup, JSON output for machine parsing, exit code handling, coverage gates, Docker usage, and parallel execution.

## Exit Codes

Scrutineer uses specific exit codes to communicate results to CI systems:

| Code | Constant          | Meaning                                    |
|------|-------------------|--------------------------------------------|
| 0    | `OK`              | All tests passed                           |
| 1    | `TestFailure`     | One or more test assertions failed         |
| 2    | `ConnectionError` | Could not reach the target (network error) |
| 3    | `ConfigError`     | Configuration or YAML parse error          |
| 4    | `InternalError`   | Unexpected framework error                 |

In CI scripts, use these exit codes to determine the appropriate action:

```bash
scrutineer run --config scrutineer.yaml
EXIT_CODE=$?

case $EXIT_CODE in
  0) echo "All tests passed" ;;
  1) echo "Test failures detected" ; exit 1 ;;
  2) echo "Could not connect to target" ; exit 1 ;;
  3) echo "Configuration error" ; exit 1 ;;
  4) echo "Internal scrutineer error" ; exit 1 ;;
  *) echo "Unknown exit code: $EXIT_CODE" ; exit 1 ;;
esac
```

## JSON Output for CI

Use `--format json` to produce machine-readable output:

```bash
scrutineer run --format json > results.json
```

Or configure it in `scrutineer.yaml`:

```yaml
reporters:
  - type: json
    output: results.json
```

The JSON output contains structured test results that can be parsed by CI tools, dashboards, or custom scripts.

## GitHub Actions

### Basic Workflow

```yaml
name: Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - name: Install scrutineer
        run: go install github.com/scrutineer/scrutineer/cmd/scrutineer@latest

      - name: Run tests
        run: scrutineer run --format json > results.json

      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: test-results
          path: results.json
```

### With Browser Tests

```yaml
name: Browser Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  browser-test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - name: Install scrutineer
        run: go install github.com/scrutineer/scrutineer/cmd/scrutineer@latest

      - name: Install browser dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y \
            libnss3 libatk-bridge2.0-0 libdrm2 libxkbcommon0 \
            libgbm1 libpango-1.0-0 libcairo2 libasound2

      - name: Install browsers
        run: scrutineer browsers install

      - name: Start application
        run: |
          docker compose up -d
          sleep 5  # wait for services to be ready

      - name: Run browser tests
        run: scrutineer run --tags browser --format json > browser-results.json

      - name: Upload screenshots
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: screenshots
          path: screenshots/

      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: browser-test-results
          path: browser-results.json

      - name: Stop application
        if: always()
        run: docker compose down
```

### Matrix Strategy for Multiple Connectors

```yaml
name: Full Test Suite

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        test-suite:
          - name: api
            tags: api
          - name: cli
            tags: cli
          - name: browser
            tags: browser

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - name: Install scrutineer
        run: go install github.com/scrutineer/scrutineer/cmd/scrutineer@latest

      - name: Run ${{ matrix.test-suite.name }} tests
        run: |
          scrutineer run \
            --tags ${{ matrix.test-suite.tags }} \
            --format json \
            > results-${{ matrix.test-suite.name }}.json

      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: results-${{ matrix.test-suite.name }}
          path: results-${{ matrix.test-suite.name }}.json
```

## Coverage Gates

Scrutineer has built-in coverage gate support. Configure a minimum threshold in `scrutineer.yaml`:

```yaml
coverage:
  threshold: 98.0
```

When the test run completes, scrutineer checks if the percentage of tests executed meets the threshold. If coverage falls below the threshold, scrutineer prints a clear error message and exits with code 1 (TestFailure), even if all executed tests passed.

The error message format:

```
coverage 85.0% is below required threshold 98.0%
```

### Using Coverage Gates in CI

```yaml
- name: Run tests with coverage gate
  run: |
    scrutineer run
    # Exit code 1 if coverage < threshold, even if all tests pass
```

For more granular control, you can parse the JSON output:

```bash
scrutineer run --format json > results.json

# Extract coverage from JSON output and enforce threshold
COVERAGE=$(jq '.coverage.percent' results.json)
THRESHOLD=98.0

if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
  echo "Coverage $COVERAGE% is below required $THRESHOLD%"
  exit 1
fi
```

## Running in Docker

### Using the Scrutineer Binary

Build a Docker image with scrutineer for your test pipeline:

```dockerfile
FROM ubuntu:latest

# Install Go
RUN apt-get update && apt-get install -y wget && \
    wget -q https://go.dev/dl/go1.26.2.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.26.2.linux-amd64.tar.gz && \
    rm go1.26.2.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}"

# Install scrutineer
RUN go install github.com/scrutineer/scrutineer/cmd/scrutineer@latest

# Copy test files
WORKDIR /tests
COPY scrutineer.yaml .
COPY tests/ tests/

ENTRYPOINT ["scrutineer", "run"]
```

Build and run:

```bash
docker build -t my-tests .
docker run --rm my-tests --format json
```

### With Browser Support

```dockerfile
FROM ubuntu:latest

RUN apt-get update && apt-get install -y \
    wget ca-certificates \
    libnss3 libatk-bridge2.0-0 libdrm2 libxkbcommon0 \
    libgbm1 libpango-1.0-0 libcairo2 libasound2 \
    && rm -rf /var/lib/apt/lists/*

# Install Go and scrutineer
RUN wget -q https://go.dev/dl/go1.26.2.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.26.2.linux-amd64.tar.gz && \
    rm go1.26.2.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}"

RUN go install github.com/scrutineer/scrutineer/cmd/scrutineer@latest

# Install browsers
RUN scrutineer browsers install

WORKDIR /tests
COPY scrutineer.yaml .
COPY tests/ tests/

ENTRYPOINT ["scrutineer", "run"]
```

### Docker Compose for Test Services

If your tests need services (databases, APIs), use Docker Compose. Per the scrutineer project rules, all Docker images must be built from `ubuntu:latest`:

```yaml
# docker-compose.test.yaml
services:
  api:
    build:
      context: .
      dockerfile: Dockerfile.api
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 5s
      timeout: 3s
      retries: 5

  tests:
    build:
      context: .
      dockerfile: Dockerfile.tests
    depends_on:
      api:
        condition: service_healthy
    environment:
      - API_BASE_URL=http://api:8080
```

Run:

```bash
docker compose -f docker-compose.test.yaml run --rm tests
```

## Parallel Execution

### Within a Single Machine

Use the `--parallelism` flag or the `parallelism` config key to run test suites concurrently:

```bash
scrutineer run --parallelism 4
```

Or in `scrutineer.yaml`:

```yaml
parallelism: 4
```

This controls how many test suites execute simultaneously. Each suite's tests still run sequentially within the suite to preserve step ordering and capture state.

### Across CI Jobs

For maximum parallelism, split test files across CI jobs:

```yaml
# scrutineer-api.yaml
tests:
  - tests/api-users.test.yaml
  - tests/api-auth.test.yaml

# scrutineer-browser.yaml
tests:
  - tests/browser-login.test.yaml
  - tests/browser-dashboard.test.yaml
```

```yaml
# GitHub Actions
jobs:
  api-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: scrutineer run --config scrutineer-api.yaml

  browser-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: scrutineer run --config scrutineer-browser.yaml
```

### Tag-Based Splitting

Use tags to split execution without separate config files:

```bash
# Job 1
scrutineer run --tags api

# Job 2
scrutineer run --tags browser

# Job 3
scrutineer run --tags cli
```

## Telemetry in CI

Enable telemetry to capture detailed timing data for post-run analysis:

```yaml
telemetry:
  enabled: true
  output: scrutineer.log
```

The binary telemetry log can be dumped to text with:

```bash
scrutineer log-dump scrutineer.log
```

Upload the telemetry log as a CI artifact for debugging failed runs:

```yaml
- name: Upload telemetry
  if: always()
  uses: actions/upload-artifact@v4
  with:
    name: telemetry
    path: scrutineer.log
```

## Next Steps

- [Installation](installation.md) -- installing scrutineer
- [Writing Tests](writing-tests.md) -- test file structure
- [Load Testing](load-testing.md) -- performance testing in CI
