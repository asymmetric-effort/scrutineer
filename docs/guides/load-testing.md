# Load Testing Guide

Scrutineer includes a built-in load testing framework that supports configurable concurrency, linear ramp-up scheduling, detailed latency metrics, and distributed execution across multiple nodes via SSH.

## Overview

The load testing system consists of:

- **Orchestrator**: coordinates the entire load test run -- manages ramp-up, worker pools, and result aggregation
- **Worker pools**: concurrent goroutines that repeatedly execute a work function
- **Metrics collector**: thread-safe collection of latency data with percentile calculations
- **Ramp-up scheduler**: linear ramp-up from 1 worker to full concurrency
- **Distributed executor**: splits concurrency across SSH-connected remote nodes

## Configuration

A load test is defined by three parameters:

| Parameter     | Description                                      | Example  |
|---------------|--------------------------------------------------|----------|
| `concurrency` | Number of concurrent virtual users               | `100`    |
| `duration`    | Total test duration                              | `5m`     |
| `ramp_up`     | Time to reach full concurrency from 1 worker     | `30s`    |

### YAML Configuration Example

```yaml
version: "0.0.1"

tests:
  - tests/api-loadtest.test.yaml

connectors:
  http:
    base_url: "https://api.example.com"
    default_headers:
      Content-Type: "application/json"
    timeout: "30s"

loadtest:
  concurrency: 50
  duration: 5m
  ramp_up: 30s
```

## Concurrency and Ramp-Up

### How Ramp-Up Works

The ramp-up scheduler computes a linear schedule of steps from 1 worker at time 0 to the full concurrency at the end of the ramp-up period. Steps are evenly spaced.

For example, with `concurrency: 10` and `ramp_up: 9s`:

| Time | Active Workers |
|------|---------------|
| 0s   | 1             |
| 1s   | 2             |
| 2s   | 3             |
| 3s   | 4             |
| 4s   | 5             |
| 5s   | 6             |
| 6s   | 7             |
| 7s   | 8             |
| 8s   | 9             |
| 9s   | 10            |

**Special cases:**
- If `concurrency` is 1, a single worker starts immediately
- If `ramp_up` is 0, all workers start simultaneously at time 0

### Worker Behavior

Each worker repeatedly calls the work function in a loop until the test duration expires. Workers check for context cancellation between iterations, so they stop promptly when the test ends. Results (latency and errors) are sent to a shared metrics collector via a buffered channel.

## Metrics Collected

The metrics collector records every request and produces a snapshot with the following data:

| Metric            | Description                                                |
|-------------------|------------------------------------------------------------|
| `TotalRequests`   | Total number of completed requests                         |
| `SuccessCount`    | Number of successful requests (no error)                   |
| `ErrorCount`      | Number of failed requests                                  |
| `MeanLatency`     | Arithmetic mean of all request latencies                   |
| `P50Latency`      | 50th percentile (median) latency                           |
| `P95Latency`      | 95th percentile latency                                    |
| `P99Latency`      | 99th percentile latency                                    |
| `MinLatency`      | Fastest request                                            |
| `MaxLatency`      | Slowest request                                            |
| `RequestsPerSec`  | Throughput (total requests / elapsed time)                  |
| `ElapsedTime`     | Total wall-clock time of the test                          |

### Percentile Calculation

Percentiles use linear interpolation between adjacent sorted values. For the p-th percentile:

1. All latencies are sorted
2. The index is computed as `p / 100 * (n - 1)` where n is the sample count
3. If the index falls between two values, linear interpolation is applied

This provides accurate percentile estimates even for small sample sizes.

## Running a Load Test

### Basic Example

```yaml
# tests/api-loadtest.test.yaml
suite: "API Load Test"

tests:
  - name: "GET /health under load"
    connector: http
    steps:
      - action: request
        method: GET
        path: /health
        assert:
          - field: status
            operator: equal
            expected: 200
```

Run with:

```bash
scrutineer run --config scrutineer.yaml
```

### Load Test with Multiple Endpoints

