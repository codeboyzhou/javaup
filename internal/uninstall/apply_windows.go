package uninstall

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func applyUninstall(spec plan) (bool, error) {
	script, err := os.CreateTemp("", ".javaup-uninstall-*.ps1")
	if err != nil {
		return false, fmt.Errorf("create uninstall helper: %w", err)
	}
	scriptPath := script.Name()
	contents := strings.ReplaceAll(`param(
  [int]$ParentPid,
  [string]$Target,
  [string]$BinDir,
  [string]$HomeDir,
  [string]$ScriptPath,
  [switch]$Purge
)
$ErrorActionPreference = 'Stop'
try {
  Wait-Process -Id $ParentPid -ErrorAction SilentlyContinue

  Get-Process -ErrorAction SilentlyContinue | Where-Object {
    try {
      $_.Path -and [string]::Equals(
        [IO.Path]::GetFullPath($_.Path),
        [IO.Path]::GetFullPath($Target),
        [StringComparison]::OrdinalIgnoreCase
      )
    } catch { $false }
  } | Wait-Process -ErrorAction SilentlyContinue

  Remove-Item -LiteralPath $Target -Force

  $normalizedBin = [IO.Path]::GetFullPath($BinDir).TrimEnd('\')
  $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
  $remaining = foreach ($entry in @($userPath -split ';')) {
    if (-not $entry) { continue }
    try { $normalizedEntry = [IO.Path]::GetFullPath($entry).TrimEnd('\') } catch { $entry; continue }
    if (-not [string]::Equals($normalizedEntry, $normalizedBin, [StringComparison]::OrdinalIgnoreCase)) {
      $entry
    }
  }
  [Environment]::SetEnvironmentVariable('Path', ($remaining -join ';'), 'User')

  $savedHome = [Environment]::GetEnvironmentVariable('JAVAUP_HOME', 'User')
  if ($Purge -and $savedHome) {
    try {
      if ([string]::Equals(
        [IO.Path]::GetFullPath($savedHome).TrimEnd('\'),
        [IO.Path]::GetFullPath($HomeDir).TrimEnd('\'),
        [StringComparison]::OrdinalIgnoreCase
      )) {
        [Environment]::SetEnvironmentVariable('JAVAUP_HOME', $null, 'User')
      }
    } catch {}
  }

  if ($Purge) {
    Remove-Item -LiteralPath $HomeDir -Recurse -Force
  } elseif (Test-Path -LiteralPath $BinDir) {
    $children = @(Get-ChildItem -LiteralPath $BinDir -Force)
    if ($children.Count -eq 0) { Remove-Item -LiteralPath $BinDir -Force }
  }

  if (-not ('JavaupUninstall.NativeMethods' -as [type])) {
    Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
namespace JavaupUninstall {
  public static class NativeMethods {
    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    public static extern IntPtr SendMessageTimeout(
      IntPtr window, uint message, UIntPtr wParam, string lParam,
      uint flags, uint timeout, out UIntPtr result);
  }
}
'@
  }
  $result = [UIntPtr]::Zero
  $null = [JavaupUninstall.NativeMethods]::SendMessageTimeout(
    [IntPtr]0xffff, 0x001a, [UIntPtr]::Zero, 'Environment', 0x0002, 5000, [ref]$result)
} finally {
  Remove-Item -LiteralPath $ScriptPath -Force -ErrorAction SilentlyContinue
}
`, "\n", "\r\n")
	if _, err := script.WriteString(contents); err != nil {
		_ = script.Close()
		_ = os.Remove(scriptPath)
		return false, fmt.Errorf("write uninstall helper: %w", err)
	}
	if err := script.Close(); err != nil {
		_ = os.Remove(scriptPath)
		return false, fmt.Errorf("close uninstall helper: %w", err)
	}

	arguments := []string{
		"-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
		"-ParentPid", strconv.Itoa(os.Getpid()),
		"-Target", spec.Target,
		"-BinDir", spec.BinDir,
		"-HomeDir", spec.Home,
		"-ScriptPath", scriptPath,
	}
	if spec.Purge {
		arguments = append(arguments, "-Purge")
	}
	// #nosec G204 -- arguments are validated local paths passed as distinct values.
	command := exec.Command("powershell.exe", arguments...)
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x08000000,
	}
	if err := command.Start(); err != nil {
		_ = os.Remove(scriptPath)
		return false, fmt.Errorf("start uninstall helper: %w", err)
	}
	if err := command.Process.Release(); err != nil {
		return false, fmt.Errorf("detach uninstall helper: %w", err)
	}
	return true, nil
}
