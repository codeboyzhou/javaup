---
title: Installation
linkTitle: Installation
weight: 10
---

# Installation

Prebuilt releases are available for Windows, macOS, and Linux on amd64 and
arm64.

## macOS or Linux

```shell
curl -fsSL https://github.com/codeboyzhou/javaup/releases/latest/download/install.sh | sh
```

The installer detects the platform, verifies the release checksum, installs
`jup` under `~/.javaup/bin`, and updates the relevant shell profile.

## Windows

Run in PowerShell 5.1 or later:

```powershell
irm https://github.com/codeboyzhou/javaup/releases/latest/download/install.ps1 | iex
```

The installer verifies the release checksum, installs `jup.exe` under
`%USERPROFILE%\.javaup\bin`, and adds that directory to the user PATH. Restart
terminals and IDEs that were open during installation so they inherit the new
PATH.

## Other Installation Methods

Install with the Go version declared in `go.mod`, or newer:

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
Linux. Archives, checksums, and installers can also be inspected on
[GitHub Releases](https://github.com/codeboyzhou/javaup/releases/latest).

## Installer Options

| Variable | Purpose |
|---|---|
| `JAVAUP_VERSION` | Install a specific release, such as `v0.3.0` |
| `JAVAUP_HOME` | Use a custom absolute installation and configuration directory |
| `JAVAUP_NO_MODIFY_PATH` | Install without updating the shell profile or user PATH |

Verify the result:

```shell
jup version
```
