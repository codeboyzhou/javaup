<h1 align="center">javaup — 一款多项目 Java 工具链管理器</h1>

<p align="center"><a href="README.md">English</a> | 简体中文</p>

<p align="center">
  <img alt="Go Version" src="https://img.shields.io/github/go-mod/go-version/codeboyzhou/javaup">
  <img alt="Platform" src="https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-blue">
  <a href="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/codeboyzhou/javaup/actions/workflows/ci.yml/badge.svg"></a>
</p>

<p align="center"><strong>让每个 Java 项目自动用对 JDK，用对 Maven，也用对 <code>settings.xml</code>。</strong></p>

<p align="center">
  <a href="#为什么打造-javaup">为什么打造 javaup</a> |
  <a href="#功能亮点">功能亮点</a> |
  <a href="#安装">安装</a> |
  <a href="#快速开始">快速开始</a> |
  <a href="#命令指南">命令指南</a>
</p>

`javaup`（命令名 `jup`）能够识别 Maven 项目所需的 Java 版本，找到匹配的本地 JDK，并记住项目对应的 Maven、JDK 和 `settings.xml`。一次配置，随时可用。并且完全不会对当前 shell 和你的系统环境变量动任何手脚。

## 为什么打造 javaup

一台开发机上往往同时存在多个项目：旧项目仍要求 Java 8，新服务使用 Java 17，
另一个项目可能已经升级到 Java 21。没有项目级工具链管理时，开发者需要反复修改
`JAVA_HOME` 和 `PATH`，记住每个项目应使用的 Maven，或者依赖 IDE 中一套与终端
互不相通的配置。

`jup` 将这些容易出错的手工操作变成一次初始化和一个稳定的执行入口。

| 场景 | 不使用 `jup` | 使用 `jup` |
| --- | --- | --- |
| 切换项目 | 手动修改 `JAVA_HOME`、`PATH` | 项目配置自动选择已保存的 JDK |
| Maven 选择 | 依赖当前 PATH，容易误用版本 | 优先使用项目 Wrapper，否则保存 PATH 中的 Maven |
| 终端与 IDE | 两套配置可能不一致 | 终端构建使用明确、可检查的工具链 |
| 私服配置 | 反复传递 `--settings` 或替换全局文件 | 用别名绑定项目与 `settings.xml` |
| 子模块目录 | 需要先回到项目根目录或自行判断环境 | 从当前目录向上查找已初始化项目 |
| 环境影响 | 修改当前 shell，可能影响后续命令 | 只修改构建子进程的环境 |

需要注意：`jup` 不负责下载或安装 JDK/Maven，也不会修改当前 shell 的
`JAVA_HOME`。它负责发现本机已有工具链、保存项目选择，并在执行构建时隔离环境。
当前支持的构建工具是 **Apache Maven**。

## 功能亮点

- 从 `pom.xml`、本地父 POM 和 `maven-compiler-plugin` 配置中识别 Java 主版本。
- 自动发现 `mvnw` / `mvnw.cmd`，没有 Wrapper 时使用 PATH 中的 Maven。
- 从环境变量、PATH、Maven Toolchains、常见安装目录及已知 JDK 的同级目录发现 JDK。
- 为每个项目保存 Maven 可执行文件、JDK 路径、版本和初始化时间。
- 在构建子进程中设置正确的 `JAVA_HOME`，并把对应的 `bin` 放到 PATH 首位。
- 支持为多个 Maven `settings.xml` 建立别名并按项目选择。
- 可从项目任意子目录执行 `status`、`run`、`settings use/unset` 和 `uninit`。
- 支持 Windows、macOS 和 Linux，CI 会在三个平台执行完整验证。

## 安装

### 使用 Go 安装

需要使用 [`go.mod`](go.mod) 中声明的 Go 版本或更高版本：

```shell
go install github.com/codeboyzhou/javaup/cmd/jup@latest
```

确保 Go 的二进制目录（通常是 `$GOBIN` 或 `$GOPATH/bin`）已经加入 PATH，然后验证：

```shell
jup version
```

### 从源码构建

```shell
git clone https://github.com/codeboyzhou/javaup.git
cd javaup
go run build.go
```

构建产物位于：

```text
Windows: dist/jup.exe
macOS:   dist/jup
Linux:   dist/jup
```

将产物复制到 PATH 中的任意目录即可使用。

### 运行前提

- 项目根目录存在 `pom.xml`。
- 项目包含 Maven Wrapper，或者 `mvn` 能从 PATH 中找到。
- 本机已经安装项目所需版本的完整 JDK；仅安装 JRE 不足以完成探测。

## 快速开始

进入一个 Maven 项目并执行：

```shell
cd /path/to/project
jup init
```

初始化会依次完成五个步骤：

```text
[1/5] Project     - 确定项目根目录
[2/5] Build Tool  - 识别 Maven、版本和 Wrapper
[3/5] Java Version - 从 POM 识别构建所需 Java 版本
[4/5] JDK         - 查找匹配的本地 JDK
[5/5] Config      - 保存项目工具链配置
```

