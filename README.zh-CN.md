<h1 align="center">javaup</h1>

<p align="center"><a href="README.md">English</a> | 简体中文</p>

<p align="center"><strong>让每个 Maven 项目自动使用正确的 Java 工具链</strong></p>

<p align="center">
  <a href="https://github.com/codeboyzhou/javaup/releases/latest"><img alt="最新版本" src="https://img.shields.io/github/v/release/codeboyzhou/javaup"></a>
  <img alt="Go 版本" src="https://img.shields.io/github/go-mod/go-version/codeboyzhou/javaup">
  <a href="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml/badge.svg"></a>
  <img alt="支持平台" src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-blue">
  <a href="LICENSE"><img alt="开源协议" src="https://img.shields.io/github/license/codeboyzhou/javaup"></a>
</p>

`javaup`（命令名 `jup`）是一款面向 Maven 项目的 Java 工具链管理器。它能自动识别项目所需的 Java 版本，选择匹配的本地 JDK，并记住项目使用的 Maven 版本路径、JDK 和可选的 `settings.xml`。之后的每次构建都会直接复用这套工具链，无需修改当前 shell 的 `JAVA_HOME` 或 `PATH`，也无需在不同版本之间来回手动切换。

<p align="center">
  <img src="docs/demo/demo.gif" alt="javaup 演示">
</p>

> [!IMPORTANT]
> `jup` 负责选择本机已经安装的 JDK 和 Maven，不会下载或卸载它们。v0.3.0 目前只支持 Apache Maven。

## 安装

macOS 或 Linux：

```shell
curl -fsSL https://github.com/codeboyzhou/javaup/releases/latest/download/install.sh | sh
```

Windows PowerShell 5.1 或更高版本：

```powershell
irm https://github.com/codeboyzhou/javaup/releases/latest/download/install.ps1 | iex
```

预编译版本支持 Windows、macOS 和 Linux 的 amd64 与 arm64。校验文件、`go install`、源码构建和安装器选项请参阅[安装指南](https://codeboyzhou.github.io/javaup/zh-cn/getting-started/installation/)。

## 快速开始

Maven 项目需要包含 `pom.xml` 和 Maven Wrapper，或者能够从 PATH 找到 `mvn`；本机还需要安装与项目 Java 版本匹配的完整 JDK。

```shell
cd /path/to/your/maven-project
jup init
jup status
jup run mvn clean package
```

在交互式终端中，`jup run mvn` 可以选择任意已初始化项目，并把最近且频繁使用的项目排在前面。在 CI 或输入重定向环境中，它会直接找到最近的已初始化项目，不显示选择器。

## 主要能力

- 从 POM 属性、编译插件配置和本地父 POM 中识别 Java 版本。
- 优先使用 Maven Wrapper，没有 Wrapper 时回退到 PATH 中的 Maven。
- 从 Maven、环境变量、PATH、Maven Toolchains 和常见平台目录中发现本机 JDK。
- 只为构建子进程应用所选 JDK 和可选的 Maven settings 别名。
- 全局管理并检查已初始化项目，不修改项目源码文件。

详细的手工切换环境对比，以及与 SDKMAN!、asdf、jEnv、Maven Wrapper 和 Maven Toolchains 的关系，请阅读[为什么选择 javaup？](https://codeboyzhou.github.io/javaup/zh-cn/#为什么选择-javaup)。

## 文档

- [简体中文文档站点](https://codeboyzhou.github.io/javaup/zh-cn/)
- [快速开始](https://codeboyzhou.github.io/javaup/zh-cn/getting-started/quick-start/)
- [使用指南](https://codeboyzhou.github.io/javaup/zh-cn/user-guide/)
- [命令参考](https://codeboyzhou.github.io/javaup/zh-cn/reference/command-reference/)
- [常见问题](https://codeboyzhou.github.io/javaup/zh-cn/reference/troubleshooting/)
- [English Documentation](https://codeboyzhou.github.io/javaup/)

## 当前边界

目前只支持 Maven。JDK 和 Maven 需要提前安装，保存的项目与工具路径是当前用户的本地绝对路径。Maven settings 别名只保存路径，不保存文件内容或凭据。完整说明请阅读[项目边界](https://codeboyzhou.github.io/javaup/zh-cn/reference/project-scope/)。

## 参与贡献

欢迎提交缺陷报告、兼容性案例、文档改进和代码贡献。运行完整的本地验证流水线：

```shell
go mod download
go run build.go verify
```

提交 Pull Request 前请阅读 [CONTRIBUTING.zh-CN.md](CONTRIBUTING.zh-CN.md)。

## License

本项目使用 [Apache License 2.0](LICENSE)。
