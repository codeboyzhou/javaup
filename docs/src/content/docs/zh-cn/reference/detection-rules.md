---
title: 探测规则
---

## 项目根目录

`jup init` 使用当前目录，并要求其中存在 `pom.xml`。初始化后，项目级命令可以从任意子目录向上查找项目。项目路径、符号链接以及 Windows 长路径/8.3 短路径会被规范化，避免同一个项目产生多份配置。

## Maven 选择

如果项目包含 `mvnw` 或 `mvnw.cmd`，`jup` 会保存 Wrapper；否则从 PATH 查找 `mvn` 或 `mvn.cmd`。最终解析的可执行文件和 Maven 版本会写入项目配置。

## Java 版本识别

`jup` 按以下顺序读取 POM，并支持解析 `${property}` 引用：

1. `maven-compiler-plugin` 中的 `<release>`；
2. `maven.compiler.release`；
3. `maven-compiler-plugin` 中的 `<target>`；
4. `maven.compiler.target`；
5. `maven-compiler-plugin` 中的 `<source>`；
6. `maven.compiler.source`；
7. `java.version`；
8. `jdk.version`。

存在本地父 POM 时，`jup` 会沿 `<relativePath>` 向上解析，最多处理 16 层父级。

## JDK 发现顺序

候选按以下顺序检查，并选择第一个 Java 主版本与项目匹配的完整 JDK：

1. Maven 运行时提供的 JDK；
2. `JAVA_HOME`、`JDK_HOME` 以及 `JAVA8_HOME`、`JAVA_HOME_17` 等变量；
3. PATH 中 `javac` 对应的 JDK；
4. 已知 JDK 的同级目录；
5. `~/.m2/toolchains.xml` 中的 `<jdkHome>`；
6. `Program Files/Java`、`~/.jdks`、SDKMAN!、Homebrew 和 asdf 等平台目录。

候选必须包含 `bin/javac`（Windows 为 `bin/javac.exe`）。版本优先从 JDK 的 `release` 文件读取，必要时回退到 `javac -version`。
