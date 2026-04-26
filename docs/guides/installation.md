# Installation Guide

## System Requirements

- **Go 1.26+** (required for building from source)
- **Operating systems**: Linux, macOS, Windows
- **Architectures**: amd64, arm64
- **Disk space**: ~50 MB for the binary; ~500 MB additional if installing browser binaries

## Installing with Go

If you have Go 1.26+ installed:

```bash
go install github.com/scrutineer/scrutineer/cmd/scrutineer@latest
```

Verify the installation:

```bash
scrutineer version
```

## Pre-Built Binaries

Download a pre-built binary from the [releases page](https://github.com/scrutineer/scrutineer/releases). Binaries are available for all supported platforms:

| Platform       | Binary Name                          |
|----------------|--------------------------------------|
| Linux amd64    | `scrutineer-linux-amd64`             |
| Linux arm64    | `scrutineer-linux-arm64`             |
| macOS amd64    | `scrutineer-darwin-amd64`            |
| macOS arm64    | `scrutineer-darwin-arm64`            |
| Windows amd64  | `scrutineer-windows-amd64.exe`       |
| Windows arm64  | `scrutineer-windows-arm64.exe`       |

After downloading:

```bash
# Linux / macOS
chmod +x scrutineer-linux-amd64
sudo mv scrutineer-linux-amd64 /usr/local/bin/scrutineer

# Verify
scrutineer version
```

## Building from Source

### Clone the Repository

```bash
git clone https://github.com/scrutineer/scrutineer.git
cd scrutineer
```

### Build for Your Platform

Using `make`:

```bash
make build
```

This compiles the binary into `bin/scrutineer` with the current Git tag embedded as the version string. The build uses linker flags to set the version:

```
-ldflags "-X main.version=$(VERSION)"
```

Using `go build` directly:

```bash
cd cmd/scrutineer
go build -o ../../bin/scrutineer .
```

### Cross-Compile for All Platforms

```bash
make cross
```

This builds binaries for all six supported platform/architecture combinations and places them in the `bin/` directory:

- `bin/scrutineer-linux-amd64`
- `bin/scrutineer-linux-arm64`
- `bin/scrutineer-darwin-amd64`
- `bin/scrutineer-darwin-arm64`
- `bin/scrutineer-windows-amd64.exe`
- `bin/scrutineer-windows-arm64.exe`

### Go Workspace

Scrutineer uses a Go workspace (`go.work`) to manage its multi-module structure. The workspace includes:

```
go.work
  ./cmd/scrutineer
  ./core
  ./connector/browser
  ./connector/cli
  ./connector/grpc
  ./connector/http
  ./connector/ssh
  ./fuzz
  ./loadtest
```

When building from source, Go automatically resolves module dependencies through the workspace file. No additional setup is required.

## Verifying Your Installation

Run the version command to confirm the binary is working:

```bash
scrutineer version
```

Expected output:

```
scrutineer 0.0.1
  go:       go1.26.2
  os/arch:  linux/amd64
```

Run the help command to see available commands:

```bash
scrutineer help
```

Expected output:

```
scrutineer -- extensible test framework

Usage:
  scrutineer <command> [options]

Commands:
  run              Run tests from scrutineer.yaml manifest
  log-dump <file>  Dump binary telemetry log to stdout
  browsers         Manage browser installations
  version          Print version information
  help             Show this help

Run Options:
  --config <file>      Config file (default: scrutineer.yaml)
  --parallelism <n>    Number of parallel tests
  --timeout <dur>      Default test timeout
  --format <type>      Output format: ansi, json
  --verbose            Verbose output
  --tags <tags>        Filter tests by tags (comma-separated)
```

## Installing Browsers (Optional)

Browser testing requires Playwright's patched browser builds (Chromium, Firefox, WebKit). These are separate downloads managed by scrutineer.

### Install All Browsers

```bash
scrutineer browsers install
```

This downloads known-good versions of Chromium, Firefox, and WebKit from vendor CDNs. The browsers are stored in a platform-specific directory under your home folder.

### List Installed Browsers

```bash
scrutineer browsers list
```

### Configuring Which Browsers to Use

In your `scrutineer.yaml`, enable the browsers you need:

```yaml
browsers:
  chromium: true
  firefox: false
  webkit: false
```

Only browsers that are both installed and enabled will be used for browser tests.

## Next Steps

- [Writing Tests](writing-tests.md) -- learn the YAML test format
- [Browser Testing](browser-testing.md) -- detailed browser automation guide
- [CI Integration](ci-integration.md) -- set up scrutineer in your CI pipeline
