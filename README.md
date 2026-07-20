<h1 align="center">javaup — Project-aware Java Toolchain Manager</h1>

<p align="center">English | <a href="README.zh-CN.md">简体中文</a></p>

<p align="center">
  <img alt="Go Version" src="https://img.shields.io/github/go-mod/go-version/codeboyzhou/javaup">
  <img alt="Platform" src="https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-blue">
  <a href="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml/badge.svg"></a>
</p>

<p align="center"><strong>The right JDK, the right Maven, the right <code>settings.xml</code>, automatically, for every project build.</strong></p>

<p align="center">
  <a href="#why-javaup-created">Why javaup created</a> |
  <a href="#highlights">Highlights</a> |
  <a href="#installation">Installation</a> |
  <a href="#quick-start">Quick Start</a> |
  <a href="#command-guide">Command Guide</a>
</p>

`javaup` (command: `jup`) detects the Java version required by a Maven project,
locates a matching local JDK, and remembers its Maven, JDK, and `settings.xml`. Configure once, run anytime — without modifying the current shell or your system env.

## Why javaup created

A development machine often hosts projects from several Java generations: a
legacy application may still require Java 8, a current service may use Java 17,
and another repository may already have moved to Java 21. Without project-level
toolchain management, developers repeatedly edit `JAVA_HOME` and `PATH`, keep
track of which Maven belongs to which project, or depend on IDE settings that
do not carry over to the terminal.

`jup` turns those error-prone manual steps into one initialization command and
one stable build entry point.

| Scenario             | Without `jup`                                               | With `jup`                                                 |
|----------------------|-------------------------------------------------------------|------------------------------------------------------------|
| Switching projects   | Edit `JAVA_HOME` and `PATH` manually                        | Use the JDK saved for the project                          |
| Selecting Maven      | Depend on the current PATH and risk using the wrong version | Prefer the project Wrapper, otherwise save Maven from PATH |
| IDE versus terminal  | Maintain two configurations that may drift apart            | Use an explicit, inspectable terminal toolchain            |
| Private repositories | Repeat `--settings` or replace the global file              | Bind a named `settings.xml` to each project                |
| Working in modules   | Return to the root or resolve the environment manually      | Find the initialized project from any descendant directory |
| Environment impact   | Mutate the shell and affect later commands                  | Change only the spawned build process                      |

`jup` does not download or install JDKs or Maven, and it does not modify
`JAVA_HOME` in the current shell. It discovers existing local toolchains,
stores a project-specific selection, and isolates the environment used by the
build process. **Apache Maven** is currently the supported build tool.

## Highlights

- Detects the Java release from `pom.xml`, local parent POMs, and
  `maven-compiler-plugin` configuration.
- Detects `mvnw` / `mvnw.cmd`; falls back to Maven from PATH when no Wrapper is
  present.
- Discovers JDKs through environment variables, PATH, Maven Toolchains, common
  installation locations, and sibling directories of known JDKs.
- Persists the Maven executable, JDK path, versions, and initialization time for
  each project.
- Sets the correct `JAVA_HOME` and places the selected JDK's `bin` first on PATH
  for the build child process.
- Assigns reusable aliases to multiple Maven `settings.xml` files and binds an
  alias per project.
- Supports `status`, `run`, `settings use/unset`, and `uninit` from any directory
  below the project root.
- Runs on Windows, macOS, and Linux, with CI verification on all three platforms.

## Installation

### Install on macOS or Linux

Run in a terminal:

```shell
curl -fsSL https://github.com/codeboyzhou/javaup/releases/latest/download/install.sh | sh
```

The installer detects the operating system and architecture, verifies the
downloaded release's SHA-256 checksum, installs `jup` under `~/.javaup/bin`,
and updates the appropriate shell profile. Set `JAVAUP_VERSION`, `JAVAUP_HOME`,
or `JAVAUP_NO_MODIFY_PATH` to customize the installation.

### Install on Windows

Run in PowerShell 5.1 or later:

```powershell
irm https://github.com/codeboyzhou/javaup/releases/latest/download/install.ps1 | iex
```

The installer downloads the latest Windows release, verifies its SHA-256
checksum, installs `jup.exe` under `%USERPROFILE%\.javaup\bin`, and adds that
directory to the user PATH. Set `JAVAUP_VERSION`, `JAVAUP_HOME`, or
`JAVAUP_NO_MODIFY_PATH` before running the command to customize the installation.

### Install with Go

Use the Go version declared in [`go.mod`](go.mod), or a newer version:

```shell
go install github.com/codeboyzhou/javaup/cmd/jup@latest
```

Make sure the Go binary directory (usually `$GOBIN` or `$GOPATH/bin`) is on
PATH, then verify the installation:

```shell
jup version
```

### Build from source

```shell
git clone https://github.com/codeboyzhou/javaup.git
cd javaup
go run build.go
```

