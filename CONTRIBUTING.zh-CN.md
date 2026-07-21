# 参与 javaup 开发

[English](CONTRIBUTING.md) | 简体中文 | [返回 README](README.zh-CN.md)

欢迎提交缺陷报告、兼容性案例、文档改进和代码贡献。

## 报告问题

提交 Issue 前，请先搜索已有 Issue，并阅读[常见问题](docs/user-guide.zh-CN.md#常见问题)。
为了让问题能够被复现，请提供：

- 操作系统和处理器架构；
- `jup version` 输出；
- 相关 POM 属性、编译插件配置和父 POM 结构；
- 预期及实际探测到的 Java 版本；
- 相关命令输出，并按需隐藏凭据和私有路径。

## 开发环境

1. 安装 [`go.mod`](go.mod) 中声明的 Go 版本或更高版本。
2. Fork 并克隆仓库。
3. 在仓库根目录下载依赖并运行验证：

```shell
go mod download
go run build.go verify
```

验证依次执行：

```text
gofmt -l .
go vet ./...
go tool -modfile=golangci-lint.mod golangci-lint run
go test ./...
go tool -modfile=govulncheck.mod govulncheck ./...
```

GolangCI-Lint 和 govulncheck 固定在独立的 Go module 文件中，不需要全局安装，
也不会污染应用依赖。

## 推荐开发流程

运行全部单元测试：

```shell
go test ./...
```

如果改动涉及并发或平台探测，可以反复运行受影响包的测试：

```shell
go test ./internal/javainfo -count=5
```

提交前执行完整验证并生成本机产物：

```shell
go run build.go
```

构建会在第一个失败阶段停止。CI 会在 Ubuntu、Windows 和 macOS 上执行
`go run build.go verify`。

## 设计与测试要求

- 参数解析和输出放在 `internal/cli`。
- 可复用行为放在对应的 `internal` 包中，并尽量通过接口注入外部操作。
- 为新增行为和缺陷修复补充测试。
- 优先编写跨平台测试；无法避免的平台差异使用 Go build tags 隔离。
- 保持当前 shell 环境不变；构建所需修改只能作用于启动出来的子进程。
- 不得将 Maven settings 内容或凭据复制到 javaup 配置中。

## 项目结构

```text
build.go                    # 本地验证和构建流水线
cmd/jup/                    # CLI 入口
internal/apphome/           # JAVAUP_HOME 解析
internal/buildinfo/         # 版本和构建元数据
internal/buildtool/maven/   # Maven、Wrapper 和 POM 探测
internal/cli/               # Cobra 命令与终端输出
internal/javainfo/          # JDK 发现与版本匹配
internal/mavensettings/     # Maven settings 别名存储
internal/project/           # 初始化、配置、状态与执行
golangci-lint.mod           # 固定的 lint 工具依赖
govulncheck.mod             # 固定的漏洞扫描工具依赖
```

## Commit 与 Pull Request

提交信息遵循 Conventional Commits：

```text
feat(java): discover sibling jdk installations
fix(maven): handle missing executable in path
docs(readme): improve quick start
```

subject 必须使用英文、小写和祈使语气，最长 72 个字符，结尾不加句号。允许的 type
包括 `feat`、`fix`、`docs`、`style`、`refactor`、`perf`、`test`、`build`、
`ci` 和 `chore`。

Pull Request 需要说明：

- 问题背景和预期行为；
- 重要的设计取舍；
- 验证命令与结果；
- 涉及平台差异或工具探测时的真实环境运行结果。

请保持 Pull Request 聚焦。如果无关的重构或文档修改会增加审核难度，应拆分提交。

## 发布元数据

Release 构建通过 ldflags 注入语义版本和 Git 提交：

```shell
go build \
  -ldflags "-X github.com/codeboyzhou/javaup/internal/buildinfo.Version=v1.0.0 -X github.com/codeboyzhou/javaup/internal/buildinfo.Commit=<commit-hash>" \
  -o jup ./cmd/jup
```

未显式注入提交时，`jup` 会从 Go 构建信息中读取 VCS revision。

## License

提交贡献即表示你同意按照 [Apache License 2.0](LICENSE) 授权该贡献。
