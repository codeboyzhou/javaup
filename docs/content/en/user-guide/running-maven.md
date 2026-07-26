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

To narrow project selection, append `--project <keyword>`. Matching is
case-insensitive and checks both the project name and its absolute path:

```shell
jup run mvn clean package --project example
```

One match is selected automatically. Multiple matches remain available in the
interactive selector; no matches produce an error.

## Interactive Terminals

Every invocation:

1. loads all initialized Maven projects, even outside a project directory;
2. orders them by a time-decaying recent-use score;
3. filters them by `--project <keyword>` when supplied;
4. selects the only match automatically, or lets you choose with the Up/Down
   keys and Enter;
5. validates the saved Maven executable;
6. sets the saved `JAVA_HOME` for the child process;
7. places the selected JDK's `bin` first on the child PATH;
8. prepends `--settings <path>` when a settings alias is bound;
9. starts Maven in the selected project root.

The ranking score has a 14-day half-life. Full paths keep projects with the
same directory name unambiguous.

## Non-Interactive Environments

In CI and redirected pipelines, `jup` does not prompt. It searches upward for
the nearest initialized project and starts Maven in the current directory.
When `--project <keyword>` uniquely identifies a configured project, that
project is used instead. Multiple matches require a more specific keyword.

Maven's standard input, output, error, and exit status remain connected to the
terminal or calling process. The current shell environment is never modified.
