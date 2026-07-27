---
title: Welcome to the javaup documentation
---

`javaup` (command: `jup`) is a Java toolchain manager built for Maven projects.
It detects the Java version required by a project, selects a matching installed
JDK, and remembers the project's Maven installation path, JDK, and optional
`settings.xml`. Every subsequent build reuses this toolchain, so there is no need
to change `JAVA_HOME` or `PATH` in the current shell or manually switch between
versions.

> `jup` selects toolchains that are already installed. It does not download or
> uninstall JDKs or Maven. Apache Maven is the only supported build tool in
> v0.3.0.

## Why javaup?

A development machine often hosts projects from several Java generations.
Switching between them usually means changing environment variables,
remembering which Maven installation to use, or relying on IDE settings that do
not follow you into the terminal.

| Task                     | Without `jup`                                      | With `jup`                                                    |
|--------------------------|----------------------------------------------------|---------------------------------------------------------------|
| Switch projects          | Edit `JAVA_HOME` and `PATH`                        | Automatically reuse the JDK version saved for the project     |
| Select Maven             | Depend on whatever is on PATH                      | Automatically prefer the Wrapper or reuse saved Maven version |
| Use private repositories | Repeat `--settings` or replace a global file       | Automatically apply the project's settings alias              |
| Build from anywhere      | Change directories and reconstruct the environment | Automatically load the selected project's toolchain           |
| Preserve the shell       | Risk affecting later commands                      | Automatically isolate changes to the spawned build process    |

`jup` complements existing Java tools instead of replacing them:

- **SDKMAN!, asdf, and jEnv** install or switch tools for a user or shell. `jup`
  can discover the JDKs they installed.
- **Maven Wrapper** pins the Maven distribution for a repository. `jup` detects
  and prefers it automatically.
- **Maven Toolchains** lets Maven plugins select a JDK. `jup` also discovers
  `<jdkHome>` entries and controls the JDK that launches Maven itself.

The layer `jup` adds is a local project binding: **this Maven executable + this
JDK + this settings alias**, launched through one stable command.

## Highlights

- Detects Java requirements from POM properties, compiler plugin configuration,
  and local parent POMs.
- Prefers Maven Wrapper and falls back to Maven from PATH.
- Discovers installed JDKs across environment variables, PATH, Maven
  Toolchains, and common installation locations.
- Runs builds with a project-specific child-process environment while leaving
  the current shell unchanged.
- Manages initialized projects globally and ranks them by recent use.
- Diagnoses stale Maven, JDK, POM, and settings configuration.
- Supports Windows, macOS, and Linux.

Start with [Installation](getting-started/installation/) and
[Quick Start](getting-started/quick-start/), or open the
[Command Reference](reference/command-reference/).
