# Architecture

## Dependency direction

```text
                 +------------------+
                 | javaup-desktop   |
                 | (future adapter) |
                 +---------+--------+
                           |
+------------+             |             +------------------+
| jup binary +-------------+------------>| javaup-core      |
| CLI adapter|                           | domain/use cases |
+------------+                           +------------------+
```

Frontend packages depend on `javaup-core`; the core package never depends on a
terminal or desktop framework. This keeps discovery, validation, and storage
behavior consistent across frontends.

## Core modules

- `java`: discovers a matching JDK and validates its launcher, compiler, and
  reported version. Candidate installations are ordered newest-first by numeric
  version components.
- `project::detection`: interprets Maven POMs and chooses a validated JDK and
  Maven environment. Progress is exposed as domain events.
- `project::build`: revalidates a recorded environment and creates a
  `ProcessInvocation`; it does not start the user's build.
- `project::config`: serializes project environments and validates their schema
  and project identity.
- `maven_settings`: validates `settings.xml` files and stores named references.
- `storage`: owns platform directories, native path encoding, atomic replacement,
  and operating-system file locking.
- `process`: defines the captured-process port (`ProcessRunner`) and the
  platform-native process description shared with frontends.

## Main flows

### Initialization

1. The frontend calls `ProjectEnvironment::detect` (or a runner/observer-aware
   variant).
2. Detection reads the project POM and eligible parent POMs without executing
   Maven project code.
3. A matching JDK is discovered and validated.
4. An executable Maven Wrapper is validated, or system Maven is probed with the
   selected JDK through `ProcessRunner`.
5. The environment is atomically saved under the user configuration directory.

### Build execution

1. The frontend loads the nearest initialized project environment.
2. Core revalidates the JDK, wrapper/system executable, and Maven version.
3. Core returns `ProcessInvocation`, containing native program/argument/path
   values plus `JAVA_HOME` and `PATH` overrides.
4. The frontend chooses how to execute it. The CLI uses blocking inherited I/O;
   a desktop frontend can use asynchronous execution, cancellation, and streamed
   output without changing core.

### Persistence

Each record contains a schema version and lossless native path encoding. Writes
take an adjacent OS lock, create and sync a temporary file in the destination
directory, atomically replace the target, and sync the parent directory where
the platform supports it. Readers reject duplicate, incomplete, unknown-version,
or wrong-project records. Corrupt data is never silently replaced.
Schema versions are compatibility boundaries: javaup does not migrate older or
unversioned records. Users recreate incompatible project/profile state.

## Domain invariants

- `ProjectEnvironment` owns common JDK state and exactly one
  `BuildToolEnvironment` variant.
- Build-tool-specific fields live inside their corresponding enum variant.
- A stored environment is usable only for the canonical project path recorded
  in that file.
- Every build validates installed tools again; saved versions are expectations,
  not trust anchors.
- Paths and process arguments stay as `PathBuf`/`OsString` until presentation,
  avoiding lossy Unicode conversion at system boundaries.
- Core errors retain their typed source chain. Presentation layers decide how
  to render them and which exit code to return.

## Adding another build tool

1. Add a variant and environment type to `BuildToolEnvironment`.
2. Implement detection and build-invocation modules behind the existing public
   use cases.
3. Introduce a new persistence schema version, reject previous versions, and
   document how users recreate the affected state.
4. Add frontend commands/adapters without moving process execution into core.
5. Add unit, property, integration, and platform-matrix coverage for the new
   branch.

## Testing strategy

- Unit tests cover parsers, domain errors, selection rules, and command
  formatting.
- Property tests exercise byte/path codecs and generated Java/Maven version
  strings.
- Integration tests run the CLI against temporary JDK, wrapper, project, and
  settings fixtures.
- CI runs formatting, Clippy with warnings denied, documentation, MSRV tests,
  current stable tests on Linux/Windows/macOS, and LLVM coverage reporting.
