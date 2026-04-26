# Fuzz Testing Guide

Scrutineer includes a declarative fuzz testing system that generates randomized inputs, feeds them to connectors, and checks that assertions hold for all inputs. Fuzz targets are defined in YAML, and scrutineer manages input generation, seed corpus storage, and failure reporting.

## Overview

Fuzz testing works by:

1. Defining a **fuzz target** with base parameters, fuzz fields, seed corpus, and assertions
2. Running the **seed corpus** first (known-good inputs that establish baseline behavior)
3. Generating **randomized mutations** of the fuzz fields and executing each against the connector
4. Recording any **failures** where the connector returns an error

The goal is to discover unexpected crashes, panics, or error responses that your normal test suite would miss.

## Defining Fuzz Targets

A fuzz target requires:

| Field        | Required | Description                                              |
|--------------|----------|----------------------------------------------------------|
| `name`       | Yes      | Human-readable name for the fuzz target                  |
| `connector`  | Yes      | Which connector to use (http, cli, grpc, etc.)           |
| `action`     | Yes      | The connector action to invoke                           |
| `parameters` | Yes      | Base parameters -- the starting point for mutation        |
| `fuzz_fields`| Yes      | List of parameter keys to mutate (must exist in parameters) |
| `seed`       | No       | Seed corpus entries (known inputs to try first)          |
| `assert`     | No       | Assertions that must hold for every input                |

### Example: Fuzzing an HTTP Endpoint

```yaml
name: "Fuzz user creation"
connector: http
action: request
parameters:
  method: POST
  path: /api/users
  body:
    name: "Alice"
    email: "alice@example.com"
    age: 25
fuzz_fields:
  - body.name
  - body.email
  - body.age
seed:
  - body.name: ""
    body.email: "not-an-email"
    body.age: -1
  - body.name: "A very long name that exceeds normal limits and keeps going"
    body.email: "valid@example.com"
    body.age: 999999
assert:
  - field: status
    operator: not_equal
    expected: 500
```

### Example: Fuzzing a CLI Tool

```yaml
name: "Fuzz JSON parser"
connector: cli
action: exec
parameters:
  command: "myapp parse --input"
  stdin: '{"key": "value"}'
fuzz_fields:
  - stdin
seed:
  - stdin: ""
  - stdin: "{}"
  - stdin: "not json at all"
  - stdin: '{"nested": {"deep": {"key": "val"}}}'
assert:
  - field: exit_code
    operator: not_equal
    expected: 139
```

## Fuzz Fields

The `fuzz_fields` list specifies which parameter keys the generator should mutate. Every field listed in `fuzz_fields` must exist as a key in the `parameters` map. The generator uses the initial value's type to choose an appropriate mutation strategy.

### Validation Rules

- At least one fuzz field is required
- Every fuzz field must exist in the parameters map
- The `name`, `connector`, and `action` fields are required

If validation fails, scrutineer reports a clear error before any execution begins.

## Input Generation Strategy

The generator produces mutations based on the type of each fuzz field's initial value.

### String Mutation

Strings are mutated with one of five strategies, chosen randomly:

| Strategy            | Description                                        |
|---------------------|----------------------------------------------------|
| Character replacement | Replace a random byte with a random printable ASCII character |
| Character insertion | Insert a random printable ASCII character at a random position |
| Character deletion  | Remove a character at a random position             |
| Bit flip            | Flip a random bit in a random byte                  |
| Full replacement    | Replace with a completely random string (1-50 chars)|

Empty strings are replaced with a random string of 1-10 characters.

### Integer Mutation

Integers are mutated with one of five strategies:

| Strategy       | Description                          |
|----------------|--------------------------------------|
| Zero           | Return 0                             |
| Max boundary   | Return `math.MaxInt32`               |
| Min boundary   | Return `math.MinInt32`               |
| Small delta    | Add a random value from -10 to +10   |
| Random         | Generate a random positive integer   |

### Float Mutation

Floats follow a similar pattern:

| Strategy       | Description                          |
|----------------|--------------------------------------|
| Zero           | Return 0.0                           |
| Max boundary   | Return `math.MaxFloat64`             |
| Min boundary   | Return `-math.MaxFloat64`            |
| Small delta    | Add a random value from -10 to +10   |
| Random         | Generate a random float (0-1000)     |

### Boolean Mutation

Booleans are simply negated (true becomes false, false becomes true).

### Nil Values

If a field's initial value is nil, the generator randomly produces one of:
- A random string (0-19 characters)
- A random integer (0-999)
- A random boolean

## Seed Corpus

The seed corpus is a list of known inputs to test before random generation begins. Seeds are useful for:

- **Boundary values**: empty strings, zero, negative numbers, maximum values
- **Known problematic inputs**: inputs that previously caused issues
- **Format edge cases**: malformed JSON, SQL injection strings, unicode

```yaml
seed:
  - body.name: ""
  - body.name: null
  - body.name: "Robert'); DROP TABLE users;--"
  - body.name: "\u0000\u0001\u0002"
  - body.name: "Alice"
    body.email: ""
```

Each seed entry is a map that overrides specific fuzz fields. Fields not present in a seed entry keep their base parameter values.

## Corpus Management

Scrutineer stores corpus entries as JSON files in a configurable directory. The corpus system supports:

### Loading a Corpus

```go
corpus := fuzz.NewCorpus("testdata/corpus/user-creation")
err := corpus.Load()
```

On load:
- The directory is created if it does not exist
- All `.json` files in the directory are read and parsed
- Corrupt (non-JSON) files are silently skipped
- Files are loaded in alphabetical order for deterministic behavior

### Adding Entries

When the fuzzer discovers an interesting input (e.g., one that triggers an error), it can be saved to the corpus:

```go
corpus.Add(map[string]any{
    "body.name": "input that caused a crash",
})
```

Each entry is written as a pretty-printed JSON file named `entry_<nanosecond_timestamp>.json`.

### Corpus Directory Structure

```
testdata/corpus/user-creation/
  entry_1714000000000000001.json
  entry_1714000000000000002.json
  entry_1714000000000000003.json
```

Each file contains one JSON object:

```json
{
  "body.name": "mutated value",
  "body.email": "another@mutation.com"
}
```

## Running Fuzz Tests

### Iteration Count

The `iterations` parameter controls how many random inputs to try:

- A positive number runs exactly that many iterations (after the seed corpus)
- Zero (`0`) runs indefinitely until the context is cancelled (useful for long-running fuzz campaigns)

### Execution Flow

1. **Validate** the target (name, connector, action, fuzz fields)
2. **Initialize** the generator with a seed derived from the current time
3. **Run seed corpus**: execute each seed entry in order
4. **Run generated inputs**: mutate fuzz fields and execute
5. For each input, call the connector's `Execute` method
6. If `Execute` returns an error, record it as a `FuzzFailure`
7. Return the `FuzzResult` with iteration count, failures, and total duration

### Fuzz Results

The result includes:

| Field        | Description                                    |
|--------------|------------------------------------------------|
| `Iterations` | Total number of inputs tested (seed + generated) |
| `Failures`   | List of failures, each with the input and error  |
| `Duration`   | Total wall-clock time of the fuzz run            |

Each failure records the exact input map that caused the error, making it easy to reproduce.

## Interpreting Results

### No Failures

If the fuzz run completes with zero failures, the connector handled all generated inputs gracefully. This does not guarantee correctness -- it means the connector did not return errors for any tested input.

### Failures Found

When failures are found:

1. **Examine the input**: what was the mutated value that caused the failure?
2. **Check the error**: is it a crash, panic, timeout, or unexpected status code?
3. **Add to seed corpus**: save interesting failures to the corpus so they are tested in future runs
4. **Fix and re-run**: fix the underlying issue and re-run the fuzz test to confirm

### Example Failure Output

```
Fuzz target "Fuzz user creation" completed:
  Iterations: 1000
  Failures:   3
  Duration:   45.2s

  Failure 1:
    Input: {"body.name": "\x00\x01\x02", "body.email": "alice@example.com", "body.age": 25}
    Error: execute: status 500: internal server error

  Failure 2:
    Input: {"body.name": "Alice", "body.email": "alice@example.com", "body.age": -2147483648}
    Error: execute: status 500: integer overflow

  Failure 3:
    Input: {"body.name": "Alice", "body.email": "L3&'qR", "body.age": 25}
    Error: execute: status 500: invalid email format caused panic
```

## Best Practices

1. **Start with seed corpus**: always include boundary values and known edge cases
2. **Assert on stability, not correctness**: fuzz assertions should check that the system does not crash (e.g., `status != 500`), not that it returns specific correct values
3. **Run long campaigns**: short fuzz runs (100 iterations) catch obvious bugs; longer runs (10,000+) find subtle issues
4. **Save interesting failures**: add discovered crash inputs to the seed corpus
5. **Fuzz one thing at a time**: focus fuzz fields on a specific input surface rather than fuzzing everything simultaneously

## Next Steps

- [Writing Tests](writing-tests.md) -- test file structure and assertions
- [Load Testing](load-testing.md) -- performance and scalability testing
- [CI Integration](ci-integration.md) -- automate fuzz testing in your pipeline