```yaml
suite: "API Load Test - Mixed Workload"

fixtures:
  test_user:
    name: "Load Test User"
    email: "loadtest@example.com"

tests:
  - name: "Health check"
    connector: http
    steps:
      - action: request
        method: GET
        path: /health
        assert:
          - field: status
            operator: status_class
            expected: "2xx"

  - name: "List users"
    connector: http
    steps:
      - action: request
        method: GET
        path: /api/users
        query:
          limit: "10"
        assert:
          - field: status
            operator: equal
            expected: 200
          - field: elapsed_ms
            operator: less_than
            expected: 2000

  - name: "Create and delete user"
    connector: http
    steps:
      - action: request
        method: POST
        path: /api/users
        body:
          name: ${fixture.test_user.name}
          email: ${fixture.test_user.email}
        capture:
          user_id: body.id
        assert:
          - field: status
            operator: equal
            expected: 201

      - action: request
        method: DELETE
        path: /api/users/${capture.user_id}
        assert:
          - field: status
            operator: equal
            expected: 204
```

## Distributed Testing

Scrutineer can distribute load across multiple remote nodes connected via SSH. This is useful when a single machine cannot generate enough load, or when you want to test from multiple geographic locations.

### How It Works

1. The total concurrency is split evenly across all configured nodes. If the concurrency does not divide evenly, the first nodes each receive one extra worker (e.g., 10 workers across 3 nodes = 4, 3, 3).
2. For each node, scrutineer connects via SSH using key-based authentication.
3. On each node, scrutineer executes the load test binary with the assigned concurrency.
4. JSON results are collected from each node's stdout and parsed.
5. Results from all nodes are aggregated into a combined summary.

### Distributed Configuration

```yaml
loadtest:
  concurrency: 300
  duration: 10m
  ramp_up: 1m

  distributed:
    binary: /usr/local/bin/scrutineer
    test_config: /opt/tests/loadtest.yaml
    nodes:
      - host: node1.example.com
        port: 22
        user: loadtest
        key_file: ~/.ssh/loadtest_key

      - host: node2.example.com
        port: 22
        user: loadtest
        key_file: ~/.ssh/loadtest_key

      - host: node3.example.com
        port: 22
        user: loadtest
        key_file: ~/.ssh/loadtest_key
```

### Node Requirements

Each remote node needs:

- The scrutineer binary installed at the configured `binary` path
- The test configuration file at the configured `test_config` path
- SSH access with key-based authentication
- Network access to the target system under test

### Concurrency Distribution

With the example above (300 concurrency, 3 nodes), each node receives 100 concurrent workers. If the total were 301, the distribution would be 101, 100, 100.

## Interpreting Results

### Sample Output

```
Load Test Results
  Duration:        5m0s
  Concurrency:     50
  Ramp-up:         30s

  Total Requests:  28547
  Successes:       28412
  Errors:          135
  Requests/sec:    95.16

  Mean Latency:    524.3ms
  P50 Latency:     487.1ms
  P95 Latency:     1.23s
  P99 Latency:     2.47s
  Min Latency:     12.4ms
  Max Latency:     5.01s

  Unique Errors:
    - context deadline exceeded
    - connection refused
```

### What to Look For

**Throughput (Requests/sec):**
Is the system handling the expected request rate? Compare against your SLA or baseline measurements.

**Latency distribution:**
- A large gap between P50 and P95 indicates tail latency issues -- most requests are fast, but some are slow
- A large gap between P95 and P99 suggests rare but severe slowdowns
- If mean latency is much higher than P50, outliers are pulling the average up

**Error rate:**
Calculate `ErrorCount / TotalRequests * 100` for the error percentage. Common errors include:
- `context deadline exceeded` -- requests timing out
- `connection refused` -- server rejecting connections under load

**Ramp-up behavior:**
If errors spike during ramp-up, the system may need time to warm up (JIT compilation, connection pools, caches). Consider a longer ramp-up period.

### Aggregated Distributed Results

When running distributed tests, the aggregated results combine data from all nodes:

- **TotalRequests**: sum across all nodes
- **MeanLatency**: weighted average based on per-node request counts
- **MinLatency**: minimum across all nodes
- **MaxLatency**: maximum across all nodes
- **RequestsPerSec**: total requests divided by the longest node elapsed time
- **Errors**: deduplicated union of all unique error messages

Note that per-node percentiles (P50, P95, P99) cannot be precisely merged without the full latency data. The aggregated percentile values are approximations based on the weighted combination of per-node statistics.

## Next Steps

- [Writing Tests](writing-tests.md) -- test file structure and assertions
- [Fuzz Testing](fuzz-testing.md) -- find edge cases with randomized input
- [CI Integration](ci-integration.md) -- automate load tests in your pipeline
