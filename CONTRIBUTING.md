# Contributing to javaup

English | [简体中文](CONTRIBUTING.zh-CN.md) | [Back to README](README.md)

Bug reports, compatibility cases, documentation improvements, and code
contributions are welcome.

## Report a problem

Before opening an issue, search existing issues and review the
[troubleshooting guide](docs/user-guide.md#troubleshooting). Include enough
information to reproduce the behavior:

- operating system and architecture;
- `jup version` output;
- relevant POM properties, compiler plugin configuration, and parent layout;
- expected and detected Java versions;
- command output, with credentials and private paths redacted as needed.

## Development environment

1. Install the Go version declared in [`go.mod`](go.mod), or newer.
2. Fork and clone the repository.
3. Download dependencies and run verification from the repository root:

```shell
go mod download
go run build.go verify
```

Verification runs these stages in order:

```text
gofmt -l .
go vet ./...
go tool -modfile=golangci-lint.mod golangci-lint run
go test ./...
go tool -modfile=govulncheck.mod govulncheck ./...
```

GolangCI-Lint and govulncheck are pinned in isolated Go module files. They do
not require global installation and do not pollute application dependencies.

## Recommended workflow

Run all unit tests:

```shell
go test ./...
```

Stress the package affected by a change when concurrency or platform discovery
is involved:

```shell
go test ./internal/javainfo -count=5
```

Run complete verification and produce a local artifact before committing:

```shell
go run build.go
```

The build stops at the first failed stage. CI runs `go run build.go verify` on
Ubuntu, Windows, and macOS.

## Design and testing expectations

- Keep argument parsing and presentation in `internal/cli`.
- Put reusable behavior in the relevant `internal` package and inject external
  operations through interfaces where practical.
- Add tests for new behavior and regressions.
- Prefer cross-platform tests. Isolate unavoidable platform-specific behavior
  with Go build tags.
- Preserve the current shell environment; build-specific changes belong in the
  spawned child process.
- Never copy Maven settings contents or credentials into javaup configuration.

## Project layout

```text
build.go                    # local verification and build pipeline
cmd/jup/                    # CLI entry point
internal/apphome/           # JAVAUP_HOME resolution
internal/buildinfo/         # version and build metadata
internal/buildtool/maven/   # Maven, Wrapper, and POM detection
internal/cli/               # Cobra commands and terminal output
internal/javainfo/          # JDK discovery and version matching
internal/mavensettings/     # Maven settings alias storage
internal/project/           # initialization, configuration, status, execution
golangci-lint.mod           # pinned lint tool dependencies
govulncheck.mod             # pinned vulnerability scanner dependencies
```

## Commits and pull requests

Commit messages follow Conventional Commits:

```text
feat(java): discover sibling jdk installations
fix(maven): handle missing executable in path
docs(readme): improve quick start
```

Use an English, lowercase, imperative subject of no more than 72 characters,
with no trailing period. Supported types are `feat`, `fix`, `docs`, `style`,
`refactor`, `perf`, `test`, `build`, `ci`, and `chore`.

In a pull request, describe:

- the problem and intended behavior;
- important design trade-offs;
- verification commands and results;
- real-world results for platform-specific or tool-discovery changes.

Keep pull requests focused. Separate unrelated refactors or documentation
changes when they make the main change harder to review.

## Release metadata

Release builds inject a semantic version and Git revision with ldflags:

```shell
go build \
  -ldflags "-X github.com/codeboyzhou/javaup/internal/buildinfo.Version=v1.0.0 -X github.com/codeboyzhou/javaup/internal/buildinfo.Commit=<commit-hash>" \
  -o jup ./cmd/jup
```

When the revision is not injected explicitly, `jup` reads the VCS revision from
Go build information.

## License

By contributing, you agree that your contribution will be licensed under the
[Apache License 2.0](LICENSE).
