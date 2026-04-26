# SSH Connector

The SSH connector executes remote commands and creates SSH tunnels over authenticated SSH connections. It is identified as `ssh` in YAML test definitions.

Source: `connector/ssh/`

Dependency: `golang.org/x/crypto/ssh`

## Setup Configuration

The `Setup` method establishes an SSH connection using the provided configuration. The connection is reused across all steps until `Teardown` is called.

| Key | Type | Required | Default | Description |
|-----|------|----------|---------|-------------|
| `host` | `string` | Yes | -- | Remote SSH host. |
| `port` | `int` | No | `22` | Remote SSH port. Accepts `int`, `int64`, or `float64`. |
| `user` | `string` | Yes | -- | SSH username. |
| `key_file` | `string` | No | -- | Path to a PEM-encoded private key file on disk. |
| `key` | `string` | No | -- | Raw PEM-encoded private key (inline). |
| `password` | `string` | No | -- | Password for password-based authentication. |
| `host_key_check` | `bool` | No | `true` | Whether to verify the remote host key. Set to `false` to skip host key verification (equivalent to `StrictHostKeyChecking=no`). |

### Authentication Methods

At least one authentication method must be provided. Multiple methods can be specified and will be tried in the following order:

1. **Key file** (`key_file`): Reads a PEM private key from disk using `os.ReadFile`, parses it with `ssh.ParsePrivateKey`, and creates a `PublicKeys` auth method.

2. **Raw PEM key** (`key`): Parses the raw PEM string with `ssh.ParsePrivateKey` and creates a `PublicKeys` auth method.

3. **Password** (`password`): Uses `ssh.Password()` for password-based authentication.

If none of `key_file`, `key`, or `password` is provided, Setup returns an error.

### Host Key Verification

- When `host_key_check` is `true` (default): Host key checking is enabled. In the current implementation, this still uses `InsecureIgnoreHostKey()` as a fallback (a production implementation would load `~/.ssh/known_hosts`).
- When `host_key_check` is `false`: Uses `ssh.InsecureIgnoreHostKey()` to accept any host key.

### Connection

The connector dials the remote host using `ssh.Dial("tcp", host:port, config)`. The connection is established during `Setup` and must succeed before any steps can execute. Context cancellation is checked before dialing.

### Setup Examples

#### Key file authentication

```yaml
connector: ssh
config:
  host: server.example.com
  port: 22
  user: deploy
  key_file: /home/user/.ssh/id_ed25519
  host_key_check: false
```

#### Inline PEM key

```yaml
connector: ssh
config:
  host: 10.0.0.5
  user: admin
  key: |
    -----BEGIN OPENSSH PRIVATE KEY-----
    b3BlbnNzaC1rZXktdjEAAAAABG5vbmU...
    -----END OPENSSH PRIVATE KEY-----
  host_key_check: false
```

#### Password authentication

```yaml
connector: ssh
config:
  host: server.example.com
  user: testuser
  password: s3cretP@ss
  host_key_check: false
```

#### Non-standard port

```yaml
connector: ssh
config:
  host: bastion.example.com
  port: 2222
  user: admin
  key_file: /home/user/.ssh/id_rsa
  host_key_check: false
```

## Actions

The SSH connector supports two actions: `exec` and `tunnel`.

---

## Action: `exec`

Executes a command on the remote host over an SSH session. Each command creates a new SSH session on the existing connection.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `command` | `string` | Yes | The command to execute on the remote host. |
| `stdin` | `string` | No | Data to send to the command's standard input. Provided via `strings.NewReader`. |

### Context and Timeout

The command is run asynchronously in a goroutine. If the context is cancelled or the deadline expires:

1. An `ssh.SIGKILL` signal is sent to the remote process.
2. The connector returns `ctx.Err()` immediately.

### Exit Code Extraction

- On success: exit code is `0`.
- On `ssh.ExitError`: the exit status is extracted via `ExitStatus()`.
- On other errors: exit code is `-1`.

### Result Data Keys

