---
title: 安装
linkTitle: 安装
weight: 10
---

# 安装

项目为 Windows、macOS 和 Linux 提供 amd64 与 arm64 预编译程序。

## macOS 或 Linux

```shell
curl -fsSL https://github.com/codeboyzhou/javaup/releases/latest/download/install.sh | sh
```

安装器会识别平台、校验 Release 文件、将 `jup` 安装到 `~/.javaup/bin`，并更新
对应的 shell 配置文件。

## Windows

在 PowerShell 5.1 或更高版本中运行：

```powershell
irm https://github.com/codeboyzhou/javaup/releases/latest/download/install.ps1 | iex
```

安装器会校验 Release 文件，将 `jup.exe` 安装到
`%USERPROFILE%\.javaup\bin`，并把该目录加入用户 PATH。安装期间已经打开的终端和
IDE 需要重启，才能读取新的 PATH。

## 其他安装方式

使用 `go.mod` 中声明的 Go 版本或更高版本安装：

```shell
go install github.com/codeboyzhou/javaup/cmd/jup@latest
```

或者从源码构建：

```shell
git clone https://github.com/codeboyzhou/javaup.git
cd javaup
go run build.go
```

Windows 产物位于 `dist/jup.exe`，macOS 和 Linux 产物位于 `dist/jup`。也可以在
[GitHub Releases](https://github.com/codeboyzhou/javaup/releases/latest) 中检查并
手工下载安装包、校验文件和安装器。

## 安装器选项

| 环境变量 | 用途 |
|---|---|
| `JAVAUP_VERSION` | 安装指定版本，例如 `v0.2.0` |
| `JAVAUP_HOME` | 使用自定义的绝对安装及配置目录 |
| `JAVAUP_NO_MODIFY_PATH` | 安装时不修改 shell 配置或用户 PATH |

验证安装结果：

```shell
jup version
```
