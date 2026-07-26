package uninstall

import (
	"fmt"
	"os"
	"strconv"

	"github.com/codeboyzhou/javaup/internal/winprocess"
)

func applyUninstall(spec plan) (bool, error) {
	contents := `param(
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
`
	err := winprocess.StartPowerShellHelper(
		"",
		".javaup-uninstall-*.ps1",
		contents,
		func(scriptPath string) []string {
			arguments := []string{
				"-ParentPid", strconv.Itoa(os.Getpid()),
				"-Target", spec.Target,
				"-BinDir", spec.BinDir,
				"-HomeDir", spec.Home,
				"-ScriptPath", scriptPath,
			}
			if spec.Purge {
				arguments = append(arguments, "-Purge")
			}
			return arguments
		},
	)
	if err != nil {
		return false, fmt.Errorf("start uninstall helper: %w", err)
	}
	return true, nil
}
