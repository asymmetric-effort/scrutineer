# Assertion Reference

This document covers every assertion type available in scrutineer's `assert` blocks. Assertions are built from YAML operator strings by `DefaultBuilder.Build()` (defined in `core/assertion/assertion.go`).

Each assertion implements the `Assertion` interface:

```go
type Assertion interface {
    Name() string
    Evaluate(actual any) error
}
```

Failed assertions return an `*AssertionError` with structured fields: `Assertion`, `Expected`, `Actual`, `Message`, and optionally `Path`.

---

## Equality Assertions

### equal

**Operator strings:** `equal`, `eq`

Checks that the actual value equals the expected value using Go's `==` operator. Falls back to numeric comparison (via `float64` conversion) when types differ but both values are numeric.

```yaml
assert:
  - operator: equal
    expected: "hello world"
```

```yaml
assert:
  - operator: eq
    expected: 200
```

**Edge cases:**
- Mixed numeric types are handled: `int(42)` equals `float64(42.0)`.
- Non-numeric types must match exactly, including type. `"42"` does not equal `42`.
- `nil == nil` passes.

---

### not_equal

**Operator strings:** `not_equal`, `neq`

Checks that the actual value does not equal the expected value. Uses both `!=` and numeric comparison to ensure mismatched numeric types that represent the same value are still considered equal (and thus fail this assertion).

```yaml
assert:
  - operator: not_equal
    expected: "error"
```

```yaml
assert:
  - operator: neq
    expected: 0
```

**Edge cases:**
- `int(42)` is considered equal to `float64(42.0)`, so `not_equal` with expected `42.0` fails when actual is `int(42)`.

---

### deep_equal

**Operator string:** `deep_equal`

Checks deep structural equality using `reflect.DeepEqual`. Use this for comparing maps, slices, and nested structures.

```yaml
assert:
  - operator: deep_equal
    expected:
      name: "Alice"
      age: 30
      roles:
        - "admin"
        - "user"
```

**Edge cases:**
- Order matters for slices: `[1, 2]` does not deep-equal `[2, 1]`.
- Map key order does not matter.
- `nil` slice and empty slice are not deep-equal in Go's `reflect.DeepEqual`.

---

## String Assertions

### contains

**Operator string:** `contains`

Checks that the actual string contains the given substring. Both the operator's expected value and the actual value must be strings.

```yaml
assert:
  - operator: contains
    expected: "success"
```

**Edge cases:**
- Fails with an error if the actual value is not a string (reports the type).
- The expected value must be a string at build time; non-string values cause a build error.
- Empty string `""` is contained in every string.

---

### not_contains

**Operator string:** `not_contains`

Checks that the actual string does not contain the given substring.

```yaml
assert:
  - operator: not_contains
    expected: "error"
```

**Edge cases:**
- Same type requirements as `contains`.
- An empty expected string `""` causes this assertion to always fail (every string contains `""`).

---

### has_prefix

**Operator string:** `has_prefix`

Checks that the actual string starts with the given prefix.

```yaml
assert:
  - operator: has_prefix
    expected: "Bearer "
```

**Edge cases:**
- Actual value must be a string.
- Empty prefix `""` matches every string.

---

### has_suffix

**Operator string:** `has_suffix`

Checks that the actual string ends with the given suffix.

```yaml
assert:
  - operator: has_suffix
    expected: ".json"
```

**Edge cases:**
- Actual value must be a string.
- Empty suffix `""` matches every string.

---

### matches

**Operator string:** `matches`

Checks that the actual string matches a regular expression pattern. The pattern is compiled at build time using Go's `regexp.Compile` (RE2 syntax).

```yaml
assert:
  - operator: matches
    expected: "^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$"
```

**Edge cases:**
- Invalid regex patterns cause a build-time error (not a runtime assertion failure).
- Actual value must be a string.
- Uses Go's RE2 syntax; backreferences and lookaheads are not supported.
- Partial matches pass; use `^...$` anchors for full-string matching.

---

## Numeric Assertions

All numeric assertions convert both actual and expected values to `float64` for comparison. Supported input types: `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`, and `time.Duration`.

### greater_than

**Operator strings:** `greater_than`, `gt`

Checks that actual > expected.

```yaml
assert:
  - operator: greater_than
    expected: 0
```

```yaml
assert:
  - operator: gt
    expected: 100
```

**Edge cases:**
- Non-numeric actual or expected values produce an assertion error (not a build error).

---

### less_than

**Operator strings:** `less_than`, `lt`

Checks that actual < expected.

```yaml
assert:
  - operator: less_than
    expected: 1000
```

---

### greater_or_equal

**Operator strings:** `greater_or_equal`, `gte`