| Key | Type | Description |
|-----|------|-------------|
| `stdout` | `string` | Captured standard output. |
| `stderr` | `string` | Captured standard error. |
| `exit_code` | `int` | The command's exit code. `0` on success, `-1` on non-exit errors. |

Result metadata (`Meta`):
- `connector`: `"ssh"`
- `action`: `"exec"`
- `command`: the command that was executed

### Examples

#### Run a remote command

```yaml
steps:
  - connector: ssh
    action: exec
    parameters:
      command: uname -a
    assert:
      - path: exit_code
        equals: 0
      - path: stdout
        contains: "Linux"
```

#### Command with stdin

```yaml
steps:
  - connector: ssh
    action: exec
    parameters:
      command: cat
      stdin: "Hello from scrutineer"
    assert:
      - path: stdout
        equals: "Hello from scrutineer"
```

#### Check a remote service

```yaml
steps:
  - connector: ssh
    action: exec
    parameters:
      command: systemctl is-active nginx
    assert:
      - path: stdout
        contains: "active"
      - path: exit_code
        equals: 0
```

#### Capture stderr

```yaml
steps:
  - connector: ssh
    action: exec
    parameters:
      command: ls /nonexistent
    assert:
      - path: exit_code
        not_equals: 0
      - path: stderr
        contains: "No such file or directory"
```

---

## Action: `tunnel`

Creates a local TCP listener that forwards connections through the SSH tunnel to a remote host and port. The tunnel remains open until `Teardown` is called.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `local_port` | `int` | Yes | Local port to listen on. Use `0` to let the OS assign an available port automatically. |
| `remote_host` | `string` | Yes | Remote host to forward connections to (as seen from the SSH server). |
| `remote_port` | `int` | Yes | Remote port to forward connections to. |

### Tunnel Behavior

1. A TCP listener is created on `127.0.0.1:<local_port>`.
2. The listener is stored for cleanup during `Teardown`.
3. A background goroutine accepts incoming connections.
4. For each accepted connection, a new goroutine:
   - Opens a TCP connection to `remote_host:remote_port` through the SSH client (`client.Dial("tcp", remote_addr)`).
   - Copies data bidirectionally using `io.Copy` in both directions.
   - Closes both connections when either direction finishes or the connector is shut down.

### Result Data Keys

| Key | Type | Description |
|-----|------|-------------|
| `local_addr` | `string` | The local address the tunnel is listening on (e.g., `"127.0.0.1:8080"` or `"127.0.0.1:54321"` if port 0 was used). |

Result metadata (`Meta`):
- `connector`: `"ssh"`
- `action`: `"tunnel"`
- `remote_addr`: the remote address being tunneled to (e.g., `"db.internal:5432"`)

### Examples

#### Forward to a remote database

```yaml
steps:
  - connector: ssh
    action: tunnel
    parameters:
      local_port: 15432
      remote_host: db.internal
      remote_port: 5432
    assert:
      - path: local_addr
        equals: "127.0.0.1:15432"
```

#### Auto-assigned local port

```yaml
steps:
  - connector: ssh
    action: tunnel
    parameters:
      local_port: 0
      remote_host: localhost
      remote_port: 6379
    assert:
      - path: local_addr
        not_empty: true
```

#### Tunnel to a web service

```yaml
steps:
  - connector: ssh
    action: tunnel
    parameters:
      local_port: 8080
      remote_host: internal-api.local
      remote_port: 80
```

## Session Management

- Each `exec` action creates a new SSH session on the existing connection (`client.NewSession()`).
- Sessions are closed after each command completes.
- The underlying SSH client connection is shared across all steps.
- Multiple tunnels can be active simultaneously.

## Teardown

The `Teardown` method performs cleanup in this order:

1. Signals all tunnel goroutines to stop by closing an internal `closed` channel.
2. Closes all active tunnel listeners.
3. Waits for all tunnel forwarding goroutines to complete (`tunnelWg.Wait()`).
4. Closes the SSH client connection.

Teardown is safe to call multiple times.
