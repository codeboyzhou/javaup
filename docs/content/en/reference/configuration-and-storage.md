---
title: Configuration and Storage
linkTitle: Configuration and Storage
weight: 30
---

# Configuration and Storage

Project configurations, usage state, and Maven settings aliases live under
`JAVAUP_HOME`; nothing is written into Maven project repositories.

| Platform | Default `JAVAUP_HOME` |
|---|---|
| Windows | `%USERPROFILE%\.javaup` |
| macOS | `$HOME/.javaup` |
| Linux | `$HOME/.javaup` |

```text
.javaup/
├── bin/
│   └── jup                # jup.exe on Windows
├── config/
│   ├── projects/          # one JSON document per initialized project
│   └── maven/
│       └── settings.json  # Maven settings alias registry
└── state/
    └── project-usage.json # recent-frequency project ranking
```

Set `JAVAUP_HOME` to a custom absolute path before installing or running `jup`
to use another location. Project configuration stores absolute path snapshots.
Run `jup init` again after moving a project, JDK, Maven Wrapper, or Maven
installation.

Settings aliases store paths only, not XML content or credentials. Treat the
referenced settings files according to their own security requirements.
