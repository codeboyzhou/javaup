//go:build !windows

package maven

import (
	"context"
	"os/exec"
)

func platformMavenVersionCommand(ctx context.Context, executable string) *exec.Cmd {
	return exec.CommandContext(ctx, executable, "--version")
}
