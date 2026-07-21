<h1 align="center">javaup</h1>

<p align="center"><strong>面向 Maven 项目的 Java 工具链管理器</strong></p>

<p align="center"><a href="README.md">English</a> | 简体中文</p>

<p align="center">
  <a href="https://github.com/codeboyzhou/javaup/releases/latest"><img alt="最新版本" src="https://img.shields.io/github/v/release/codeboyzhou/javaup"></a>
  <img alt="Go 版本" src="https://img.shields.io/github/go-mod/go-version/codeboyzhou/javaup">
  <a href="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml/badge.svg"></a>
  <img alt="支持平台" src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-blue">
  <a href="LICENSE"><img alt="开源协议" src="https://img.shields.io/github/license/codeboyzhou/javaup"></a>
</p>

<p align="center">
  <strong>让每一次项目构建自动用对 JDK、Maven 和 <code>settings.xml</code>。</strong>
</p>

`javaup`（命令名 `jup`）能够识别 Maven 项目所需的 Java 版本，找到匹配的本地
JDK，并记住项目对应的 Maven 可执行文件、JDK 和可选的 `settings.xml`。以后每次
构建都会复用这套工具链，而且不会修改当前 shell 的 `JAVA_HOME` 或 `PATH`。

```console
$ cd myproject
$ jup init
[1/5] Project OK - /work/myproject
[2/5] Build Tool OK - Maven 3.9.11 (wrapper)
[3/5] Java Version OK - Java 8
[4/5] JDK OK - Java 1.8.0_472 at /opt/jdks/temurin-8
[5/5] Config OK - /home/alex/.javaup/config/projects/...
Initialized javaup project.

$ jup run mvn test
[INFO] BUILD SUCCESS
```

> [!IMPORTANT]
> `jup` 负责选择本机已经安装的工具链，不会下载或卸载 JDK/Maven。v0.1.0 目前
> 只支持 Apache Maven。

## 为什么选择 javaup？

一台开发机上往往同时存在多个 Java 项目：旧系统仍使用 Java 8，当前服务使用
Java 17，新项目可能已经升级到 Java 21。切换项目时，开发者经常需要修改环境变量、
记住每个项目应该使用哪个 Maven，或者依赖一套无法带到终端中的 IDE 配置。

使用 `jup` 后，每个项目都有一套明确、可检查的工具链：

| 操作 | 不使用 `jup` | 使用 `jup` |
|---|---|---|
| 切换项目 | 手动修改 `JAVA_HOME` 和 `PATH` | 自动使用项目保存的 JDK |
| 选择 Maven | 依赖 PATH，容易误用版本 | 优先使用项目 Wrapper，否则保存 PATH 中的 Maven |
| 使用私服 | 反复传递 `--settings` 或替换全局文件 | 为项目绑定命名的 `settings.xml` |
| 在子模块构建 | 回到根目录并重新判断环境 | 从任意子目录识别已初始化项目 |
| 保持 shell 干净 | 修改环境后可能影响后续命令 | 只改变启动出来的构建子进程 |

### 与现有工具是什么关系？

`javaup` 并不是要替代 JDK 版本管理器或 Maven 自带的工具：

- **SDKMAN!、asdf 和 jEnv** 负责安装工具或切换用户、shell 使用的版本；`jup`
  可以发现它们已经安装的 JDK，并按项目使用。
- **Maven Wrapper** 为仓库固定 Maven 发行版；`jup` 会自动发现并优先使用 Wrapper。
- **Maven Toolchains** 允许 Maven 插件选择 JDK；`jup` 可以读取
  `~/.m2/toolchains.xml` 中的 `<jdkHome>`，同时控制启动 Maven 自身所用的 JDK。

`jup` 补充的是项目级绑定：**这个 Maven 可执行文件 + 这个 JDK + 这个 settings
别名**，以后统一通过一个稳定的命令启动。

## 安装

### macOS 或 Linux

```shell
curl -fsSL https://github.com/codeboyzhou/javaup/releases/latest/download/install.sh | sh
```

安装器会识别操作系统和处理器架构、校验 Release 文件、将 `jup` 安装到
`~/.javaup/bin`，并更新对应的 shell 配置文件。

### Windows

在 PowerShell 5.1 或更高版本中运行：

```powershell
irm https://github.com/codeboyzhou/javaup/releases/latest/download/install.ps1 | iex
```

安装器会校验 Release 文件，将 `jup.exe` 安装到
`%USERPROFILE%\.javaup\bin`，并将该目录加入用户 PATH。

