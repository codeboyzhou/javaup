---
title: 快速开始
linkTitle: 快速开始
weight: 20
---

# 快速开始

开始之前请确认：

- 项目根目录包含 `pom.xml`；
- 项目包含 Maven Wrapper，或者能够从 PATH 找到 `mvn`；
- 本机已经安装与项目 Java 版本匹配的完整 JDK，只有 JRE 无法完成探测。

初始化、检查、构建只需三步：

```shell
cd /path/to/your/maven-project
jup init
jup status
jup run mvn clean package
```

`jup status` 输出示例：

```text
Project: /work/demo
Build tool: Maven 3.9.11 (wrapper)
Build executable: /work/demo/mvnw
Java version: 17.0.12
Java home: /opt/jdks/temurin-17
Maven settings: default
```

在交互式终端中，`jup run mvn` 会列出所有已经初始化的 Maven 项目。使用方向键和
回车选择项目；最近且频繁使用的项目会自动排在前面，Maven 从所选项目根目录启动。

在 CI 或输入重定向等非交互环境中不会显示选择器。`jup` 从当前目录向上寻找最近的
已初始化项目，并直接启动 Maven。

接下来可以阅读[管理项目](../../user-guide/managing-projects/)和
[运行 Maven](../../user-guide/running-maven/)。
