---
title: 常见问题
---

## IDE 内置终端找不到 `jup` 命令

进程只在启动时继承环境变量。完全退出 IDE 和常驻启动器，重新打开后创建新终端。Windows 上仍未生效时，可以注销并重新登录。使用 PowerShell 检查：

```powershell
Get-Command jup
[Environment]::GetEnvironmentVariable('Path', 'User')
```

只修复当前 PowerShell 会话：

```powershell
$env:Path = "$env:USERPROFILE\.javaup\bin;$env:Path"
jup version
```

## Maven 已安装但未找到

在同一个终端运行 `mvn --version`。修改 PATH 后重启终端，或者为项目添加 Maven Wrapper。

## JDK 已安装但未找到

确认安装的是包含 `javac` 的完整 JDK，而不是 JRE。可以通过版本化变量暴露自定义目录，或者在 Maven Toolchains 中添加 `<jdkHome>`：

```powershell
$env:JAVA8_HOME = "D:\OpenJDK8"
jup init
```

## 保存的路径已经失效

恢复缺失的 Maven 或 JDK，也可以重新探测：

```shell
jup doctor
jup init
```

## settings 别名不存在

```shell
jup settings list
jup settings use <alias>
# 或者
jup settings unset
```

## 关闭颜色输出

设置标准的 `NO_COLOR` 环境变量。仓库构建脚本还支持 `JUP_BUILD_COLOR=always` 和 `JUP_BUILD_COLOR=never`。

如果问题仍未解决，请提交 [GitHub Issue](https://github.com/codeboyzhou/javaup/issues)，并提供操作系统、`jup version`、相关 POM 配置、预期 Java 版本和经过脱敏的命令输出。
