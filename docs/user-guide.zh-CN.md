# javaup 详细用户指南

[English](user-guide.md) | 简体中文 | [返回 README](../README.zh-CN.md)

本文档详细说明 `javaup` v0.1.0 的命令行为、工具链探测、配置存储和常见问题。

## 运行前提

- 项目根目录包含 `pom.xml`。
- 项目包含 Maven Wrapper，或者能够从 PATH 找到 `mvn`。
- 本机已经安装与项目 Java 版本匹配的完整 JDK；只有 JRE 无法完成探测。

## 初始化项目

在 Maven 项目中执行：

```shell
jup init
```

初始化依次完成五个阶段：

```text
[1/5] Project      - 确定项目根目录
[2/5] Build Tool   - 识别 Maven、版本和 Wrapper
[3/5] Java Version - 从 POM 读取项目需要的 Java 版本
[4/5] JDK          - 查找匹配的本地 JDK
[5/5] Config       - 保存项目工具链
```

再次执行 `init` 会重新探测 Maven 和 JDK，同时保留已经绑定的 Maven settings
别名。项目路径、符号链接以及 Windows 长路径/8.3 短路径会被规范化，避免同一个
项目生成多份配置。

### 如何确定项目根目录

`jup init` 使用当前目录作为项目根目录，并要求该目录中存在 `pom.xml`。初始化完成后，
`status`、`run`、`settings use/unset` 和 `uninit` 等项目级命令会从当前目录向上
查找，因此也可以在模块或其他子目录中执行。

### 如何选择 Maven

如果项目包含 `mvnw` 或 `mvnw.cmd`，`jup` 会保存并使用 Wrapper；否则会从 PATH
查找 `mvn` 或 `mvn.cmd`。最终解析出的可执行文件和 Maven 版本会写入项目配置。

### 如何识别 Java 版本

`jup` 按以下顺序读取 POM 配置，并支持解析 `${property}` 引用：

1. `maven-compiler-plugin` 中的 `<release>`；
2. `maven.compiler.release`；
3. `maven-compiler-plugin` 中的 `<target>`；
4. `maven.compiler.target`；
5. `maven-compiler-plugin` 中的 `<source>`；
6. `maven.compiler.source`；
7. `java.version`；
8. `jdk.version`。

如果存在本地父 POM，`jup` 会沿 `<relativePath>` 向上解析，最多处理 16 层父级。

### 如何发现本地 JDK

`jup` 按候选顺序检查完整 JDK，并选择第一个 Java 主版本与项目匹配的安装：

1. Maven 运行时提供的 JDK 线索；
2. `JAVA_HOME`、`JDK_HOME` 以及 `JAVA8_HOME`、`JAVA_HOME_17` 等版本化变量；
3. PATH 中 `javac` 对应的 JDK；
4. 已知 JDK 同级、名称符合常见发行版特征的目录；
5. `~/.m2/toolchains.xml` 中的 `<jdkHome>`；
6. 各平台常见安装目录，例如 `Program Files/Java`、`~/.jdks`、SDKMAN!、
   Homebrew 和 asdf。

候选目录必须包含 `bin/javac`（Windows 为 `bin/javac.exe`）。版本优先从 JDK 的
`release` 文件读取，必要时回退到 `javac -version`。

## 查看保存的工具链

```shell
jup status
```

输出包括项目根目录、Maven 版本及来源、Maven 可执行文件、Java 版本、JDK 路径和
Maven settings 别名：

```text
Project: /work/demo
Build tool: Maven 3.9.11 (wrapper)
Build executable: /work/demo/mvnw
Java version: 17.0.12
Java home: /opt/jdks/temurin-17
Maven settings: default
```

该命令会从当前目录向上查找最近的已初始化项目，因此可以在项目任意子目录执行。

## 运行 Maven

将 Maven 参数放在 `jup run mvn` 后面：

```shell
jup run mvn test
jup run mvn clean package -DskipTests
jup run mvn dependency:tree
```

每次运行时，`jup` 都会：

1. 从当前目录向上查找最近的已初始化项目；
2. 检查保存的 Maven 可执行文件是否仍然存在；
3. 为子进程设置保存的 `JAVA_HOME`；
4. 将对应 JDK 的 `bin` 放到子进程 PATH 首位；
5. 如果项目绑定了 settings 别名，在参数前加入 `--settings <path>`；
6. 从当前目录启动 Maven，而不是强制切换到项目根目录。

Maven 的标准输入、标准输出和标准错误会直接连接到当前终端，交互行为、日志和退出码
都会保留。当前 shell 的环境变量不会被修改。

## 管理 Maven settings

命名别名适合在公司私服、公共镜像及具有不同凭据的环境之间切换。

### 添加或更新别名

```shell
jup settings add intranet /path/to/settings-intranet.xml
jup settings add public /path/to/settings-public.xml
```

`jup` 会检查路径是否存在、是否为普通文件、XML 是否有效，以及根元素是否为
`<settings>`。配置只保存规范化后的路径，不会复制文件内容。

### 列出别名

