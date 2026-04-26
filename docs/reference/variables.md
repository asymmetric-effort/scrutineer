# Variable Interpolation Reference

Scrutineer supports variable interpolation in YAML test definitions using `${...}` expressions. Variables resolve fixtures, captured values from prior steps, and environment variables. The implementation is in `core/fixture/fixture.go`.

---

## Variable Syntax

Variables use the form `${<prefix>.<path>}` where:

- **prefix** identifies the variable source: `fixture`, `capture`, or `env`.
- **path** is a dot-notation key path (for `fixture` and `capture`) or a plain name (for `env`).

```yaml
steps:
  - action: request
    url: "${fixture.base_url}/users/${capture.user_id}"
    headers:
      Authorization: "Bearer ${env.API_TOKEN}"
```

---

## Variable Types

### ${fixture.*} -- Fixtures

Resolves values from the `fixtures` section of the test suite YAML. Fixtures are static data loaded before test execution.

```yaml
fixtures:
  base_url: "https://api.example.com"
  user:
    name: "Alice"
    email: "alice@example.com"
  ids:
    - 1
    - 2
    - 3

tests:
  - name: Create user
    steps:
      - action: request
        url: "${fixture.base_url}/users"
        body:
          name: "${fixture.user.name}"
          email: "${fixture.user.email}"
```

Fixtures support nested dot-notation paths. The path `fixture.user.name` navigates: `fixtures["user"]["name"]`.

---

### ${capture.*} -- Captured Values

Resolves values captured from previous test step results. Captures are defined in a step's `capture` block, which maps variable names to dot-notation extraction paths.

```yaml
steps:
  - action: request
    method: POST
    url: "${fixture.base_url}/users"
    body:
      name: "Alice"
    capture:
      user_id: "body.id"
      user_name: "body.name"

  - action: request
    method: GET
    url: "${fixture.base_url}/users/${capture.user_id}"
    assert:
      - operator: equal
        expected: "${capture.user_name}"
```

The `capture` block on a step uses the `Extract()` function (`core/fixture/capture.go`) to pull values from the step's result data using dot-notation paths. For example, `"body.id"` extracts `result["body"]["id"]`.

Captured values support nested path resolution just like fixtures:

```yaml
capture:
  token: "body.auth.token"
  role: "body.user.roles.0"
```

---

### ${env.*} -- Environment Variables

Resolves values from the process environment using `os.Getenv` / `os.LookupEnv`.

```yaml
steps:
  - action: request
    url: "https://api.example.com/users"
    headers:
      Authorization: "Bearer ${env.API_TOKEN}"
      X-Tenant: "${env.TENANT_ID}"
```

**Edge cases:**
- If the environment variable is not set (`LookupEnv` returns `false`), interpolation fails with an `unresolved variable` error.
- An environment variable set to the empty string `""` resolves successfully to `""`.
- Environment variable names do not support dot-notation nesting. The entire portion after `env.` is treated as the variable name: `${env.MY_VAR}` looks up `MY_VAR`.

---

## Dot-Notation Path Resolution

The `navigatePath()` function traverses nested `map[string]any` structures using dot-separated keys.

Given data:

```yaml
body:
  user:
    name: "Alice"
    address:
      city: "Portland"
```

| Path | Resolves To |
|------|-------------|
| `body` | `{"user": {"name": "Alice", "address": {"city": "Portland"}}}` |
| `body.user` | `{"name": "Alice", "address": {"city": "Portland"}}` |
| `body.user.name` | `"Alice"` |
| `body.user.address.city` | `"Portland"` |
| `body.missing` | resolution failure |
| `body.user.name.first` | resolution failure (cannot navigate through string) |

Path resolution returns `(value, true)` on success or `(nil, false)` on failure. Failures occur when:

- A key does not exist in the map at any level.
- An intermediate value is not a `map[string]any` (e.g., trying to traverse through a string or number).

---

## String Interpolation Rules

The `Interpolate()` method processes a string left-to-right, replacing each `${...}` expression with the resolved value formatted via `fmt.Sprintf("%v", val)`.

### Rules

1. `${ref}` is replaced with the string representation of the resolved value.
2. Multiple variables can appear in a single string: `"${fixture.scheme}://${fixture.host}:${fixture.port}"`.
3. Non-string resolved values are converted to their string representation: integers become `"42"`, booleans become `"true"`, etc.
4. If a variable cannot be resolved, interpolation fails with an error: `unresolved variable: <ref>`.

