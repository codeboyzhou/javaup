# javaup user guide

English | [简体中文](user-guide.zh-CN.md) | [Back to README](../README.md)

This guide documents command behavior, toolchain detection, configuration
storage, and troubleshooting for `javaup` v0.1.0.

## Prerequisites

- The project root contains a `pom.xml`.
- The project contains Maven Wrapper, or `mvn` is available on PATH.
- A full JDK matching the project's Java version is already installed. A JRE
  alone is not sufficient for discovery.

## Initialize a project

Run this from a Maven project:

```shell
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

Running `init` again refreshes the Maven and JDK selection while preserving an
existing Maven settings alias. Project paths, symbolic links, and Windows
long-path/8.3-path variants are normalized so one project does not produce
multiple configurations.

### How the project root is selected

`jup init` uses the current directory as the project root and requires its
`pom.xml` to be present there. After initialization, project-aware commands
such as `status`, `run`, `settings use/unset`, and `uninit` search upward from
the current directory, so they also work inside modules and other descendant
directories.

### How Maven is selected

If the project contains `mvnw` or `mvnw.cmd`, `jup` saves the Wrapper. Otherwise
it finds `mvn` or `mvn.cmd` on PATH. The resolved executable and reported Maven
version are stored with the project configuration.

### How the Java version is detected

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

### How local JDKs are discovered

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
   SDKMAN!, Homebrew, and asdf.

A candidate must contain `bin/javac` (`bin/javac.exe` on Windows). Its version
is read from the JDK `release` file when possible, with `javac -version` as a
fallback.

## Inspect the saved toolchain

```shell
jup status
```

The output includes the project root, Maven version and source, Maven
executable, Java version, JDK path, and Maven settings alias:

```text
Project: /work/demo
Build tool: Maven 3.9.11 (wrapper)
Build executable: /work/demo/mvnw
Java version: 17.0.12
Java home: /opt/jdks/temurin-17
Maven settings: default
```

The command searches upward for the nearest initialized project, so it works
from any descendant directory.

## Run Maven

Pass Maven arguments after `jup run mvn`:

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

Maven's standard input, output, and error streams connect directly to the
current terminal. Interactive behavior, logs, and exit codes are preserved.
The current shell environment is not modified.

## Manage Maven settings

Named aliases make it easy to switch between a company repository, public
mirrors, and environments with different credentials.

### Add or update an alias

```shell
jup settings add intranet /path/to/settings-intranet.xml
jup settings add public /path/to/settings-public.xml
```

`jup` checks that the path exists, is a regular file, contains valid XML, and
has `<settings>` as its root element. Only the normalized path is saved; the
file contents are not copied.

### List aliases

```shell
jup settings list
```

### Bind an alias to a project

```shell
cd /path/to/company-project
jup settings use intranet
jup run mvn clean deploy
```

### Unbind a project

```shell
jup settings unset
```

This changes only the current project's selection and leaves the global alias
available for other projects.

### Remove a global alias

```shell
jup settings remove intranet
```

If a removed alias is still referenced by a project, the next build reports
that it is missing. Bind another alias or run `jup settings unset` to recover.

## Remove project configuration

```shell
jup uninit
```

The command removes the nearest initialized project configuration found from
the current directory. It is idempotent: if no configuration exists, it
reports that there is nothing to remove. Project files, JDKs, Maven
installations, and settings files are never modified.

## Help and version

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

## Self-update

Check for a newer stable GitHub Release without changing the executable:

```shell
jup update --check
```

Download and install it:

```shell
jup update
```

The updater selects the archive for the current operating system and
architecture, verifies it against the release's `checksums.txt`, and only then
replaces the executable that launched the command. If the current version is
already the latest (or newer), no files are changed. On Windows, replacement
finishes immediately after the `jup update` process exits because a running
`.exe` cannot replace itself.

## Configuration storage

Project configurations and Maven settings aliases live under `JAVAUP_HOME` and
are not written into project repositories. Release installers also place the
executable in its `bin` directory.

| Platform | Default `JAVAUP_HOME` |
|---|---|
| Windows | `%USERPROFILE%\.javaup` |
| macOS | `$HOME/.javaup` |
| Linux | `$HOME/.javaup` |

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

Set `JAVAUP_HOME` to an absolute path before installing or running `jup` to use
another location. Project configurations contain absolute path snapshots; run
`jup init` again after moving a project, JDK, Maven Wrapper, or Maven
installation.

## Troubleshooting

### The IDEA terminal cannot find `jup` after installation

Windows processes inherit environment variables only when they start. The
installer adds `%USERPROFILE%\.javaup\bin` to the user PATH and notifies Windows
about the environment change, but IDEA, JetBrains Toolbox, and terminals that
were already running during installation may retain the old PATH. The installer
updates its own PowerShell process immediately, so `jup` can work in Windows
Terminal while an IDEA terminal still cannot find it.

Exit IDEA completely. If IDEA is launched through JetBrains Toolbox, also exit
Toolbox from the system tray, then reopen both applications and create a new
terminal. If that still does not work, sign out of Windows and sign back in so
every process reloads the user PATH. Verify the result in a new IDEA terminal:

```powershell
Get-Command jup
[Environment]::GetEnvironmentVariable('Path', 'User')
```

To repair only the current PowerShell session immediately, run:

```powershell
$env:Path = "$env:USERPROFILE\.javaup\bin;$env:Path"
jup version
```

### Maven is installed, but `mvn` is not found

Check Maven in the same terminal:

```shell
mvn --version
```

Restart the terminal after changing environment variables. An IDE terminal
often requires fully exiting the IDE and any resident launcher so its parent
process reloads PATH. If global Maven is unavailable, add Maven Wrapper to the
project.

### A JDK is installed, but `jup` cannot find it

Make sure it is a JDK containing `javac`, not a JRE. For an arbitrary custom
location, expose a versioned variable while keeping the default `JAVA_HOME`:

```powershell
$env:JAVA8_HOME = "D:\OpenJDK8"
jup init
```

Alternatively, add a `<jdkHome>` entry to `~/.m2/toolchains.xml`.

### A saved Maven or JDK path no longer exists

Refresh the project configuration:

```shell
jup init
```

### A settings alias is missing

List available aliases, then bind a valid one or return to the default:

```shell
jup settings list
jup settings use <alias>
# or
jup settings unset
```

### Disable colored output

Set the standard `NO_COLOR` environment variable when running `jup`. The build
script additionally supports:

```shell
JUP_BUILD_COLOR=always go run build.go
JUP_BUILD_COLOR=never go run build.go
```

## Getting help

If the guide does not resolve the problem, open a
[GitHub issue](https://github.com/codeboyzhou/javaup/issues) and include:

- the operating system and architecture;
- `jup version` output;
- the relevant POM properties or compiler plugin configuration;
- expected and detected Java versions;
- the command output, with credentials and private paths redacted as needed.
