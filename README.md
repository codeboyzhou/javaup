# javaup

`jup` is a command-line tool for managing Java versions.

## Development

Run the complete local build pipeline:

```shell
go run build.go
```

The script stops on the first failure and runs these stages in order:

```text
go fmt ./...
go vet ./...
go tool -modfile=golangci-lint.mod golangci-lint run
go test ./...
go tool -modfile=govulncheck.mod govulncheck ./...
go build -trimpath -o dist/<binary> ./cmd/jup
```

GolangCI-Lint and govulncheck are pinned in isolated Go module files, so no
global tool installation is required and their dependencies do not affect the
application module. Linter selection is defined in `.golangci.yml`. The
resulting executable is written to `dist`.

Build output uses colors when stdout is an interactive terminal. Set
`JUP_BUILD_COLOR=always` or `JUP_BUILD_COLOR=never` to override detection;
setting `NO_COLOR` also disables colors in automatic mode. Color rendering is
provided by `github.com/fatih/color` for cross-platform terminal support.

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
jup init
jup uninit
jup help [command]
jup version
```

`jup init` currently detects Maven projects, their Maven or Maven Wrapper
version, the Java build version, and the matching local JDK. Project metadata is
stored as JSON under the platform-specific user configuration directory:

```text
Windows: %AppData%\javaup\projects
macOS:   ~/Library/Application Support/javaup/projects
Linux:   $XDG_CONFIG_HOME/javaup/projects (or ~/.config/javaup/projects)
```

Initialization reports each detection stage with cross-platform colored output
when running in an interactive terminal. Setting `NO_COLOR` disables colors.
The `initializedAt` value uses the local `YYYY-MM-DD HH:mm:ss` format.

`jup uninit` removes the saved configuration for the current project. Repeated
execution is safe when the project has already been uninitialized.

The standard `--help`, `-h`, `--version`, and `-v` flags are also supported.

## Project structure

```text
.golangci.yml       lint policy and enabled analyzers
build.go            local verification and build pipeline
cmd/jup/            executable entry point
golangci-lint.mod   isolated GolangCI-Lint dependencies
govulncheck.mod     isolated govulncheck dependencies
internal/buildinfo/ build-time version metadata
internal/cli/       Cobra command tree and built-in commands
```

Each command has its own constructor in `internal/cli` and is registered by the
root command. Command handlers should only parse input and render output;
reusable business logic belongs in a separate package under `internal` and is
passed into command constructors as a dependency.
