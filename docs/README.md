# Scrutineer Documentation

## Getting Started

- [Getting Started](getting-started.md) — Quick introduction and first test

## Guides

- [Installation](guides/installation.md) — Install from source, pre-built binaries, browser setup
- [Writing Tests](guides/writing-tests.md) — Test structure, assertions, captures, fixtures, best practices
- [Browser Testing](guides/browser-testing.md) — Headless browser automation with Chromium, Firefox, WebKit
- [Load Testing](guides/load-testing.md) — Concurrent execution, ramp-up, distributed SSH nodes, metrics
- [Fuzz Testing](guides/fuzz-testing.md) — Declarative fuzz targets, corpus management, input generation
- [CI Integration](guides/ci-integration.md) — GitHub Actions, JSON output, exit codes, Docker

## Connector Reference

- [CLI](connectors/cli.md) — Process execution, stdin/stdout, filesystem assertions
- [HTTP](connectors/http.md) — REST APIs, TLS, authentication
- [GraphQL](connectors/graphql.md) — Queries, mutations, subscriptions, introspection
- [SSH](connectors/ssh.md) — Remote command execution, tunneling
- [gRPC](connectors/grpc.md) — Unary and streaming RPCs, protobuf, reflection
- [Browser](connectors/browser.md) — CDP automation, selectors, interactions, screenshots

## Reference

- [Assertions](reference/assertions.md) — All assertion types and operators
- [Configuration](reference/configuration.md) — scrutineer.yaml fields and CLI flags
- [CLI Commands](reference/cli.md) — All commands, subcommands, and flags
- [Exit Codes](reference/exit-codes.md) — Process exit codes and CI handling
- [Telemetry](reference/telemetry.md) — TLV binary log format specification
- [Variables](reference/variables.md) — Interpolation, fixtures, captures, environment

## Architecture

- [Overview](architecture/overview.md) — Module structure, data flow, engine design
- [Extending](architecture/extending.md) — Writing custom connectors and assertions

## Contributing

- [Development](contributing/development.md) — Repo setup, testing, code style, hooks
- [Versioning](contributing/versioning.md) — Semver policy, release process

## Examples

- [REST API](examples/rest-api.yaml) — CRUD operations with auth and captures
- [CLI Tool](examples/cli-tool.yaml) — Process execution and filesystem checks
- [gRPC Service](examples/grpc-service.yaml) — Unary and streaming RPCs
- [Browser Login](examples/browser-login.yaml) — Login flow with form interaction
- [Load Test](examples/load-test.yaml) — Concurrent load with metrics
