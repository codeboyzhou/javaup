---
title: Updates and Uninstallation
---

## Self-Update

Check without changing the executable, or install the latest stable release:

```shell
jup update --check
jup update
```

The updater selects the archive for the current platform, verifies it against
`checksums.txt`, and only then replaces the running installation.

## Uninstall

Remove an installation created by the release installer while preserving
project configurations and settings aliases:

```shell
jup uninstall
```

Also remove all data under `JAVAUP_HOME`:

```shell
jup uninstall --purge
```

`--purge` is irreversible. Safety checks reject filesystem roots, paths that
contain the user home directory, and executables outside `JAVAUP_HOME/bin`.
Shell profile cleanup removes only installer-managed entries.

The command does not remove binaries installed by `go install` or another
package manager. Remove those with the same installation tool.