如果希望先检查安装文件，可以从 [GitHub Releases](https://github.com/codeboyzhou/javaup/releases/latest)
手动下载安装包、校验文件或安装器。项目为 Windows、macOS 和 Linux 提供 amd64
及 arm64 的预编译程序。

<details>
<summary>其他安装方式</summary>

使用 [`go.mod`](go.mod) 中声明的 Go 版本或更高版本安装：

```shell
go install github.com/codeboyzhou/javaup/cmd/jup@latest
```

或者从源码构建：

```shell
git clone https://github.com/codeboyzhou/javaup.git
cd javaup
go run build.go
```

Windows 产物位于 `dist/jup.exe`，macOS 和 Linux 产物位于 `dist/jup`。

</details>

安装器支持以下可选环境变量：

| 环境变量 | 用途 |
|---|---|
| `JAVAUP_VERSION` | 安装指定版本，例如 `v0.1.0` |
| `JAVAUP_HOME` | 使用自定义的绝对安装及配置目录 |
| `JAVAUP_NO_MODIFY_PATH` | 安装时不修改 shell 配置或用户 PATH |

验证安装结果：

```shell
jup version
```

## 快速开始

开始之前，Maven 项目需要包含 `pom.xml` 和 Maven Wrapper，或者能够从 PATH 找到
`mvn`；本机还需要安装与项目 Java 版本匹配的完整 JDK。

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

`jup run mvn` 会将 Maven 直接连接到当前终端，保留交互输入、日志和退出码，并从
当前目录而不是强制从项目根目录启动，因此可以正常执行针对子模块的构建。

## 功能亮点

- 从 `pom.xml`、编译插件配置、属性和本地父 POM 中识别 Java 版本。
- 自动发现 `mvnw` / `mvnw.cmd`，没有 Wrapper 时回退到 PATH 中的 Maven。
- 从 Maven、环境变量、PATH、Maven Toolchains、常见安装目录和同级 JDK 目录中
  发现本机 JDK。
- 为每个项目保存 Maven 可执行文件、JDK 路径、版本和初始化时间。
- 只在构建子进程中设置 `JAVA_HOME`，并把对应 JDK 的 `bin` 放到 PATH 首位。
- 支持将可复用的 Maven `settings.xml` 别名绑定到不同项目。
- 可以从项目任意子目录执行 `status`、`run`、`settings use/unset` 和 `uninit`。
- 支持 Windows、macOS 和 Linux，CI 会在三个平台执行完整验证。

## Maven settings 别名

只需注册一次 settings 文件，之后可以为每个项目单独选择：

```shell
jup settings add intranet /path/to/settings-intranet.xml
jup settings add public /path/to/settings-public.xml

cd /path/to/company-project
jup settings use intranet
jup run mvn clean deploy
```

`jup` 只保存规范化后的文件路径，不会复制 XML 文件或其中的凭据。执行
`jup settings unset` 可以让项目重新使用 Maven 默认配置。

## 命令速查

| 命令 | 用途 |
|---|---|
| `jup init` | 识别并保存当前项目的 Maven 和 JDK |
| `jup status` | 显示项目保存的工具链 |
| `jup run mvn <args...>` | 使用保存的工具链运行 Maven |
| `jup settings add <别名> <文件>` | 添加或更新 settings 别名 |
| `jup settings list` | 列出 settings 别名 |
| `jup settings use <别名>` | 为当前项目绑定别名 |
| `jup settings unset` | 取消项目的 settings 绑定 |
| `jup settings remove <别名>` | 删除全局别名 |
| `jup uninit` | 删除项目保存的 `jup` 配置 |

Java 版本探测规则、JDK 查找顺序、配置存储和完整排障说明请阅读
[详细用户指南](docs/user-guide.zh-CN.md)。

## 项目状态与当前边界

v0.1.0 是 `javaup` 的第一个公开版本。Release 为 Windows、macOS 和 Linux 提供
amd64 与 arm64 安装包，所有文件都包含在公开的 SHA-256 校验文件中。

当前边界：

- 目前只支持 Maven，尚未支持 Gradle。
- JDK 和 Maven 需要提前安装在本机。
- 项目配置属于当前用户，不会写入项目仓库。
- JDK 和 Maven 使用绝对路径保存；移动工具或项目后需要重新执行 `jup init`。
- Maven settings 别名只保存路径，不保存文件内容或凭据。

如果 `javaup` 无法处理真实项目，请提交
[Issue](https://github.com/codeboyzhou/javaup/issues)，并附上操作系统、POM 结构、预期
Java 版本和相关命令输出。

## 文档

- [详细用户指南](docs/user-guide.zh-CN.md)——命令、探测规则、配置存储和排障
- [贡献指南](CONTRIBUTING.zh-CN.md)——开发环境、完整验证和项目结构
- [英文 README](README.md)
- [英文用户指南](docs/user-guide.md)

## 参与贡献

欢迎提交缺陷报告、兼容性案例、文档改进和代码贡献。运行完整的本地验证流水线：

```shell
go mod download
go run build.go verify
```

提交 Pull Request 前请阅读 [贡献指南](CONTRIBUTING.zh-CN.md)。

## License

本项目使用 [Apache License 2.0](LICENSE)。
