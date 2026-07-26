---
title: Maven Settings
linkTitle: Maven Settings
weight: 30
---

# Maven Settings

Named aliases make it easy to use a company repository, public mirrors, or
environments with different credentials without replacing the global file.

## Add or Update an Alias

```shell
jup settings add intranet /path/to/settings-intranet.xml
jup settings add public /path/to/settings-public.xml
```

`jup` checks that the path is a regular file containing valid XML with
`<settings>` as its root. It stores only the normalized path; it never copies
the file or its credentials.

## List and Bind Aliases

```shell
jup settings list
cd /path/to/company-project
jup settings use intranet
jup run mvn clean deploy
```

The binding belongs to the local project configuration. When Maven starts,
`jup` automatically prepends the corresponding `--settings` argument.

## Unbind or Remove an Alias

```shell
jup settings unset
jup settings remove intranet
```

`unset` changes only the current project. `remove` deletes the global alias. If
a removed alias is still referenced by a project, bind another alias or run
`jup settings unset` before the next build.
