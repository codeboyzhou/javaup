//go:build !windows

package project

import (
	"context"
	"os/exec"
)

func platformRunCommand(ctx context.Context, executable string, args []string) *exec.Cmd {
	// #nosec G204 -- executable is restricted to a validated path from the initialized project configuration.
	return exec.CommandContext(ctx, executable, args...)
}