The artifact is written to:

```text
Windows: dist/jup.exe
macOS:   dist/jup
Linux:   dist/jup
```

Copy the artifact to any directory on PATH.

### Runtime prerequisites

- The project root contains a `pom.xml`.
- The project contains Maven Wrapper, or `mvn` is available on PATH.
- A full JDK matching the project's Java version is already installed. A JRE
  alone is not sufficient for discovery.

## Quick start

Enter a Maven project and initialize it:

```shell
cd /path/to/project
jup init
```

Initialization runs five stages:

```text
[1/5] Project      - resolve the project root
[2/5] Build Tool   - detect Maven, its version, and Wrapper
[3/5] Java Version - read the required Java version from the POM
[4/5] JDK          - locate a matching local JDK
[5/5] Config       - persist the project toolchain
```

Inspect the result:

```shell
jup status
```

Example output:

```text
Project: E:\code\demo
Build tool: Maven 3.9.16 (PATH)
Build executable: D:\tools\maven\bin\mvn.cmd
Java version: 1.8.0_472
Java home: D:\OpenJDK8
Maven settings: default
```

Run a build with the saved toolchain:

```shell
jup run mvn clean package
```

`jup` connects Maven's standard input, output, and error streams directly to
the current terminal, preserving interactive behavior, logs, and exit codes.

## Command guide

### `jup init`

Detect and save the toolchain for the current Maven project:

```shell
jup init
```

Running `init` again refreshes the Maven and JDK selection while preserving an
existing Maven settings alias. Project paths, symbolic links, and Windows
long-path/8.3-path variants are normalized so that one project does not produce
multiple configurations.

#### How the Java version is detected

`jup` evaluates these POM values in order and resolves `${property}` references:

1. `<release>` in `maven-compiler-plugin`;
2. `maven.compiler.release`;
3. `<target>` in `maven-compiler-plugin`;
4. `maven.compiler.target`;
5. `<source>` in `maven-compiler-plugin`;
6. `maven.compiler.source`;
7. `java.version`;
8. `jdk.version`.

When a local parent POM exists, `jup` follows `<relativePath>` and resolves up
to 16 parent levels.

#### How local JDKs are discovered

`jup` checks full JDK installations in candidate order and selects the first
one whose Java major version matches the project:

1. the JDK reported by the Maven runtime, when available;
2. `JAVA_HOME`, `JDK_HOME`, and versioned variables such as `JAVA8_HOME` or
   `JAVA_HOME_17`;
3. the JDK that owns `javac` on PATH;
4. sibling directories of known JDKs whose names resemble common JDK
   distributions;
5. `<jdkHome>` entries in `~/.m2/toolchains.xml`;
6. platform-specific locations such as `Program Files/Java`, `~/.jdks`,
   SDKMAN, Homebrew, and asdf.

A candidate must contain `bin/javac` (`bin/javac.exe` on Windows). The version
is read from the JDK `release` file when possible, with `javac -version` as a
fallback.

### `jup status`

Show the saved toolchain for the project containing the current directory:

```shell
jup status
```

The output includes the project root, Maven version and source, Maven
executable, Java version, JDK path, and Maven settings alias. The command works
from any directory below the project root.

### `jup run mvn`

Run Maven with the executable and JDK saved during initialization:

```shell
jup run mvn test
jup run mvn clean package -DskipTests
jup run mvn dependency:tree
```

For every invocation, `jup`:

1. searches upward from the current directory for the nearest initialized
   project;
2. checks that the saved Maven executable still exists;
3. sets the saved `JAVA_HOME` for the child process;
4. places that JDK's `bin` first on the child process PATH;
5. prepends `--settings <path>` when the project uses a settings alias;
6. starts Maven in the current directory rather than forcing the project root.

The current shell environment is not modified.

### `jup settings`

Assign reusable aliases to Maven `settings.xml` files. This is useful for
switching between a company repository, public mirrors, and environments with
different credentials.

Add or update an alias:

```shell
jup settings add intranet D:\maven\settings-intranet.xml
jup settings add public D:\maven\settings-public.xml
```

`jup` checks that the path exists, is a regular file, contains valid XML, and
has `<settings>` as its root element. Only the normalized file path is saved;
the contents of `settings.xml` are not copied.

List aliases:

```shell
jup settings list
```

Bind the current project to an alias:

```shell
jup settings use intranet
jup run mvn clean deploy
```

Unbind the current project without removing the global alias:

```shell
jup settings unset
```

Remove a global alias:

```shell
jup settings remove intranet
```

`unset` changes the project selection; `remove` deletes the global alias. If an
alias is removed while a project still references it, the next build reports
that the alias is missing. Bind another alias or run `settings unset` to recover.

### `jup uninit`

Remove the local `jup` configuration for the project containing the current
directory:

```shell
jup uninit
```

The command searches upward for the nearest initialized project. It is
idempotent: if no configuration exists, it reports that there is nothing to
remove. Project files, JDKs, Maven installations, and Maven settings files are
never modified.

### Help and version

```shell
jup --help
jup <command> --help
jup version
jup --version
```

Version output contains the semantic version, target platform, and abbreviated
Git revision used for the build:

```text
javaup version v0.1.0 windows/amd64 (64c2fb07bcad)
```

## Configuration storage

Project configurations and Maven settings aliases live under `JAVAUP_HOME` and
are not written into project repositories. Release installers also place the
executable in its `bin` directory. The default is `.javaup` in the current
user's home directory:

| Platform | Default `JAVAUP_HOME`   |
|----------|-------------------------|
| Windows  | `%USERPROFILE%\.javaup` |
| macOS    | `$HOME/.javaup`         |
| Linux    | `$HOME/.javaup`         |

Directory layout:

```text
.javaup/
├── bin/
│   └── jup                # jup.exe on Windows
└── config/
    ├── projects/          # one JSON document per initialized project
    └── maven/
        └── settings.json  # Maven settings alias registry
```

Set `JAVAUP_HOME` to an absolute path before installing or running `jup` to
use another location.

Project configurations contain absolute path snapshots. Run `jup init` again
after moving or deleting a JDK, Maven Wrapper, or Maven installation.

## Troubleshooting

### Maven is installed, but `mvn` is not found on PATH

Check Maven in the same terminal:

```shell
mvn --version
```

Restart the terminal after changing environment variables. An IDE terminal
often requires restarting the entire IDE so that its parent process reloads
PATH. If global Maven is unavailable, add Maven Wrapper to the project instead.

### A JDK is installed, but `jup` cannot find it

Make sure it is a JDK containing `javac`, not a JRE. For an arbitrary custom
location, expose a versioned variable while keeping the default `JAVA_HOME`:

```powershell
$env:JAVA8_HOME = "D:\OpenJDK8"
jup init
```

Alternatively, configure `<jdkHome>` in Maven's `~/.m2/toolchains.xml`.

### A saved Maven or JDK path no longer exists

Refresh the project configuration:

```shell
jup init
```

### Disable colored output

Set the standard `NO_COLOR` environment variable when running `jup`. The build
script additionally supports:

```shell
JUP_BUILD_COLOR=always go run build.go
JUP_BUILD_COLOR=never go run build.go
```

## Contributing

### Development environment

1. Install the Go version declared in [`go.mod`](go.mod).
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

### Recommended workflow

```shell
# Run all unit tests
go test ./...

# Stress the package affected by a change
go test ./internal/javainfo -count=5

# Run complete verification and produce a local artifact before committing
go run build.go
```

The build stops at the first failed stage. CI runs `go run build.go verify` on
Ubuntu, Windows, and macOS.

Commit messages follow Conventional Commits:

```text
feat(java): discover sibling jdk installations
fix(maven): handle missing executable in path
docs(readme): rewrite project documentation
```

When opening a pull request, describe the problem, design trade-offs,
verification commands, and real-world results for platform-specific changes.

### Project layout

```text
build.go                    # local verification and build pipeline
cmd/jup/                    # CLI entry point
internal/buildinfo/         # version and build metadata
internal/buildtool/maven/   # Maven, Wrapper, and POM detection
internal/cli/               # Cobra commands and terminal output
internal/javainfo/          # JDK discovery and version matching
internal/mavensettings/     # Maven settings alias storage
internal/project/           # initialization, configuration, status, execution
golangci-lint.mod           # pinned lint tool dependencies
govulncheck.mod             # pinned vulnerability scanner dependencies
```

The command layer handles argument parsing and presentation. Reusable business
logic lives in `internal` packages and is injected through interfaces for
testability. New behavior should include cross-platform tests whenever
possible; platform-specific behavior is isolated with Go build tags.

### Inject release metadata

Release builds can inject a version and revision with ldflags:

```shell
go build \
  -ldflags "-X github.com/codeboyzhou/javaup/internal/buildinfo.Version=v1.0.0 -X github.com/codeboyzhou/javaup/internal/buildinfo.Commit=<commit-hash>" \
  -o jup ./cmd/jup
```

When the revision is not injected explicitly, `jup` reads the VCS revision from
Go build information.

## Current scope

- Maven is currently the only supported build tool; Gradle is not yet supported.
- `jup` discovers and selects existing JDKs. It does not download, upgrade, or
  uninstall them.
- Project configurations are local to the current user and are not shared
  through the repository.
- JDK and Maven locations are stored as absolute paths; run `jup init` after a
  tool is moved.
- The Maven settings registry stores file paths, not file contents or credentials.

## License

Licensed under the [Apache License 2.0](LICENSE).
