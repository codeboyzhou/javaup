---
title: 欢迎来到 javaup 项目文档
---

# 让每个 Maven 项目自动使用正确的 Java 工具链

`javaup`（命令名 `jup`）是一款面向 Maven 项目的 Java 工具链管理器。它能自动识别项目所需的 Java 版本，选择匹配的本地 JDK，并记住项目使用的 Maven 版本路径、JDK 和可选的 `settings.xml`。之后的每次构建都会直接复用这套工具链，无需修改当前 shell 的 `JAVA_HOME` 或 `PATH`，也无需在不同版本之间来回手动切换。

> `jup` 负责选择本机已经安装的工具链，不会下载或卸载 JDK/Maven。v0.3.0 目前只支持 Apache Maven。

## 为什么选择 javaup？

一台开发机上往往同时存在多个 Java 项目。切换项目时，开发者经常需要修改环境变量、记住每个项目应该使用哪个 Maven，或者依赖一套无法带到终端中的 IDE 配置。

| 操作            | 不使用 `jup`                         | 使用 `jup`                                 |
|-----------------|--------------------------------------|--------------------------------------------|
| 切换项目        | 手工修改 `JAVA_HOME` 和 `PATH`       | 自动使用项目保存的 JDK                     |
| 选择 Maven      | 依赖 PATH 中的版本                   | 优先使用 Wrapper，否则保存 PATH 中的 Maven |
| 使用私服        | 反复传递 `--settings` 或替换全局文件 | 为项目绑定命名的 `settings.xml`            |
| 从任意位置构建  | 切换目录并重新配置环境               | 选择任意已初始化项目                       |
| 保持 shell 干净 | 环境修改可能影响后续命令             | 只改变启动出来的构建子进程                 |

## 与现有工具的关系

- **SDKMAN!、asdf 和 jEnv** 负责安装工具或切换用户、shell 使用的版本；`jup` 可以发现它们已经安装的 JDK。
- **Maven Wrapper** 为仓库固定 Maven 发行版；`jup` 会自动发现并优先使用。
- **Maven Toolchains** 允许 Maven 插件选择 JDK；`jup` 还会读取 `<jdkHome>`，并控制启动 Maven 自身所用的 JDK。

`jup` 补充的是项目级绑定：**这个 Maven 可执行文件 + 这个 JDK + 这个 settings 别名**，以后统一通过一个稳定的命令启动。

## 功能亮点

- 从 POM 属性、编译插件配置和本地父 POM 中识别 Java 版本。
- 优先使用 Maven Wrapper，没有 Wrapper 时回退到 PATH 中的 Maven。
- 从环境变量、PATH、Maven Toolchains 和常见安装目录中发现本机 JDK。
- 只为构建子进程创建项目专用环境，不改变当前 shell。
- 全局管理已初始化项目，并按最近使用情况排序。
- 检查 Maven、JDK、POM 和 settings 配置是否失效。
- 支持 Windows、macOS 和 Linux。

请先阅读[安装](getting-started/installation/)和[快速开始](getting-started/quick-start/)，也可以直接查看[命令参考](reference/command-reference/)。
