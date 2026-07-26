---
title: 更新与卸载
linkTitle: 更新与卸载
weight: 40
---

# 更新与卸载

## 自更新

只检查是否存在新版本，或者安装最新稳定版：

```shell
jup update --check
jup update
```

更新程序会选择当前平台的安装包，使用 `checksums.txt` 完成校验后才替换程序。
Windows 上的替换会在当前进程退出后立即完成。

## 卸载

卸载由 Release 安装器安装的程序，同时保留项目配置和 settings 别名：

```shell
jup uninstall
```

同时删除 `JAVAUP_HOME` 下的全部数据：

```shell
jup uninstall --purge
```

`--purge` 不可撤销。安全检查会拒绝文件系统根目录、包含用户主目录的路径，以及
`JAVAUP_HOME/bin` 之外的可执行文件。清理 shell 配置时只删除安装器管理的内容。

该命令不会删除通过 `go install` 或其他包管理器安装的程序，请使用对应的安装工具
卸载。`jup uninit` 的含义不同，它只删除单个项目的配置。
