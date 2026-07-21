package selfupdate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func applyUpdate(staged, target string) (bool, error) {
	script, err := os.CreateTemp(filepath.Dir(target), ".jup-update-*.ps1")
	if err != nil {
		return false, fmt.Errorf("create update helper: %w", err)
	}
	scriptPath := script.Name()
	contents := strings.ReplaceAll(`param([int]$ParentPid, [string]$Source, [string]$Target, [string]$Script)
$ErrorActionPreference = 'Stop'
try {
  Wait-Process -Id $ParentPid -ErrorAction SilentlyContinue
  $backup = $Target + '.update-backup'
  Remove-Item -LiteralPath $backup -Force -ErrorAction SilentlyContinue
  if (Test-Path -LiteralPath $Target) { Move-Item -LiteralPath $Target -Destination $backup }
  try {
    Move-Item -LiteralPath $Source -Destination $Target
    Remove-Item -LiteralPath $backup -Force -ErrorAction SilentlyContinue
  } catch {
    if (-not (Test-Path -LiteralPath $Target) -and (Test-Path -LiteralPath $backup)) {
      Move-Item -LiteralPath $backup -Destination $Target
    }
    throw
  }
} finally {
  Remove-Item -LiteralPath $Script -Force -ErrorAction SilentlyContinue
}
`, "\n", "\r\n")
	if _, err := script.WriteString(contents); err != nil {
		_ = script.Close()
		_ = os.Remove(scriptPath)
		return false, fmt.Errorf("write update helper: %w", err)
	}
	if err := script.Close(); err != nil {
		_ = os.Remove(scriptPath)
		return false, fmt.Errorf("close update helper: %w", err)
	}

	// #nosec G204 -- arguments are validated updater paths passed as distinct values.
	command := exec.Command(
		"powershell.exe",
		"-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
		"-ParentPid", strconv.Itoa(os.Getpid()),
		"-Source", staged,
		"-Target", target,
		"-Script", scriptPath,
	)
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x08000000,
	}
	if err := command.Start(); err != nil {
		_ = os.Remove(scriptPath)
		return false, fmt.Errorf("start update helper: %w", err)
	}
	if err := command.Process.Release(); err != nil {
		return false, fmt.Errorf("detach update helper: %w", err)
	}
	return true, nil
}
