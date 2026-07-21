<h1 align="center">javaup</h1>

<p align="center"><strong>Project-aware Java toolchains for Maven builds</strong></p>

<p align="center">English | <a href="README.zh-CN.md">简体中文</a></p>

<p align="center">
  <a href="https://github.com/codeboyzhou/javaup/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/codeboyzhou/javaup"></a>
  <img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/codeboyzhou/javaup">
  <a href="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml/badge.svg"></a>
  <img alt="Platform" src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-blue">
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/github/license/codeboyzhou/javaup"></a>
</p>

<p align="center">
  <strong>The right JDK, the right Maven, and the right <code>settings.xml</code> for every project build.</strong>
</p>

`javaup` (command: `jup`) detects the Java version required by a Maven project,
finds a matching installed JDK, and remembers the Maven executable, JDK, and
optional `settings.xml` selection for that project. It recreates the saved
toolchain whenever you build, without changing `JAVA_HOME` or `PATH` in your
current shell.

```console
$ cd myproject
$ jup init
[1/5] Project OK - /work/myproject
[2/5] Build Tool OK - Maven 3.9.11 (wrapper)
[3/5] Java Version OK - Java 8
[4/5] JDK OK - Java 1.8.0_472 at /opt/jdks/temurin-8
[5/5] Config OK - /home/alex/.javaup/config/projects/...
Initialized javaup project.

$ jup run mvn test
[INFO] BUILD SUCCESS
```

> [!IMPORTANT]
> `jup` selects toolchains that are already installed. It does not download or
> uninstall JDKs or Maven. Apache Maven is the only supported build tool in
> v0.1.0.

## Why javaup?

A single development machine often hosts projects from several Java
generations: a legacy application on Java 8, a current service on Java 17, and
a newer repository on Java 21. Switching between them usually means changing
environment variables, remembering which Maven installation to use, or relying
on IDE settings that do not follow you into the terminal.

With `jup`, each project gets one explicit, inspectable toolchain:

| Task | Without `jup` | With `jup` |
|---|---|---|
| Switch projects | Edit `JAVA_HOME` and `PATH` | Use the JDK saved for the project |
| Select Maven | Depend on whatever is on PATH | Prefer the project Wrapper, otherwise save Maven from PATH |
| Use private repositories | Repeat `--settings` or replace a global file | Bind a named `settings.xml` to the project |
| Build from a submodule | Return to the root and reconstruct the environment | Resolve the initialized project from any descendant directory |
| Preserve the shell | Risk affecting later commands | Change only the spawned build process |

### How it fits with existing tools

`javaup` complements JDK version managers and Maven's own tooling:

- **SDKMAN!, asdf, and jEnv** install or switch tools for a user or shell. `jup`
  does not replace them; it can discover the JDKs they installed.
- **Maven Wrapper** pins the Maven distribution for a repository. `jup` detects
  and prefers the Wrapper automatically.
- **Maven Toolchains** lets Maven plugins select a JDK. `jup` can discover
  `<jdkHome>` entries from `~/.m2/toolchains.xml` and also controls the JDK that
  launches Maven itself.

The useful layer `jup` adds is a local project binding: **this Maven executable
+ this JDK + this settings alias**, launched through one stable command.

## Install

### macOS or Linux

```shell
curl -fsSL https://github.com/codeboyzhou/javaup/releases/latest/download/install.sh | sh
```

The installer detects the operating system and architecture, verifies the
release checksum, installs `jup` under `~/.javaup/bin`, and updates the relevant
shell profile.

### Windows

Run in PowerShell 5.1 or later:

```powershell
irm https://github.com/codeboyzhou/javaup/releases/latest/download/install.ps1 | iex
```

The installer verifies the release checksum, installs `jup.exe` under
`%USERPROFILE%\.javaup\bin`, and adds that directory to the user PATH.