查看结果：

```shell
jup status
```

示例输出：

```text
Project: E:\code\demo
Build tool: Maven 3.9.16 (PATH)
Build executable: D:\tools\maven\bin\mvn.cmd
Java version: 1.8.0_472
Java home: D:\OpenJDK8
Maven settings: default
```

使用保存的工具链执行构建：

```shell
jup run mvn clean package
```

`jup` 会把 Maven 的标准输入、标准输出和标准错误直接连接到当前终端，因此交互、
日志和退出码行为与直接运行 Maven 保持一致。

## 命令指南

### `jup init`

探测当前 Maven 项目并保存工具链配置：

```shell
jup init
```

再次执行 `init` 会重新探测 Maven 和 JDK，并更新配置；已经绑定的 Maven settings
别名会被保留。项目路径、符号链接以及 Windows 长路径/8.3 短路径会被规范化，避免
同一项目生成多份配置。

#### Java 版本如何识别

`jup` 按以下优先级读取 POM 配置，并支持 `${property}` 引用：

1. `maven-compiler-plugin` 的 `<release>`；
2. `maven.compiler.release`；
3. `maven-compiler-plugin` 的 `<target>`；
4. `maven.compiler.target`；
5. `maven-compiler-plugin` 的 `<source>`；
6. `maven.compiler.source`；
7. `java.version`；
8. `jdk.version`。

如果存在本地父 POM，`jup` 会沿 `<relativePath>` 向上解析，最多处理 16 层父级。

#### 本地 JDK 如何发现

`jup` 会按候选顺序检查完整 JDK，并选择 Java 主版本匹配的第一个安装：

1. Maven 当前运行时提供的 JDK 线索；
2. `JAVA_HOME`、`JDK_HOME` 以及 `JAVA8_HOME`、`JAVA_HOME_17` 等版本化变量；
3. PATH 中 `javac` 对应的 JDK；
4. 已知 JDK 同级、名称符合常见 JDK 发行版特征的目录；
5. `~/.m2/toolchains.xml` 中的 `<jdkHome>`；
6. 各平台的常见安装目录，例如 `Program Files/Java`、`~/.jdks`、SDKMAN、
   Homebrew、asdf 等。

候选目录必须包含 `bin/javac`（Windows 为 `bin/javac.exe`）。版本优先从 JDK 的
`release` 文件读取，必要时回退到 `javac -version`。

### `jup status`

显示当前目录所属项目保存的工具链：

```shell
jup status
```

输出包括项目根目录、Maven 版本及来源、Maven 可执行文件、Java 版本、JDK 路径和
Maven settings 别名。可以在项目的任意子目录运行。

### `jup run mvn`

使用初始化时保存的 Maven 和 JDK 执行命令：

```shell
jup run mvn test
jup run mvn clean package -DskipTests
jup run mvn dependency:tree
```

执行时，`jup` 会：

1. 从当前目录向上查找最近的已初始化项目；
2. 检查保存的 Maven 可执行文件是否仍然存在；
3. 为子进程设置保存的 `JAVA_HOME`；
4. 将该 JDK 的 `bin` 放到子进程 PATH 首位；
5. 如果项目绑定了 settings 别名，自动在参数前加入 `--settings <path>`；
6. 在当前目录启动 Maven，而不是强制切换到项目根目录。

当前 shell 的环境变量不会被修改。

### `jup settings`

为 Maven `settings.xml` 建立可复用别名，适合在公司私服、公共镜像和不同认证环境
之间切换。

添加或更新别名：

```shell
jup settings add intranet D:\maven\settings-intranet.xml
jup settings add public D:\maven\settings-public.xml
```

`jup` 会检查文件是否存在、是否为普通文件、XML 是否有效，以及根元素是否为
`<settings>`。配置只保存规范化后的文件路径，不会复制 `settings.xml` 内容。

列出所有别名：

```shell
jup settings list
```

将当前项目绑定到某个别名：

```shell
jup settings use intranet
jup run mvn clean deploy
```

取消当前项目的绑定，但保留全局别名：

```shell
jup settings unset
```

删除全局别名：

```shell
jup settings remove intranet
```

`unset` 与 `remove` 的区别：前者修改项目选择，后者删除全局别名。删除仍被项目引用
的别名后，后续构建会提示别名不存在，需要重新绑定或执行 `settings unset`。

### `jup uninit`

删除当前目录所属项目的本地 `jup` 配置：

```shell
jup uninit
```

它会从当前目录向上查找最近的已初始化项目。重复执行是安全的；如果没有配置，命令
会报告无需删除。该操作不会修改项目文件、JDK、Maven 或 Maven settings 文件。

### 帮助与版本

```shell
jup --help
jup <command> --help
jup version
jup --version
```

版本输出包含语义版本、目标平台和构建对应的 Git 提交短哈希：

