<h1 align="center">javaup</h1>

<p align="center">English | <a href="README.zh-CN.md">简体中文</a></p>

<p align="center"><strong>Use the correct Java toolchain for every Maven project — automatically.</strong></p>

<p align="center">
  <a href="https://github.com/codeboyzhou/javaup/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/codeboyzhou/javaup"></a>
  <img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/codeboyzhou/javaup">
  <a href="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml/badge.svg"></a>
  <img alt="Platform" src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-blue">
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/github/license/codeboyzhou/javaup"></a>
</p>

`javaup` (command: `jup`) is a Java toolchain manager built for Maven projects.
It detects the Java version required by a project, selects a matching installed
JDK, and remembers the project's Maven installation path, JDK, and optional
`settings.xml`. Every subsequent build reuses this toolchain, so there is no need
to change `JAVA_HOME` or `PATH` in the current shell or manually switch between
versions.

<p align="center">
  <img src="docs/demo/demo.gif" alt="javaup demo">
</p>

> [!IMPORTANT]
> `jup` selects JDKs and Maven installations that are already present. It does
> not download or uninstall them. Apache Maven is the only supported build tool
> in v0.3.0.

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
[installation guide](https://codeboyzhou.github.io/javaup/getting-started/installation/)
for checksums, `go install`, source builds, and installer options.

## Quick Start

The Maven project needs a `pom.xml`, Maven Wrapper or `mvn` on PATH, and a
matching full JDK installed locally.

```shell
cd /path/to/your/maven-project
jup init
jup status
jup run mvn clean package
```

In an interactive terminal, `jup run mvn` can select any initialized project
and ranks frequently and recently used projects first. In CI or redirected
pipelines, it resolves the nearest initialized project without prompting.

## What It Does

- Detects Java requirements from POM properties, compiler plugin configuration,
  and local parent POMs.
- Prefers Maven Wrapper and falls back to Maven from PATH.
- Finds installed JDKs through Maven, environment variables, PATH, Maven
  Toolchains, and common platform locations.
- Applies the selected JDK and optional Maven settings alias only to the spawned
  build process.
- Manages and diagnoses initialized projects globally without modifying project
  source files.

See [Why javaup?](https://codeboyzhou.github.io/javaup/#why-javaup) for a
detailed comparison with manual environment switching, SDKMAN!, asdf, jEnv,
Maven Wrapper, and Maven Toolchains.

## Documentation

- [Documentation site](https://codeboyzhou.github.io/javaup/)
- [Quick start](https://codeboyzhou.github.io/javaup/getting-started/quick-start/)
- [User guide](https://codeboyzhou.github.io/javaup/user-guide/)
- [Command reference](https://codeboyzhou.github.io/javaup/reference/command-reference/)
- [Troubleshooting](https://codeboyzhou.github.io/javaup/reference/troubleshooting/)
- [简体中文文档](https://codeboyzhou.github.io/javaup/zh-cn/)

## Current Scope

Maven is currently the only supported build tool. JDKs and Maven must already
be installed, and saved project/tool paths are local absolute paths. Maven
settings aliases store paths only, never file contents or credentials. See the
[project scope](https://codeboyzhou.github.io/javaup/reference/project-scope/)
for the complete boundaries.

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