Prefer to inspect the files first? Download an archive, checksum, or installer
from [GitHub Releases](https://github.com/codeboyzhou/javaup/releases/latest).
Prebuilt binaries are available for Windows, macOS, and Linux on amd64 and
arm64.

<details>
<summary>Other installation options</summary>

Install with the Go version declared in [`go.mod`](go.mod), or newer:

```shell
go install github.com/codeboyzhou/javaup/cmd/jup@latest
```

Or build from source:

```shell
git clone https://github.com/codeboyzhou/javaup.git
cd javaup
go run build.go
```

The binary is written to `dist/jup.exe` on Windows or `dist/jup` on macOS and
Linux.

</details>

The installers support these optional environment variables:

| Variable | Purpose |
|---|---|
| `JAVAUP_VERSION` | Install a specific release, such as `v0.1.0` |
| `JAVAUP_HOME` | Use a custom absolute installation and configuration directory |
| `JAVAUP_NO_MODIFY_PATH` | Install without updating the shell profile or user PATH |

Verify the installation:

```shell
jup version
```

## Quick start

Before starting, the Maven project must have a `pom.xml`, a Maven Wrapper or
`mvn` on PATH, and an installed full JDK matching the project's Java version.

Initialize, inspect, and build:

```shell
cd /path/to/your/maven-project
jup init
jup status
jup run mvn clean package
```

Example status:

```text
Project: /work/demo
Build tool: Maven 3.9.11 (wrapper)
Build executable: /work/demo/mvnw
Java version: 17.0.12
Java home: /opt/jdks/temurin-17
Maven settings: default
```

`jup run mvn` connects Maven directly to the current terminal, preserving
interactive input, logs, and exit codes. It starts Maven in the current
directory, so module-specific builds work as expected.

## Highlights

- Detects the Java release from `pom.xml`, compiler plugin configuration,
  properties, and local parent POMs.
- Detects `mvnw` / `mvnw.cmd` and falls back to Maven from PATH.
- Finds installed JDKs through Maven, environment variables, PATH, Maven
  Toolchains, common installation locations, and sibling JDK directories.
- Saves the Maven executable, JDK path, versions, and initialization time per
  project.
- Sets `JAVA_HOME` and puts the selected JDK's `bin` first on PATH only for the
  spawned build process.
- Binds reusable Maven `settings.xml` aliases to individual projects.
- Resolves `status`, `run`, `settings use/unset`, and `uninit` from any
  descendant directory.
- Runs on Windows, macOS, and Linux, with CI verification on all three.

## Maven settings aliases

Register settings files once, then select one per project:

```shell
jup settings add intranet /path/to/settings-intranet.xml
jup settings add public /path/to/settings-public.xml

cd /path/to/company-project
jup settings use intranet
jup run mvn clean deploy
```

`jup` saves only the normalized file path. It does not copy the XML file or its
credentials. Use `jup settings unset` to return a project to Maven's default
settings.

## Command overview

| Command | Purpose |
|---|---|
| `jup init` | Detect and save the current project's Maven and JDK |
| `jup status` | Show the saved toolchain |
| `jup run mvn <args...>` | Run Maven with the saved toolchain |
| `jup settings add <alias> <file>` | Register or update a settings alias |
| `jup settings list` | List settings aliases |
| `jup settings use <alias>` | Bind an alias to the current project |
| `jup settings unset` | Remove the project's settings binding |
| `jup settings remove <alias>` | Remove a global alias |
| `jup uninit` | Remove the project's saved `jup` configuration |
| `jup uninstall` | Uninstall jup while preserving configuration |
| `jup uninstall --purge` | Uninstall jup and remove all javaup data |
| `jup update --check` | Check whether a newer release is available |
| `jup update` | Download, verify, and install the latest release |

See the [full user guide](docs/user-guide.md) for detection rules,
configuration storage, detailed command behavior, and troubleshooting.

## Project status and scope

v0.1.0 is the first public release of `javaup`. Release archives are built for
Windows, macOS, and Linux on amd64 and arm64, and every archive is covered by a
published SHA-256 checksum.

Current boundaries:

- Maven is the only supported build tool; Gradle is not yet supported.
- JDKs and Maven must already be installed locally.
- Project configuration is local to the current user and is not written to the
  repository.
- Saved JDK and Maven locations are absolute paths; run `jup init` again after
  moving a tool or project.
- Maven settings aliases store paths, not file contents or credentials.

If `javaup` cannot handle a real project, please open an
[issue](https://github.com/codeboyzhou/javaup/issues) with the operating system,
POM structure, expected Java version, and relevant command output.

## Documentation

- [User guide](docs/user-guide.md) — commands, detection, storage, and troubleshooting
- [Contributing guide](CONTRIBUTING.md) — development setup, verification, and project layout
- [Simplified Chinese README](README.zh-CN.md)
- [Simplified Chinese user guide](docs/user-guide.zh-CN.md)

## Contributing

Bug reports, compatibility cases, documentation improvements, and code
contributions are welcome. To run the complete local verification pipeline:

```shell
go mod download
go run build.go verify
```

Read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting a pull request.

## License

Licensed under the [Apache License 2.0](LICENSE).
