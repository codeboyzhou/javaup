---
title: Detection Rules
linkTitle: Detection Rules
weight: 20
---

# Detection Rules

## Project Root

`jup init` uses the current directory and requires `pom.xml` there. After
initialization, project-aware commands search upward from any descendant.
Paths, symbolic links, and Windows long-path/8.3-path variants are normalized
so one project does not produce multiple configurations.

## Maven Selection

If the project contains `mvnw` or `mvnw.cmd`, `jup` saves the Wrapper.
Otherwise it finds `mvn` or `mvn.cmd` on PATH. The resolved executable and
reported Maven version are stored with the project.

## Java Version Detection

POM values are evaluated in this order, with `${property}` references
resolved:

1. `<release>` in `maven-compiler-plugin`;
2. `maven.compiler.release`;
3. `<target>` in `maven-compiler-plugin`;
4. `maven.compiler.target`;
5. `<source>` in `maven-compiler-plugin`;
6. `maven.compiler.source`;
7. `java.version`;
8. `jdk.version`.

When a local parent POM exists, `jup` follows `<relativePath>` for up to 16
parent levels.

## JDK Discovery

Candidates are checked in this order; the first full JDK whose Java major
version matches the project is selected:

1. the JDK reported by the Maven runtime;
2. `JAVA_HOME`, `JDK_HOME`, and variables such as `JAVA8_HOME` or
   `JAVA_HOME_17`;
3. the JDK that owns `javac` on PATH;
4. sibling directories of known JDKs;
5. `<jdkHome>` entries in `~/.m2/toolchains.xml`;
6. platform locations such as `Program Files/Java`, `~/.jdks`, SDKMAN!,
   Homebrew, and asdf.

A candidate must contain `bin/javac` (`bin/javac.exe` on Windows). Its version
comes from the JDK `release` file when possible, with `javac -version` as a
fallback.
