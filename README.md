<h1 align="center">javaup</h1>

<p align="center">English | <a href="README.zh-CN.md">简体中文</a></p>

<p align="center"><strong>Let every Maven project use the right Java toolchain automatically.</strong></p>

<p align="center">
  <a href="https://github.com/codeboyzhou/javaup/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/codeboyzhou/javaup"></a>
  <img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/codeboyzhou/javaup">
  <a href="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml/badge.svg"></a>
  <img alt="Platform" src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-blue">
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/github/license/codeboyzhou/javaup"></a>
</p>

`javaup` (command: `jup`) is a project-aware Java toolchain manager for Maven
builds. It detects the Java version a project needs, selects a matching local
JDK, and remembers the Maven executable, JDK, and optional `settings.xml` that
belong together. Every build then runs with the saved toolchain in a fresh child
process, without changing `JAVA_HOME`, `PATH`, or any other setting in your
current shell.

<p align="center">
  <img src="docs/demo/demo.gif" alt="javaup demo">
</p>

## Why javaup?

A single development machine often needs to build projects targeting Java 8,
Java 17, Java 21, and beyond. Without `jup`, every switch means reconstructing
the right environment: choosing a JDK, finding the expected Maven installation,
and sometimes supplying a different `settings.xml`. That knowledge is easy to
lose and tedious to repeat.

| Task                     | Without `jup`                                      | With `jup`                                                    |
|--------------------------|----------------------------------------------------|---------------------------------------------------------------|
| Switch projects          | Edit `JAVA_HOME` and `PATH`                        | Reuse the JDK saved for that project                          |
| Select Maven             | Depend on whichever `mvn` is on PATH               | Prefer Maven Wrapper or reuse the saved Maven executable      |
| Use private repositories | Repeat `--settings` or replace a global file       | Apply the project's saved settings alias automatically        |
| Build from anywhere      | Change directories and reconstruct the environment | Select an initialized project and load its complete toolchain |
| Preserve the shell       | Risk affecting later commands                      | Keep build-specific changes inside the child process          |

`jup` complements existing Java tools instead of replacing them:

- **SDKMAN! and asdf** install or switch tools for a user or shell. `jup` can
  discover their JDKs, as well as JDKs exposed through environment variables
  and PATH.
- **Maven Wrapper** pins the Maven distribution for a repository. `jup` detects
  and prefers it automatically.
- **Maven Toolchains** lets Maven plugins select a JDK. `jup` also discovers
  `<jdkHome>` entries and controls the JDK that launches Maven itself.

The layer `jup` adds is a durable local project binding: **this Maven executable
+ this JDK + this settings alias**, launched through one stable command from any
terminal.

> [!IMPORTANT]
> `jup` selects JDKs and Maven installations that are already present. It does
> not download or uninstall them. Apache Maven is currently the only supported
> build tool.

## Install

macOS or Linux:

```shell
curl -fsSL https://github.com/codeboyzhou/javaup/releases/latest/download/install.sh | sh
```

Windows PowerShell 5.1 or later:

```powershell
irm https://github.com/codeboyzhou/javaup/releases/latest/download/install.ps1 | iex
```

Prebuilt releases support Windows, macOS, and Linux on amd64 and arm64. See the
[installation guide](https://codeboyzhou.github.io/javaup/start-here/installation/)
for checksums, `go install`, source builds, and installer options.

## Quick Start

All you need is a project with `pom.xml`, Maven Wrapper or `mvn` on PATH, and a
matching full JDK installed locally:

```shell
cd /path/to/your/maven-project
jup init
jup status
jup run mvn clean package
```

In an interactive terminal, `jup run mvn` lets you select any initialized
project, with frequently and recently used projects ranked first. In CI or
redirected pipelines, it resolves the nearest initialized project without
prompting.

## What It Does

- Detects Java requirements from POM properties, compiler plugin configuration,
  and local parent POMs.
- Prefers Maven Wrapper and falls back to Maven from PATH.
- Finds installed JDKs through Maven, SDKMAN!, asdf, environment variables,
  PATH, Maven Toolchains, and common platform locations.
- Applies the selected JDK and optional Maven settings alias only to the spawned
  build process.
- Manages and diagnoses initialized projects globally without modifying project
  source files.

## Contributing

Bug reports, compatibility cases, documentation improvements, and code
contributions are welcome. Run the complete local verification pipeline with:

```shell
go mod download
go run build.go verify
```

Read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting a pull request.

## License

Licensed under the [Apache License 2.0](LICENSE).
