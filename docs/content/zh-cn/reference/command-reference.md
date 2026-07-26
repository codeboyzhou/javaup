---
title: 命令参考
linkTitle: 命令参考
weight: 10
---

# 命令参考

| 命令 | 用途 |
|---|---|
| `jup init` | 识别并保存当前项目的 Maven 和 JDK |
| `jup doctor` | 检查保存的工具链并给出修复建议 |
| `jup projects list` | 列出已初始化项目及其状态 |
| `jup projects remove <路径>` | 删除指定项目保存的配置 |
| `jup projects prune [--dry-run]` | 预览或清理失效项目记录 |
| `jup status` | 显示保存的工具链 |
| `jup run mvn <参数...> [--project <关键词>]` | 使用保存的工具链运行 Maven，并可按关键词过滤项目选择 |
| `jup settings add <别名> <文件>` | 添加或更新 settings 别名 |
| `jup settings list` | 列出 settings 别名 |
| `jup settings use <别名>` | 为当前项目绑定别名 |
| `jup settings unset` | 取消项目的 settings 绑定 |
| `jup settings remove <别名>` | 删除全局别名 |
| `jup uninit` | 删除当前项目保存的配置 |
| `jup update --check` | 检查是否有新版本 |
| `jup update` | 下载、校验并安装最新版本 |
| `jup uninstall` | 卸载 `jup`，但保留配置 |
| `jup uninstall --purge` | 卸载 `jup` 并删除全部 javaup 数据 |
| `jup version` | 显示版本、平台和构建提交 |

使用内置帮助查看权威的参数说明：

```shell
jup --help
jup <command> --help
```

项目级命令会从当前目录向上查找最近的已初始化项目。全局项目注册表和 settings 别名管理命令不要求终端位于某个项目中。
