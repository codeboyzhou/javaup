---
title: Troubleshooting
---

## An IDE Terminal Cannot Find `jup`

Processes inherit environment variables when they start. Completely exit the
IDE and any resident launcher, reopen them, and create a new terminal. On
Windows, sign out and back in if necessary. Verify in PowerShell:

```powershell
Get-Command jup
[Environment]::GetEnvironmentVariable('Path', 'User')
```

Repair only the current PowerShell session with:

```powershell
$env:Path = "$env:USERPROFILE\.javaup\bin;$env:Path"
jup version
```

## Maven Is Installed but Not Found

Run `mvn --version` in the same terminal. Restart the terminal after changing
PATH, or add Maven Wrapper to the project.

## A JDK Is Installed but Not Found

Confirm that it is a full JDK containing `javac`, not a JRE. Expose a custom
location through a versioned variable, or add `<jdkHome>` to Maven Toolchains:

```powershell
$env:JAVA8_HOME = "D:\OpenJDK8"
jup init
```

## A Saved Path Is Stale

Restore the missing Maven or JDK, or refresh detection:

```shell
jup doctor
jup init
```

## A Settings Alias Is Missing

```shell
jup settings list
jup settings use <alias>
# or
jup settings unset
```

## Disable Colored Output

Set the standard `NO_COLOR` environment variable. The repository build script
also accepts `JUP_BUILD_COLOR=always` or `JUP_BUILD_COLOR=never`.

If the problem remains, open a
[GitHub issue](https://github.com/codeboyzhou/javaup/issues) with the operating
system, `jup version`, relevant POM configuration, expected Java version, and
redacted command output.