```shell
jup settings list
```

### 为项目绑定别名

```shell
cd /path/to/company-project
jup settings use intranet
jup run mvn clean deploy
```

### 取消项目绑定

```shell
jup settings unset
```

该命令只修改当前项目的选择，全局别名仍然可以被其他项目使用。

### 删除全局别名

```shell
jup settings remove intranet
```

如果删除的别名仍被某个项目引用，下一次构建会提示别名不存在。重新绑定其他别名或
执行 `jup settings unset` 即可恢复。

## 删除项目配置

```shell
jup uninit
```

该命令会删除从当前目录向上找到的最近一个已初始化项目配置。重复执行是安全的；如果
没有配置，会提示无需删除。项目文件、JDK、Maven 安装和 settings 文件都不会被修改。

## 帮助与版本

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

## 自更新

只检查是否有新的稳定版 GitHub Release，不修改当前程序：

```shell
jup update --check
```

下载并安装新版本：

```shell
jup update
```

更新程序会根据当前操作系统和处理器架构选择安装包，使用 Release 中的
`checksums.txt` 校验完成后才替换发起命令的可执行文件。如果当前版本已经是最新版或
更高版本，不会修改任何文件。Windows 不允许运行中的 `.exe` 替换自身，因此替换会在
`jup update` 进程退出后立即完成。

## 配置存储

项目配置和 Maven settings 别名保存在 `JAVAUP_HOME` 下，不会写入项目仓库；
Release 安装器也会将可执行文件放在其中的 `bin` 目录。

| 平台 | 默认 `JAVAUP_HOME` |
|---|---|
| Windows | `%USERPROFILE%\.javaup` |
| macOS | `$HOME/.javaup` |
| Linux | `$HOME/.javaup` |

目录结构：

```text
.javaup/
├── bin/
│   └── jup                # Windows 上为 jup.exe
└── config/
    ├── projects/          # 每个已初始化项目一个 JSON 文件
    └── maven/
        └── settings.json  # Maven settings 别名注册表
```

安装或运行 `jup` 前，可以将 `JAVAUP_HOME` 设置为其他绝对路径。项目配置保存的是
绝对路径快照；移动项目、JDK、Maven Wrapper 或 Maven 安装后，请重新执行
`jup init`。

## 常见问题

### 安装后 IDEA 内置终端找不到 `jup`

Windows 进程只在启动时继承环境变量。安装器会把 `%USERPROFILE%\.javaup\bin`
写入用户 PATH，并通知 Windows 环境已经变化，但安装期间已经运行的 IDEA、JetBrains
Toolbox 和终端仍可能保留旧 PATH。安装命令所在的 PowerShell 会被安装器立即更新，
所以可能出现 Windows Terminal 能运行 `jup`、IDEA 内置终端却找不到的情况。

完全退出 IDEA；如果通过 JetBrains Toolbox 启动，还要从系统托盘退出 Toolbox，然后
重新打开两者并新建终端。仍未生效时，注销并重新登录 Windows 可以确保所有进程重新
读取用户 PATH。可以在新的 IDEA 终端中检查：

```powershell
Get-Command jup
[Environment]::GetEnvironmentVariable('Path', 'User')
```

只想立即修复当前 PowerShell 终端时，可以执行：

```powershell
$env:Path = "$env:USERPROFILE\.javaup\bin;$env:Path"
jup version
```

### Maven 已安装，但找不到 `mvn`

先在同一个终端确认 Maven：

```shell
mvn --version
```

如果刚修改过环境变量，需要重启终端；IDE 内置终端通常需要完全退出 IDE 及其常驻
启动器，才能让父进程重新加载 PATH。没有全局 Maven 时，也可以为项目添加 Maven
Wrapper。

### JDK 已安装，但 `jup` 没有找到

确认安装的是包含 `javac` 的 JDK，而不是 JRE。对于任意自定义目录，可以通过版本化
环境变量显式暴露，同时保留默认 `JAVA_HOME`：

```powershell
$env:JAVA8_HOME = "D:\OpenJDK8"
jup init
```

也可以在 `~/.m2/toolchains.xml` 中添加 `<jdkHome>`。

### 保存的 Maven 或 JDK 路径已经失效

重新探测并保存项目配置：

```shell
jup init
```

### settings 别名不存在

列出可用别名，然后绑定一个有效别名或恢复默认配置：

```shell
jup settings list
jup settings use <alias>
# 或者
jup settings unset
```

### 关闭颜色输出

运行 `jup` 时设置标准的 `NO_COLOR` 环境变量。构建脚本还支持：

```shell
JUP_BUILD_COLOR=always go run build.go
JUP_BUILD_COLOR=never go run build.go
```

## 获取帮助

如果本文档没有解决问题，请提交
[GitHub Issue](https://github.com/codeboyzhou/javaup/issues)，并提供：

- 操作系统和处理器架构；
- `jup version` 输出；
- 相关 POM 属性或编译插件配置；
- 预期及实际探测到的 Java 版本；
- 相关命令输出，并按需隐藏凭据和私有路径。
