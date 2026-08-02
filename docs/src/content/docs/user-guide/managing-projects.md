---
title: Managing Projects
---

## Initialize

Run `jup init` from a Maven project root. The current directory must contain
`pom.xml`.

```shell
jup init
```

Initialization resolves the project root, detects Maven, reads the Java
requirement, locates a matching JDK, and persists the toolchain. Running it
again refreshes Maven and JDK while preserving an existing settings alias.

After initialization, project-aware commands search upward from the current
directory. `status`, `doctor`, `run`, `settings use`, `settings unset`,
and `uninit` therefore work inside modules and other descendant directories.

## Inspect

```shell
jup status
```

The output includes project root, Maven version and source, Maven executable,
Java version, JDK path, and the Maven settings alias.

## Diagnose

```shell
jup doctor
```

The command checks the saved configuration, POM, Maven executable, JDK, and
bound settings file without changing configuration. Results are reported as
`PASS`, `WARN`, or `FAIL`. Any failure produces exit status 1 and an actionable
repair, usually restoring the resource or running `jup init` again.

## Manage the Global Registry

List every initialized project:

```shell
jup projects list
```

Registry status is `available`, `missing`, or `invalid`. Remove one record by
explicit path, or preview and prune every stale record:

```shell
jup projects remove /path/to/project
jup projects prune --dry-run
jup projects prune
```

These commands remove saved configuration and usage records only. They never
delete project sources, JDKs, Maven installations, or settings files.

## Remove One Project Configuration

```shell
jup uninit
```

This removes the nearest initialized project configuration and its usage
ranking record. It is idempotent and does not modify project files.
