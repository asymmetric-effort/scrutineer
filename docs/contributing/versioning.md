# Versioning Policy

Scrutineer follows [Semantic Versioning](https://semver.org/) (semver) for all releases.

## Version Format

```
MAJOR.MINOR.PATCH
```

The current development version is `0.0.1-dev`. The version is embedded in the binary at build time via linker flags:

```makefile
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.0.1-dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
```

## What Constitutes Each Version Bump

### Patch (0.0.x)

Patch releases fix bugs and make non-breaking configuration changes. Users can upgrade without modifying their test files.

Examples:
- Bug fixes in connectors (e.g., HTTP client not closing connections properly)
- Fixes to assertion evaluation logic
- Corrections to YAML parsing edge cases
- Telemetry format fixes
- Documentation corrections
- Performance improvements with no behavioral change
- Configuration default adjustments

### Minor (0.x.0)

Minor releases add new features and enhancements. Existing test files continue to work without changes.

Examples:
- New connector (e.g., adding a Redis connector)
- New assertion operators (e.g., adding a `between` operator)
- New browser actions (e.g., adding `drag_and_drop`)
- New reporter formats
- New CLI flags or commands
- New YAML schema fields (backward-compatible additions)
- New load testing or fuzz testing capabilities
- Improvements to error messages or reporting

### Major (x.0.0)

Major releases involve significant changes that may require users to update their test files or workflows. Major version bumps are determined manually based on the scope and impact of changes.

Examples:
- Breaking changes to the YAML test file format
- Removing or renaming connector actions
- Changing the Connector interface in ways that break existing implementations
- Changing assertion behavior (e.g., `equal` becoming case-insensitive)
- Changing exit code semantics
- Removing CLI commands or flags
- Changing the structure of JSON reporter output
- Breaking changes to the TLV telemetry format

## Pre-1.0 Development

While the version is below 1.0.0 (current: 0.0.1), the API is considered unstable. Breaking changes may occur in minor releases during this period. The project aims to reach 1.0.0 when the following milestones are met:

- All connectors (CLI, HTTP, SSH, gRPC, browser) are feature-complete
- Browser installation (`scrutineer browsers install`) is fully implemented
- The YAML schema is stable and well-documented
- Load testing and fuzz testing are production-ready
- The project has been used in real-world testing scenarios

## Release Process

### 1. Ensure All Checks Pass

Before releasing, all pre-push checks must pass:

```bash
make precommit
```

This runs: formatting, vet, vulnerability scanning, tests, and coverage gate (98%).

### 2. Update the Version

The version is derived from Git tags. Create a new annotated tag:

```bash
git tag -a v0.0.2 -m "Release v0.0.2: brief description"
```

### 3. Build Binaries

```bash
make cross
```

This produces binaries for all six supported platforms in the `bin/` directory.

### 4. Push the Tag

```bash
git push origin v0.0.2
```

### 5. Create a Release

Create a release on GitHub with the tag, attaching the cross-compiled binaries.

## Changelog Expectations

Each release should include a changelog entry that:

- Lists all user-visible changes, grouped by category (Added, Changed, Fixed, Removed)
- References relevant commits or pull requests
- Calls out any breaking changes prominently at the top
- Uses clear, non-technical language where possible (users read changelogs, not just developers)

### Changelog Format

```markdown
## v0.0.2 (2026-05-15)

### Added
- New `between` assertion operator for numeric range checks
- SSH tunnel support in the SSH connector

### Fixed
- HTTP connector now properly closes idle connections on teardown
- YAML parser handles multi-line strings correctly

### Changed
- Default timeout increased from 10s to 30s
```

### What Not to Include

- Internal refactoring that does not change behavior
- Test-only changes
- Documentation updates (unless they reflect a feature change)
- Dependency updates (unless they fix a security vulnerability)

## Git Tagging Convention

- Tags follow the format `v<MAJOR>.<MINOR>.<PATCH>` (e.g., `v0.0.1`, `v1.2.3`)
- Use annotated tags (`git tag -a`) with a message describing the release
- Pre-release versions use a suffix: `v0.0.1-rc.1`, `v0.0.1-beta.2`
- Development builds show the Git description: `v0.0.1-3-gabcdef` (3 commits after v0.0.1, at commit abcdef)
- Dirty working trees are marked: `v0.0.1-dirty`

## Cross-Platform Build Matrix

Every release must build cleanly for all six targets:

| GOOS    | GOARCH | Binary Name                    |
|---------|--------|--------------------------------|
| linux   | amd64  | `scrutineer-linux-amd64`       |
| linux   | arm64  | `scrutineer-linux-arm64`       |
| darwin  | amd64  | `scrutineer-darwin-amd64`      |
| darwin  | arm64  | `scrutineer-darwin-arm64`      |
| windows | amd64  | `scrutineer-windows-amd64.exe` |
| windows | arm64  | `scrutineer-windows-arm64.exe` |

## Next Steps

- [Developer Guide](development.md) -- development setup and conventions
- [Architecture Overview](../architecture/overview.md) -- system design
