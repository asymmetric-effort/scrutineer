# CLI Connector

The CLI connector executes local commands, captures their output (stdout, stderr, exit code), and validates filesystem side-effects. It is identified as `cli` in YAML test definitions.

Source: `connector/cli/`

## Setup Configuration

The `Setup` method accepts the following configuration keys:

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `work_dir` | `string` | No | Working directory for command execution. If omitted, the process inherits the current working directory. |
| `env` | `map[string]string` | No | Environment variables to set for all commands. Each key-value pair is appended to the process environment. Also accepts `map[string]any` where values must be strings. |

### Setup Example

```yaml
connector: cli
config:
  work_dir: /tmp/myproject
  env:
    PATH: /usr/local/bin:/usr/bin
    NODE_ENV: test
    DEBUG: "true"
```

## Actions

The CLI connector supports two actions: `exec` and `filesystem`.

---

## Action: `exec`

Runs a local command, captures stdout and stderr, and returns the exit code.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `command` | `string` | Yes | The command to execute. Parsed using shell-style quoting rules on non-Windows platforms. On Windows, passed directly to `cmd.exe /C`. |
| `stdin` | `string` | No | Data to send to the command's standard input. The stdin pipe is opened before the process starts, and data is written after the process has launched. The pipe is closed after all data is written. |

### Step Timeout

If `step.Timeout` is set (a `time.Duration` value), a context deadline is applied. When the deadline expires, the process is killed and the connector returns an error: `"cli: command timed out or was cancelled"`.

### Shell Argument Parsing

On non-Windows platforms, the `command` string is split into arguments using the `parseShellArgs` function, which implements shell-style quoting:

- **Single quotes** (`'...'`): everything inside is treated literally; no escape processing.
- **Double quotes** (`"..."`): backslash escapes are processed (e.g., `\"` becomes `"`).
- **Unquoted whitespace** (spaces, tabs): splits arguments.
- **Backslash inside double quotes**: the character following `\` is included literally.
- **Unterminated quotes**: returns an error (`"cli: unterminated quote in command"`).

Examples of argument parsing:

| Command String | Parsed Arguments |
|----------------|------------------|
| `echo hello world` | `["echo", "hello", "world"]` |
| `echo "hello world"` | `["echo", "hello world"]` |
| `echo 'hello world'` | `["echo", "hello world"]` |
| `echo "say \"hi\""` | `["echo", "say \"hi\""]` |
| `ls -la /tmp` | `["ls", "-la", "/tmp"]` |

### Cross-Platform Behavior (Windows)

On Windows (`runtime.GOOS == "windows"`), the `command` string is **not** parsed by `parseShellArgs`. Instead, it is wrapped as:

```
cmd.exe /C <command string>
```

This delegates all shell interpretation (pipes, redirects, environment variable expansion) to `cmd.exe`. The command string is passed as a single argument to `/C`.

### Stdin Handling

When `stdin` is provided:

1. A stdin pipe is created **before** the process starts (`cmd.StdinPipe()`).
2. The process is started.
3. The data is written to the pipe using `io.WriteString`.
4. The pipe is closed, signaling EOF to the process.

If `stdin` is empty or omitted, no pipe is created.

### Result Data Keys

| Key | Type | Description |
|-----|------|-------------|
| `stdout` | `string` | Captured standard output of the command. |
| `stderr` | `string` | Captured standard error of the command. |
| `exit_code` | `int` | The process exit code. `0` on success. Non-zero on failure. |
| `command` | `string` | The original command string (echoed back). |

Result metadata (`Meta`):
- `connector`: `"cli"`
- `action`: `"exec"`

### Exit Code Capture

- If the command exits with a non-zero status, the exit code is extracted from `exec.ExitError.ExitCode()`.
- If the context was cancelled or timed out, the connector returns an error rather than a result.
- If the command fails to start (e.g., binary not found), the connector returns an error.

### Examples

#### Simple command execution

```yaml
steps:
  - connector: cli
    action: exec
    parameters:
      command: echo "Hello, World!"
    assert:
      - path: stdout
        equals: "Hello, World!\n"
      - path: exit_code
        equals: 0
