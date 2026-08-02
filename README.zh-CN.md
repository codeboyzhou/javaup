<h1 align="center">javaup</h1>

<p align="center"><a href="README.md">English</a> | 简体中文</p>

<p align="center"><strong>让每个 Maven 项目都能自动使用正确的 Java 工具链</strong></p>

<p align="center">
  <a href="https://github.com/codeboyzhou/javaup/releases/latest"><img alt="最新版本" src="https://img.shields.io/github/v/release/codeboyzhou/javaup"></a>
  <img alt="Go 版本" src="https://img.shields.io/github/go-mod/go-version/codeboyzhou/javaup">
  <a href="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml/badge.svg"></a>
  <img alt="支持平台" src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-blue">
  <a href="LICENSE"><img alt="开源协议" src="https://img.shields.io/github/license/codeboyzhou/javaup"></a>
</p>

`javaup`（命令名 `jup`）是一款面向 Maven 项目的 Java 工具链管理器。它能理解每个项目的工具链需求，自动识别所需的 Java 版本，选择匹配的本地 JDK，并记住应该配套使用的 Maven、JDK 和可选的 `settings.xml`。之后每次构建都会在新的子进程中加载这套工具链，不修改当前 shell 的 `JAVA_HOME`、`PATH` 或其他环境配置。

<p align="center">
  <img src="docs/demo/demo.gif" alt="javaup 演示">
</p>

## 为什么选择 javaup？

一台开发机上经常需要同时构建面向 Java 8、Java 17、Java 21 等不同版本的项目。不使用 `jup` 时，每次切换都要重新拼装环境：选择正确的 JDK、找到项目期望的 Maven，有时还要指定不同的 `settings.xml`。这些信息容易遗忘，重复配置也十分繁琐。

| 操作            | 不使用 `jup`                         | 使用 `jup`                                     |
|-----------------|--------------------------------------|------------------------------------------------|
| 切换项目        | 手工修改 `JAVA_HOME` 和 `PATH`       | 直接复用该项目保存的 JDK                       |
| 选择 Maven      | 依赖 PATH 中碰巧存在的 `mvn`         | 优先使用 Maven Wrapper 或保存的 Maven          |
| 使用私服        | 反复传递 `--settings` 或替换全局文件 | 自动应用项目保存的 settings 别名               |
| 从任意位置构建  | 切换目录并重新配置环境               | 选择已初始化项目并加载完整工具链               |
| 保持 shell 干净 | 环境修改可能影响后续命令             | 将构建专用环境限制在子进程中                   |

`jup` 是对现有 Java 工具的补充，而不是替代：

- **SDKMAN! 和 asdf** 负责安装工具或切换用户、shell 使用的版本；`jup` 可以发现它们安装的 JDK，也可以识别通过环境变量和 PATH 暴露的 JDK。
- **Maven Wrapper** 为仓库固定 Maven 发行版；`jup` 会自动发现并优先使用。
- **Maven Toolchains** 允许 Maven 插件选择 JDK；`jup` 还会读取 `<jdkHome>`，并控制启动 Maven 自身所用的 JDK。

`jup` 补充的是一份持久的本地项目绑定：**这个 Maven + 这个 JDK + 这个 settings 别名**。以后在任意终端中都可以通过同一个稳定命令启动。

> [!IMPORTANT]
> `jup` 负责选择本机已经安装的 JDK 和 Maven，不会下载或卸载它们。目前唯一支持的构建工具是 Apache Maven。

## 安装

macOS 或 Linux：

```shell
curl -fsSL https://github.com/codeboyzhou/javaup/releases/latest/download/install.sh | sh
```

Windows PowerShell 5.1 或更高版本：

```powershell
irm https://github.com/codeboyzhou/javaup/releases/latest/download/install.ps1 | iex
```

预编译版本支持 Windows、macOS 和 Linux 的 amd64 与 arm64。校验文件、`go install`、源码构建和安装器选项请参阅[安装指南](https://codeboyzhou.github.io/javaup/zh-cn/start-here/installation/)。

## 快速开始

只需准备一个包含 `pom.xml` 的项目、Maven Wrapper 或 PATH 中的 `mvn`，以及与项目 Java 版本匹配的完整 JDK：

```shell
cd /path/to/your/maven-project
jup init
jup status
jup run mvn clean package
```

在交互式终端中，`jup run mvn` 可以选择任意已初始化项目，并优先显示最近且频繁使用的项目。在 CI 或输入重定向环境中，它会直接找到最近的已初始化项目，不显示选择器。

## 主要能力

- 从 POM 属性、编译插件配置和本地父 POM 中识别 Java 版本。
- 优先使用 Maven Wrapper，没有 Wrapper 时回退到 PATH 中的 Maven。
- 从 Maven、SDKMAN!、asdf、环境变量、PATH、Maven Toolchains 和常见平台目录中发现本机 JDK。
- 只为构建子进程应用所选 JDK 和可选的 Maven settings 别名。
- 全局管理并检查已初始化项目，不修改项目源码文件。

## 参与贡献

欢迎提交缺陷报告、兼容性案例、文档改进和代码贡献。运行完整的本地验证流水线：

```shell
go mod download
go run build.go verify
```

提交 Pull Request 前请阅读 [CONTRIBUTING.zh-CN.md](CONTRIBUTING.zh-CN.md)。

## License

本项目使用 [Apache License 2.0](LICENSE)。
