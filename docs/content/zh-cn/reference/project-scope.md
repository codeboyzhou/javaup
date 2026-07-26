---
title: 项目边界
linkTitle: 项目边界
weight: 50
---

# 项目边界

当前边界是明确且有意保留的：

- 目前只支持 Maven，尚未支持 Gradle。
- JDK 和 Maven 需要提前安装在本机。
- 项目配置属于当前用户，不会写入项目仓库。
- JDK、Maven 和项目位置使用绝对路径保存；移动后需要重新执行 `jup init`。
- Maven settings 别名只保存路径，不保存文件内容或凭据。
- `jup uninstall` 管理由 Release 安装器完成的安装；包管理器或 `go install`
  安装的程序需要使用对应工具删除。

Release 为 Windows、macOS 和 Linux 提供 amd64 与 arm64 安装包。每个安装包都有
公开的 SHA-256 校验，自更新也会先校验再替换程序。
