---
title: Running Maven
linkTitle: Running Maven
weight: 20
---

# Running Maven

Pass Maven arguments after `jup run mvn`:

```shell
jup run mvn test
jup run mvn clean package -DskipTests
jup run mvn dependency:tree
```

## Interactive Terminals

Every invocation:

1. loads all initialized Maven projects, even outside a project directory;
2. orders them by a time-decaying recent-use score;
3. lets you choose with the Up/Down keys and Enter;
4. validates the saved Maven executable;
5. sets the saved `JAVA_HOME` for the child process;
6. places the selected JDK's `bin` first on the child PATH;
7. prepends `--settings <path>` when a settings alias is bound;
8. starts Maven in the selected project root.

The ranking score has a 14-day half-life. Full paths keep projects with the
same directory name unambiguous.

## Non-Interactive Environments

In CI and redirected pipelines, `jup` does not prompt. It searches upward for
the nearest initialized project and starts Maven in the current directory.

Maven's standard input, output, error, and exit status remain connected to the
terminal or calling process. The current shell environment is never modified.
