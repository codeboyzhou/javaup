---
title: 运行 Maven
linkTitle: 运行 Maven
weight: 20
---

# 运行 Maven

将 Maven 参数放在 `jup run mvn` 后面：

```shell
jup run mvn test
jup run mvn clean package -DskipTests
jup run mvn dependency:tree
```

如需缩小项目选择范围，可在末尾追加 `--project <关键词>`。匹配不区分大小写，并同时检查项目名称和绝对路径：

```shell
jup run mvn clean package --project example
```

只有一个项目命中时会自动选中；多个项目命中时仍显示交互选择器；没有项目命中时会报错。

## 交互式终端

每次运行时，`jup` 都会：

1. 加载所有已初始化的 Maven 项目，即使当前终端不在项目目录中；
2. 按带时间衰减的最近使用分数排序；
3. 如果提供了 `--project <关键词>`，按关键词过滤项目；
4. 唯一命中时自动选中，否则让用户使用上下方向键和回车选择项目；
5. 检查保存的 Maven 可执行文件；
6. 为子进程设置保存的 `JAVA_HOME`；
7. 将所选 JDK 的 `bin` 放到子进程 PATH 首位；
8. 如果绑定了 settings 别名，在参数前加入 `--settings <path>`；
9. 从所选项目根目录启动 Maven。

排序分数采用 14 天半衰期。列表会显示完整路径，确保同名项目也能明确区分。

## 非交互环境

在 CI 和输入重定向场景中，`jup` 不显示选择器，而是从当前目录向上查找最近的已初始化项目，并从当前目录启动 Maven。如果 `--project <关键词>` 能唯一确定一个已配置项目，则改用该项目；如果命中多个项目，需要提供更精确的关键词。

Maven 的标准输入、输出、错误和退出码会直接连接到当前终端或调用进程。当前 shell 的环境变量不会被修改。
