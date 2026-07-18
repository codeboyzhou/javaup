# javaup

`jup` is a command-line tool for managing Java versions.

## Development

Build the CLI:

```shell
go build -o jup ./cmd/jup
```

Run the test suite:

```shell
go test ./...
```

Inject a release version at build time:

```shell
go build -ldflags "-X github.com/codeboyzhou/javaup/internal/buildinfo.Version=v1.0.0 -X github.com/codeboyzhou/javaup/internal/buildinfo.Commit=<commit-hash>" -o jup ./cmd/jup
```

When `Commit` is not explicitly injected, `jup` uses the VCS revision embedded
by the Go toolchain. Version output includes its first 12 characters:

```text
javaup version v0.1.0 windows/amd64 (64c2fb07bcad)
```

## Commands

```text
jup help [command]
jup version
```

The standard `--help`, `-h`, `--version`, and `-v` flags are also supported.

## Project structure

```text
cmd/jup/            executable entry point
internal/buildinfo/ build-time version metadata
internal/cli/       Cobra command tree and built-in commands
```

Each command has its own constructor in `internal/cli` and is registered by the
root command. Command handlers should only parse input and render output;
reusable business logic belongs in a separate package under `internal` and is
passed into command constructors as a dependency.