Checks that actual >= expected.

```yaml
assert:
  - operator: greater_or_equal
    expected: 1
```

---

### less_or_equal

**Operator strings:** `less_or_equal`, `lte`

Checks that actual <= expected.

```yaml
assert:
  - operator: less_or_equal
    expected: 500
```

---

### in_range

**Operator string:** `in_range`

Checks that actual is within the inclusive range [min, max]. The `min` and `max` values are provided in the `options` map, not in `expected`.

```yaml
assert:
  - operator: in_range
    min: 200
    max: 299
```

**Edge cases:**
- Both `min` and `max` must be present in options; missing either causes a build error.
- The range is inclusive on both ends: `min <= actual <= max`.
- All three values (actual, min, max) must be numeric.

---

## JSON / Structured Data Assertions

### json_path

**Operator string:** `json_path`

Extracts a value from a `map[string]any` using a dot-notation path, then compares it to an expected value. Comparison uses `reflect.DeepEqual` with a `float64` numeric fallback.

The `expected` field in the operator definition is the dot-notation path string. The value to compare against is provided in `options["expected"]`.

```yaml
assert:
  - operator: json_path
    expected: "body.user.name"
    options:
      expected: "Alice"
```

```yaml
assert:
  - operator: json_path
    expected: "data.items.0.id"
    options:
      expected: 42
```

**Edge cases:**
- The actual value must be `map[string]any` (the standard type produced by JSON unmarshalling).
- Path traversal fails if an intermediate value is not a map; the error includes the partial path.
- Missing keys produce an error identifying which key was not found.
- Numeric values at the path are compared via `float64` fallback if `DeepEqual` fails.

---

### not_empty

**Operator string:** `not_empty`

Checks that a value is not empty. Handles `nil`, strings, slices, arrays, and maps.

```yaml
assert:
  - operator: not_empty
```

**Edge cases:**
- `nil` always fails.
- Empty string `""` fails.
- Empty slice `[]` or empty map `{}` fails.
- Zero-value integers and booleans (`0`, `false`) pass -- only collection/string emptiness is checked.

---

## HTTP Status Assertions

### status_code

**Operator string:** `status_code`

Checks that the actual value (an integer status code) matches exactly.

```yaml
assert:
  - operator: status_code
    expected: 200
```

```yaml
assert:
  - operator: status_code
    expected: 404
```

**Edge cases:**
- The expected value must be convertible to `int`. Float values like `200.0` are truncated to `200`.
- The actual value must also be numeric.

---

### status_class

**Operator string:** `status_class`

Checks that the actual status code belongs to an HTTP status class. Valid classes: `1xx`, `2xx`, `3xx`, `4xx`, `5xx`.

```yaml
assert:
  - operator: status_class
    expected: "2xx"
```

```yaml
assert:
  - operator: status_class
    expected: "4xx"
```

**Ranges:**

| Class | Min | Max |
|-------|-----|-----|
| `1xx` | 100 | 199 |
| `2xx` | 200 | 299 |
| `3xx` | 300 | 399 |
| `4xx` | 400 | 499 |
| `5xx` | 500 | 599 |

**Edge cases:**
- Unknown class strings (e.g., `"6xx"`, `"200"`) cause a build-time error.
- The actual value must be an integer.

---

## Header Assertions

Header assertions operate on the response headers, which must be provided as `map[string]string` or `map[string][]string`. All header name comparisons are case-insensitive (normalized to lowercase internally).

### header_equals

**Operator string:** `header_equals`

Checks that a specific header's value equals the expected value. The header name is provided in `options["header"]`.

```yaml
assert:
  - operator: header_equals
    expected: "application/json"
    options:
      header: "Content-Type"
```

**Edge cases:**
- If the header has multiple values (multi-valued header), any single match passes.
- Missing headers produce an assertion error.
- The expected value is converted to string via `fmt.Sprintf("%v", ...)` before comparison.

---

### header_contains

**Operator string:** `header_contains`

Checks that a specific header's value contains the given substring.

```yaml
assert:
  - operator: header_contains
    expected: "json"
    options:
      header: "Content-Type"
```

**Edge cases:**
- For multi-valued headers, passes if any single value contains the substring.
- Both expected and the header name must be strings.

---

### header_exists

**Operator string:** `header_exists`

Checks that a header with the given name exists. The expected value is the header name itself.

```yaml
assert:
  - operator: header_exists
    expected: "X-Request-Id"
```

**Edge cases:**
- Case-insensitive lookup. `"x-request-id"` matches `"X-Request-Id"`.
- Does not check the header's value, only its presence.

---

## Timing Assertions

### response_time_below

**Operator string:** `response_time_below`

