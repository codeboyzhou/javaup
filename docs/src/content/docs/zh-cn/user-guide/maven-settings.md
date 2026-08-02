---
title: 管理 settings.xml
---

给不同的 `settings.xml` 文件配置别名。这种用法适合在公司私服、公共镜像和带有不同凭据的环境之间快速进行 `settings.xml` 切换，无需替换全局 `settings.xml` 文件。

## 添加或更新别名

```shell
jup settings add intranet /path/to/settings-intranet.xml
jup settings add public /path/to/settings-public.xml
```

`jup` 会检查路径是否为普通文件、XML 是否有效，以及根元素是否为 `<settings>`。配置只保存规范化后的路径，不会复制文件内容或凭据。

## 列出并使用别名

```shell
jup settings list
cd /path/to/company-project
jup settings use intranet
jup run mvn clean deploy
```

绑定关系保存在本地项目配置中。启动 Maven 时，`jup` 会自动添加对应的 `--settings` 参数。

## 取消绑定或删除别名

```shell
jup settings unset
jup settings remove intranet
```

`unset` 只修改当前项目；`remove` 删除全局别名。如果仍有项目引用已删除的别名，请先绑定其他别名或执行 `jup settings unset`，再运行项目构建。