```text
javaup version v0.1.0 windows/amd64 (64c2fb07bcad)
```

## 配置存储

项目配置和 Maven settings 别名保存在用户配置目录，不写入项目仓库：

| 平台 | 配置根目录 |
| --- | --- |
| Windows | `%AppData%\javaup` |
| macOS | `~/Library/Application Support/javaup` |
| Linux | `$XDG_CONFIG_HOME/javaup`，未设置时为 `~/.config/javaup` |

目录内容：

```text
javaup/
├── projects/              # 每个已初始化项目一个 JSON 文件
└── maven/
    └── settings.json      # Maven settings 别名注册表
```

项目配置记录的是绝对路径快照。如果移动或删除 JDK、Maven Wrapper、Maven 安装目录，
请重新运行 `jup init`。

## 常见问题

### Maven 已安装，但提示 `mvn` 不在 PATH

先在同一个终端确认：

```shell
mvn --version
```

如果刚修改过环境变量，需要重启终端；IDE 内置终端通常需要重启整个 IDE，才能让其
父进程重新加载 PATH。没有全局 Maven 时，也可以为项目添加 Maven Wrapper。

### JDK 已安装，但 `jup` 没有找到

确认安装的是包含 `javac` 的 JDK，而不是 JRE。对于任意自定义目录，可以通过版本化
环境变量显式暴露，同时保留默认 `JAVA_HOME`：

```powershell
$env:JAVA8_HOME = "D:\OpenJDK8"
jup init
```

也可以在 Maven `~/.m2/toolchains.xml` 中配置 `<jdkHome>`。

### 已保存的 Maven 或 JDK 路径失效

工具链移动后重新初始化即可：

```shell
jup init
```

### 关闭颜色输出

运行 `jup` 时设置标准的 `NO_COLOR` 环境变量。构建脚本还支持：

```shell
JUP_BUILD_COLOR=always go run build.go
JUP_BUILD_COLOR=never go run build.go
```

## 参与贡献

### 开发环境

1. 安装 [`go.mod`](go.mod) 声明的 Go 版本。
2. Fork 并克隆仓库。
3. 在仓库根目录下载依赖并运行验证：

```shell
go mod download
go run build.go verify
```

`verify` 模式依次执行：

```text
gofmt -l .
go vet ./...
go tool -modfile=golangci-lint.mod golangci-lint run
go test ./...
go tool -modfile=govulncheck.mod govulncheck ./...
```

GolangCI-Lint 和 govulncheck 固定在独立的 Go module 文件中，不需要全局安装，也不会
污染应用依赖。

### 推荐开发流程

```shell
# 运行全部单元测试
go test ./...

# 反复执行相关包测试
go test ./internal/javainfo -count=5

# 提交前执行完整验证并生成本机产物
go run build.go
```

构建在首个失败阶段停止。CI 会在 Ubuntu、Windows 和 macOS 上执行
`go run build.go verify`。

提交信息遵循 Conventional Commits：

```text
feat(java): discover sibling jdk installations
fix(maven): handle missing executable in path
docs(readme): rewrite project documentation
```

提交 Pull Request 时，请说明问题背景、设计取舍、验证命令，以及涉及平台差异时的
实际运行结果。

### 项目结构

```text
build.go                    # 本地验证和构建流水线
cmd/jup/                    # CLI 入口
internal/buildinfo/         # 版本和构建元数据
internal/buildtool/maven/   # Maven、Wrapper 和 POM 探测
internal/cli/               # Cobra 命令与终端输出
internal/javainfo/          # JDK 发现与版本匹配
internal/mavensettings/     # Maven settings 别名存储
internal/project/           # 初始化、配置、状态、执行与反初始化
golangci-lint.mod           # 固定的 lint 工具依赖
govulncheck.mod             # 固定的漏洞扫描工具依赖
```

命令层只负责参数解析和输出，可复用的业务逻辑位于 `internal` 包，并通过接口注入以便
测试。新增功能应尽量提供跨平台测试；平台特有行为使用 Go build tags 隔离。

### 注入发布版本

发布构建可以通过 ldflags 注入版本和提交：

```shell
go build \
  -ldflags "-X github.com/codeboyzhou/javaup/internal/buildinfo.Version=v1.0.0 -X github.com/codeboyzhou/javaup/internal/buildinfo.Commit=<commit-hash>" \
  -o jup ./cmd/jup
```

未显式注入提交时，程序会读取 Go 构建信息中的 VCS revision。

## 当前边界

- 当前只支持 Maven 项目，尚未支持 Gradle。
- `jup` 发现并选择已有 JDK，不负责下载、升级或卸载 JDK。
- 项目配置保存在当前用户目录，不会随仓库共享给其他开发者。
- JDK 和 Maven 路径是本地绝对路径；工具移动后需要重新执行 `jup init`。
- Maven settings 注册表保存文件路径，不管理文件内容或凭据。

## License

本项目使用 [Apache License 2.0](LICENSE)。