```

#### Command with environment and working directory

```yaml
connector: cli
config:
  work_dir: /home/user/project
  env:
    LANG: en_US.UTF-8
steps:
  - action: exec
    parameters:
      command: ls -la
```

#### Providing stdin data

```yaml
steps:
  - connector: cli
    action: exec
    parameters:
      command: cat
      stdin: "input data from test"
    assert:
      - path: stdout
        equals: "input data from test"
```

#### Capturing exit code from a failing command

```yaml
steps:
  - connector: cli
    action: exec
    parameters:
      command: grep "nonexistent" /dev/null
    assert:
      - path: exit_code
        equals: 1
```

#### Command with timeout

```yaml
steps:
  - connector: cli
    action: exec
    timeout: 5s
    parameters:
      command: sleep 100
```

This step will fail after 5 seconds with a timeout error.

#### Quoted arguments

```yaml
steps:
  - connector: cli
    action: exec
    parameters:
      command: echo "hello world" 'single quoted'
    assert:
      - path: stdout
        equals: "hello world single quoted\n"
```

---

## Action: `filesystem`

Checks file existence, content, and size constraints. This action does not execute a command; it inspects the filesystem directly.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | `string` | Yes | The file or directory path to inspect. |
| `exists` | `bool` | No | If set, asserts whether the path should exist (`true`) or not exist (`false`). When the assertion fails, an `error` key is set in the result data. |
| `contains` | `string` | No | A substring to search for in the file content. Only checked for regular files (not directories). Populates the `contains` boolean in results. |
| `size` | `map` | No | Size constraints. Supports `greater_than` (int64) and `less_than` (int64) sub-keys. Values are compared against the file size in bytes. |

### Result Data Keys

| Key | Type | Condition | Description |
|-----|------|-----------|-------------|
| `exists` | `bool` | Always | Whether the path exists on the filesystem. |
| `size` | `int64` | When path exists | File size in bytes. |
| `is_dir` | `bool` | When path exists | Whether the path is a directory. |
| `content` | `string` | When path exists and is a regular file | Full file content as a string. |
| `contains` | `bool` | When `contains` parameter is set and file is readable | Whether the file content contains the specified substring. |
| `size_greater_than` | `bool` | When `size.greater_than` is set | Whether the file size exceeds the threshold. |
| `size_less_than` | `bool` | When `size.less_than` is set | Whether the file size is below the threshold. |
| `error` | `string` | When `exists` assertion fails | Human-readable error message describing the mismatch. |

Result metadata (`Meta`):
- `connector`: `"cli"`
- `action`: `"filesystem"`

### Examples

#### Check file exists

```yaml
steps:
  - connector: cli
    action: filesystem
    parameters:
      path: /tmp/output.txt
      exists: true
    assert:
      - path: exists
        equals: true
```

#### Check file does not exist

```yaml
steps:
  - connector: cli
    action: filesystem
    parameters:
      path: /tmp/should-not-exist.txt
      exists: false
```

#### Check file content

```yaml
steps:
  - connector: cli
    action: filesystem
    parameters:
      path: /tmp/config.json
      contains: '"version"'
    assert:
      - path: contains
        equals: true
```

#### Check file size constraints

```yaml
steps:
  - connector: cli
    action: filesystem
    parameters:
      path: /tmp/data.bin
      size:
        greater_than: 1024
        less_than: 1048576
    assert:
      - path: size_greater_than
        equals: true
      - path: size_less_than
        equals: true
```

#### Read full file content

```yaml
steps:
  - connector: cli
    action: filesystem
    parameters:
      path: /tmp/greeting.txt
    assert:
      - path: content
        equals: "Hello, World!"
```

#### Check if path is a directory

```yaml
steps:
  - connector: cli
    action: filesystem
    parameters:
      path: /tmp/mydir
    assert:
      - path: is_dir
        equals: true
```

## Teardown

The CLI connector's `Teardown` method is a no-op. There are no persistent resources to clean up.
