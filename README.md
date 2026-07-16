# javaup

`javaup` records the JDK and Maven environment required by a project and reuses
that environment for later Maven commands. The command-line executable is
`jup`.

The project currently supports Maven projects. A desktop package exists as a
frontend boundary, but its user interface is not implemented yet.

## What it does

- Reads Maven compiler requirements from the effective project/parent POM
  chain while ignoring inactive profile configuration.
- Finds and validates a matching JDK, including the Java launcher, compiler,
  and installed major version.
- Prefers an executable Maven Wrapper and otherwise validates Maven from
  `PATH` with the selected JDK.
- Stores project environments and named Maven `settings.xml` profiles outside
  the repository.
- Revalidates the recorded JDK and Maven version before every build.
- Redacts common password, token, secret, credential, and key arguments from
  command summaries.

## Requirements

- Rust 1.89.0 or newer to build javaup.
- A JDK matching the Maven project's configured Java release.
- Either an executable Maven Wrapper or Maven available on `PATH`.

The checked-in `rust-toolchain.toml` follows the current stable Rust toolchain
and installs the formatting and lint components used during development. CI
separately verifies the minimum supported Rust 1.89.0 toolchain and installs
LLVM tooling for coverage reporting.

## Build

```text
cargo build --workspace --locked
cargo test --workspace --all-targets --all-features --locked
```

The CLI binary is produced as `target/debug/jup` (or `jup.exe` on Windows).
Build metadata is reproducible: set `SOURCE_DATE_EPOCH` for an epoch-derived
date or `JAVAUP_BUILD_DATE` for an explicit date. Without either value the
reported build date is `unknown`.

## Usage

From a Maven project:

```text
jup init
jup status
jup mvn clean verify
```

`jup init` detects and stores the environment. `jup mvn` forwards all arguments
unchanged, injects the recorded `JAVA_HOME`, and uses the selected wrapper or
system Maven executable.

Named Maven settings files can be managed without copying credentials into the
project:

```text
jup settings add corp-nexus /absolute/path/to/settings.xml
jup settings use corp-nexus
jup settings list
jup settings clear
jup settings remove corp-nexus
```

An explicit Maven `-s` or `--settings` argument overrides the project binding
for that invocation.

## Configuration

The storage root follows the platform configuration directory:

- Windows: `%APPDATA%\javaup`
- macOS: `$HOME/Library/Application Support/javaup`
- Other Unix systems: `$XDG_CONFIG_HOME/javaup`, or `$HOME/.config/javaup`

Set `JAVAUP_HOME` to an absolute path to override it. Set
`JAVAUP_JDK_<MAJOR>_HOME` (for example, `JAVAUP_JDK_21_HOME`) to force a JDK for
a specific Java major version.

Stored files carry an explicit schema version, preserve platform-native paths,
and are replaced atomically while holding an operating-system file lock.
Unversioned, unsupported-version, and otherwise invalid files are rejected
rather than migrated or overwritten. Reinitialize the affected project or
register the settings profile again to create a current record.

## Repository structure

- `core/`: UI-independent discovery, validation, persistence, and process
  descriptions.
- `cli/`: argument parsing, terminal output, and synchronous process execution.
- `desktop/`: reserved desktop frontend package.

See [ARCHITECTURE.md](ARCHITECTURE.md) for boundaries, invariants, and extension
guidance, and [CONTRIBUTING.md](CONTRIBUTING.md) for the validation workflow.
