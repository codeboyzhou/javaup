//go:build windows

package project

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/codeboyzhou/javaup/internal/winprocess"
)

func platformRunCommand(ctx context.Context, executable string, args []string) *exec.Cmd {
	extension := strings.ToLower(filepath.Ext(executable))
	if extension != ".cmd" && extension != ".bat" {
		// #nosec G204 -- executable is restricted to a validated path from the initialized project configuration.
		return exec.CommandContext(ctx, executable, args...)
	}

	return winprocess.BatchCommand(ctx, executable, args...)
}
