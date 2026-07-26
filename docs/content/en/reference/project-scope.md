---
title: Project Scope
linkTitle: Project Scope
weight: 50
---

# Project Scope

Current boundaries are intentional:

- Maven is the only supported build tool; Gradle is not yet supported.
- JDKs and Maven must already be installed locally.
- Project configuration is local to the current user and is not written to the
  repository.
- Saved JDK, Maven, and project locations are absolute paths. Re-run `jup init`
  after moving them.
- Maven settings aliases store paths, not file contents or credentials.
- `jup uninstall` manages release-installer installations; package-manager and
  `go install` binaries must be removed by their installation tool.

Release archives are built for Windows, macOS, and Linux on amd64 and arm64.
Every archive is covered by a published SHA-256 checksum, and self-update
verifies the checksum before replacing an executable.