### Escape Syntax

Use `\${` to produce a literal `${` in the output without triggering interpolation:

```yaml
steps:
  - action: request
    body:
      template: "Hello \${name}"  # produces "Hello ${name}" literally
```

The backslash is consumed; only the `$` is written to the output.

### Unclosed Braces

If `${` appears without a closing `}`, the `$` character is written literally and processing continues. This is not an error.

```yaml
# "${unclosed" produces "${unclosed" literally (no closing brace found)
```

---

## InterpolateMap -- Recursive Map Interpolation

The `InterpolateMap()` method recursively interpolates all string values in a `map[string]any` structure. Non-string values (integers, booleans, nested maps, slices) are processed recursively or passed through unchanged.

### Behavior by Value Type

| Type | Behavior |
|------|----------|
| `string` | `Interpolate()` is called; `${...}` expressions are resolved. |
| `map[string]any` | `InterpolateMap()` is called recursively. |
| `[]any` | Each element is recursively interpolated. |
| All other types | Passed through unchanged (integers, floats, booleans, nil). |

### Example

Given fixtures `{"host": "api.example.com", "port": 8080}` and input map:

```yaml
url: "https://${fixture.host}:${fixture.port}/api"
headers:
  Authorization: "Bearer ${env.TOKEN}"
body:
  count: 42
  items:
    - name: "${fixture.item_name}"
    - name: "static"
```

After `InterpolateMap()`:

```yaml
url: "https://api.example.com:8080/api"
headers:
  Authorization: "Bearer <resolved-token>"
body:
  count: 42           # integer, unchanged
  items:
    - name: "Widget"  # resolved from fixture
    - name: "static"  # no variables, unchanged
```

---

## Store

The `Store` struct (`core/fixture/fixture.go`) is the central variable resolution engine. It holds three namespaces:

| Namespace | Source | Populated |
|-----------|--------|-----------|
| `fixtures` | `fixtures` section of YAML test suite | At suite load time |
| `captures` | `capture` blocks on test steps | During test execution via `SetCapture()` |
| `env` | Process environment | On demand via `os.Getenv` / `os.LookupEnv` |

### API

```go
// Create a store with fixtures from the test suite.
store := fixture.NewStore(suite.Fixtures)

// Capture a value from a step result.
store.SetCapture("user_id", 42)

// Retrieve a captured value.
val, ok := store.GetCapture("user_id")

// Resolve a variable reference.
val, ok := store.Resolve("fixture.user.name")
val, ok := store.Resolve("capture.user_id")
val, ok := store.Resolve("env.API_KEY")

// Interpolate a string.
result, err := store.Interpolate("Hello ${fixture.user.name}")

// Recursively interpolate all strings in a map.
resolved, err := store.InterpolateMap(stepParams)
```

---

## Value Extraction

The `Extract()` function (`core/fixture/capture.go`) extracts a value from a `map[string]any` result using a dot-notation path. It is used by the capture mechanism to pull values from step results.

```go
data := map[string]any{
    "body": map[string]any{
        "user": map[string]any{
            "id":   42,
            "name": "Alice",
        },
    },
}

val, err := fixture.Extract(data, "body.user.id") // 42, nil
val, err := fixture.Extract(data, "body.missing")  // nil, error: key "missing" not found
val, err := fixture.Extract(nil, "body")            // nil, error: cannot extract from nil map
```

---

## Parameterized Tests

The `fixture` package also supports parameterized test expansion via `Expand()` (`core/fixture/parameterized.go`). A test template combined with multiple parameter sets produces multiple expanded tests.

### ParameterSet

```yaml
parameters:
  - name: "admin"
    values:
      role: "admin"
      expected_status: 200
  - name: "guest"
    values:
      role: "guest"
      expected_status: 403
```

### Expansion

Each parameter set produces a separate test with the name `"OriginalName [paramSet.Name]"`:

- `"Access Control [admin]"` with `{role: "admin", expected_status: 200}`
- `"Access Control [guest]"` with `{role: "guest", expected_status: 403}`

Steps are deep-copied for each expanded test so mutations in one do not affect others. The `Params` map on each `ExpandedTest` can be merged into the fixture store for variable interpolation.

If no parameter sets are provided, `Expand()` returns `nil` (no tests generated).
