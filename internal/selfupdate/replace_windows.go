package selfupdate

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/codeboyzhou/javaup/internal/winprocess"
)

func applyUpdate(staged, target string) (bool, error) {
	contents := `param([int]$ParentPid, [string]$Source, [string]$Target, [string]$Script)
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
`
	err := winprocess.StartPowerShellHelper(
		filepath.Dir(target),
		".jup-update-*.ps1",
		contents,
		func(scriptPath string) []string {
			return []string{
				"-ParentPid", strconv.Itoa(os.Getpid()),
				"-Source", staged,
				"-Target", target,
				"-Script", scriptPath,
			}
		},
	)
	if err != nil {
		return false, fmt.Errorf("start update helper: %w", err)
	}
	return true, nil
}
