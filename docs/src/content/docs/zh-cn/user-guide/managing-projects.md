---
title: 管理项目
---

## 初始化项目

在 Maven 项目根目录运行 `jup init`，当前目录中必须存在 `pom.xml`。

```shell
jup init
```

初始化会确定项目根目录、探测 Maven、读取 Java 要求、查找匹配 JDK，最后保存工具链。再次执行会刷新 Maven 和 JDK，同时保留已有的 settings 别名。

初始化完成后，项目级命令会从当前目录向上查找项目。因此 `status`、`doctor`、`run`、`settings use`、`settings unset` 和 `uninit` 也可以在模块或其他子目录中执行。

## 查看工具链

```shell
jup status
```

输出包含项目根目录、Maven 版本及来源、Maven 可执行文件、Java 版本、JDK 路径和 Maven settings 别名。

## 检查工具链

```shell
jup doctor
```

该命令不会修改配置，而是检查保存的配置、POM、Maven 可执行文件、JDK 和绑定的 settings 文件。结果分为 `PASS`、`WARN` 和 `FAIL`。存在失败项时以状态码 1 退出，并给出恢复资源或重新执行 `jup init` 等修复建议。

## 管理全局项目注册表

列出所有已初始化项目：

```shell
jup projects list
```

注册状态分为 `available`、`missing` 和 `invalid`。可以按明确路径删除单个记录，也可以先预览再批量清理失效记录：

```shell
jup projects remove /path/to/project
jup projects prune --dry-run
jup projects prune
```

这些命令只删除保存的配置和使用排序记录，不会删除项目源码、JDK、Maven 安装或 settings 文件。

## 删除单个项目配置

```shell
jup uninit
```

该命令删除从当前目录向上找到的最近一个已初始化项目配置及其使用排序记录。重复执行是安全的，也不会修改项目文件。
