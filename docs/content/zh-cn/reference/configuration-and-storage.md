---
title: 配置与存储
linkTitle: 配置与存储
weight: 30
---

# 配置与存储

项目配置、使用状态和 Maven settings 别名都位于 `JAVAUP_HOME` 下，不会写入
Maven 项目仓库。

| 平台 | 默认 `JAVAUP_HOME` |
|---|---|
| Windows | `%USERPROFILE%\.javaup` |
| macOS | `$HOME/.javaup` |
| Linux | `$HOME/.javaup` |

```text
.javaup/
├── bin/
│   └── jup                # Windows 上为 jup.exe
├── config/
│   ├── projects/          # 每个已初始化项目一个 JSON 文件
│   └── maven/
│       └── settings.json  # Maven settings 别名注册表
└── state/
    └── project-usage.json # 项目最近使用频率排序
```

安装或运行 `jup` 前，可以将 `JAVAUP_HOME` 设置为其他绝对路径。项目配置保存的是
绝对路径快照；移动项目、JDK、Maven Wrapper 或 Maven 安装后，请重新执行
`jup init`。

settings 别名只保存路径，不保存 XML 内容或凭据。请按照引用文件自身的安全要求
管理这些文件。