Checks that the actual elapsed time (a `time.Duration`) is strictly less than the specified maximum.

```yaml
assert:
  - operator: response_time_below
    expected: "500ms"
```

```yaml
assert:
  - operator: response_time_below
    expected: "2s"
```

The expected value can be:
- A Go duration string (parsed via `time.ParseDuration`): `"100ms"`, `"1.5s"`, `"2m30s"`
- A `time.Duration` value directly
- An `int`, `int64`, or `float64` interpreted as nanoseconds

**Edge cases:**
- The actual value must be a `time.Duration`. Other types cause an assertion error.
- The comparison is strictly less-than: if actual equals `MaxDuration`, the assertion fails.

---

## Collection Assertions

### length

**Operator string:** `length`

Checks that the length of a value equals the expected integer. Works on strings, slices, arrays, maps, and channels.

```yaml
assert:
  - operator: length
    expected: 3
```

**Edge cases:**
- `nil` has an effective length of 0: passes if expected is `0`, fails otherwise.
- Types without a concept of length produce an assertion error.
- The expected value must be convertible to `int`.

---

### empty

**Operator string:** `empty`

Checks that a value is empty (length zero). Works on strings, slices, arrays, maps, and channels.

```yaml
assert:
  - operator: empty
```

**Edge cases:**
- `nil` is treated as empty (passes).
- Types without a concept of length produce an assertion error.

---

### collection_not_empty

**Operator string:** `collection_not_empty`

Checks that a slice, array, map, string, or channel is not empty.

```yaml
assert:
  - operator: collection_not_empty
```

**Edge cases:**
- `nil` fails.
- Types without a concept of length produce an assertion error.

---

### each

**Operator string:** `each`

Checks that every element of a slice or array passes an inner assertion. The inner assertion is provided in `options["assertion"]` and must itself be an `Assertion` instance.

```yaml
assert:
  - operator: each
    assertion:
      operator: greater_than
      expected: 0
```

**Edge cases:**
- The actual value must be a slice or array; `nil` causes an error.
- If the slice is empty, the assertion passes (vacuous truth).
- On failure, the error message includes the index of the first failing element.

---

### any

**Operator string:** `any`

Checks that at least one element of a slice or array passes the inner assertion.

```yaml
assert:
  - operator: any
    assertion:
      operator: equal
      expected: "admin"
```

**Edge cases:**
- The actual value must be a slice or array.
- An empty slice causes the assertion to fail (no element can pass).

---

### all

**Operator string:** `all`

Functionally identical to `each`. Checks that all elements pass the inner assertion. Provided as a semantic alias for readability.

```yaml
assert:
  - operator: all
    assertion:
      operator: not_empty
```

**Edge cases:**
- Same behavior as `each`, including vacuous truth for empty slices.

---

## Operator String Quick Reference

| Operator String | Alias | Assertion Type |
|-----------------|-------|----------------|
| `equal` | `eq` | EqualAssertion |
| `not_equal` | `neq` | NotEqualAssertion |
| `deep_equal` | | DeepEqualAssertion |
| `contains` | | ContainsAssertion |
| `not_contains` | | NotContainsAssertion |
| `has_prefix` | | HasPrefixAssertion |
| `has_suffix` | | HasSuffixAssertion |
| `matches` | | MatchesAssertion |
| `greater_than` | `gt` | GreaterThanAssertion |
| `less_than` | `lt` | LessThanAssertion |
| `greater_or_equal` | `gte` | GreaterOrEqualAssertion |
| `less_or_equal` | `lte` | LessOrEqualAssertion |
| `in_range` | | InRangeAssertion |
| `json_path` | | JSONPathAssertion |
| `not_empty` | | NotEmptyAssertion |
| `status_code` | | StatusCodeAssertion |
| `status_class` | | StatusClassAssertion |
| `header_equals` | | HeaderEqualsAssertion |
| `header_contains` | | HeaderContainsAssertion |
| `header_exists` | | HeaderExistsAssertion |
| `response_time_below` | | ResponseTimeBelowAssertion |
| `length` | | LengthAssertion |
| `empty` | | EmptyAssertion |
| `collection_not_empty` | | CollectionNotEmptyAssertion |
| `each` | | EachAssertion |
| `any` | | AnyAssertion |
| `all` | | AllAssertion |

## Error Format

All assertion failures produce an `*AssertionError` with this string format:

```
assertion "equal" failed: expected 42 to equal 100 (expected: 100, actual: 42)
```

For path-aware assertions (e.g., `json_path`):

```
assertion "json_path" failed at path "body.user.name": value at path "body.user.name" is Bob, expected Alice (expected: Alice, actual: Bob)
```
