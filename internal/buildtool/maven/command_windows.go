//go:build windows

package maven

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/codeboyzhou/javaup/internal/winprocess"
)

func platformMavenVersionCommand(ctx context.Context, executable string) *exec.Cmd {
	resolved := executable
	if path, err := exec.LookPath(executable); err == nil {
		resolved = path
	}
	extension := strings.ToLower(filepath.Ext(resolved))
	if extension != ".cmd" && extension != ".bat" {
		return exec.CommandContext(ctx, executable, "--version")
	}

	return winprocess.BatchCommand(ctx, resolved, "--version")
}
