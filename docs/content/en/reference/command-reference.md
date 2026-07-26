---
title: Command Reference
linkTitle: Command Reference
weight: 10
---

# Command Reference

| Command | Purpose |
|---|---|
| `jup init` | Detect and save the current project's Maven and JDK |
| `jup doctor` | Diagnose the saved toolchain and suggest repairs |
| `jup projects list` | List initialized projects and their status |
| `jup projects remove <path>` | Remove one project's saved configuration |
| `jup projects prune [--dry-run]` | Preview or remove stale project records |
| `jup status` | Show the saved toolchain |
| `jup run mvn <args...> [--project <keyword>]` | Run Maven with the saved toolchain and optionally filter project selection |
| `jup settings add <alias> <file>` | Register or update a settings alias |
| `jup settings list` | List settings aliases |
| `jup settings use <alias>` | Bind an alias to the current project |
| `jup settings unset` | Remove the project's settings binding |
| `jup settings remove <alias>` | Remove a global alias |
| `jup uninit` | Remove the current project's saved configuration |
| `jup update --check` | Check whether a newer release is available |
| `jup update` | Download, verify, and install the latest release |
| `jup uninstall` | Uninstall `jup` while preserving configuration |
| `jup uninstall --purge` | Uninstall `jup` and remove all javaup data |
| `jup version` | Print version, platform, and build revision |

Use the built-in help for authoritative flags and arguments:

```shell
jup --help
jup <command> --help
```

Project-aware commands resolve the nearest initialized project by searching
upward from the current directory. Global registry and alias commands do not
require the terminal to be inside a project.
