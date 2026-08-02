---
title: Quick Start
---

Before starting, make sure that:

- the project root contains `pom.xml`;
- the project contains Maven Wrapper, or `mvn` is available on PATH;
- a full JDK matching the project's Java version is already installed. A JRE
  alone is not sufficient.

Initialize, inspect, and build:

```shell
cd /path/to/your/maven-project
jup init
jup status
jup run mvn clean package
```

Example status:

```text
Project: /work/demo
Build tool: Maven 3.9.11 (wrapper)
Build executable: /work/demo/mvnw
Java version: 17.0.12
Java home: /opt/jdks/temurin-17
Maven settings: default
```

In an interactive terminal, `jup run mvn` lists every initialized Maven
project. Select one with the arrow keys and Enter. Frequently and recently used
projects rise to the top, and Maven starts in the selected project root.

In CI or redirected pipelines, no picker is displayed. `jup` resolves the
nearest initialized project from the current directory and starts Maven there.
