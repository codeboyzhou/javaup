//go:build windows

package maven

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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

	commandInterpreter := os.Getenv("ComSpec")
	if commandInterpreter == "" {
		commandInterpreter = "cmd.exe"
	}
	// #nosec G204,G702 -- ComSpec launches only the detected Maven batch script with --version.
	command := exec.CommandContext(ctx, commandInterpreter)
	command.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: fmt.Sprintf(`/d /s /c ""%s" --version"`, strings.ReplaceAll(resolved, `"`, `""`)),
	}
	return command
}
