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

## 交互式终端

每次运行时，`jup` 都会：

1. 加载所有已初始化的 Maven 项目，即使当前终端不在项目目录中；
2. 按带时间衰减的最近使用分数排序；
3. 让用户使用上下方向键和回车选择项目；
4. 检查保存的 Maven 可执行文件；
5. 为子进程设置保存的 `JAVA_HOME`；
6. 将所选 JDK 的 `bin` 放到子进程 PATH 首位；
7. 如果绑定了 settings 别名，在参数前加入 `--settings <path>`；
8. 从所选项目根目录启动 Maven。

排序分数采用 14 天半衰期。列表会显示完整路径，确保同名项目也能明确区分。

## 非交互环境

在 CI 和输入重定向场景中，`jup` 不显示选择器，而是从当前目录向上查找最近的已
初始化项目，并从当前目录启动 Maven。

Maven 的标准输入、输出、错误和退出码会直接连接到当前终端或调用进程。当前 shell
的环境变量不会被修改。
