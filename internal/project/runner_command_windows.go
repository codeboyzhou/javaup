//go:build windows

package project

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func platformRunCommand(ctx context.Context, executable string, args []string) *exec.Cmd {
	extension := strings.ToLower(filepath.Ext(executable))
	if extension != ".cmd" && extension != ".bat" {
		// #nosec G204 -- executable is restricted to a validated path from the initialized project configuration.
		return exec.CommandContext(ctx, executable, args...)
	}

	commandInterpreter := os.Getenv("ComSpec")
	if commandInterpreter == "" {
		commandInterpreter = "cmd.exe"
	}
	arguments := make([]string, 0, len(args)+1)
	arguments = append(arguments, quoteBatchArgument(executable))
	for _, argument := range args {
		arguments = append(arguments, quoteBatchArgument(argument))
	}
	// #nosec G204,G702 -- cmd.exe is required to launch the configured Maven batch script.
	command := exec.CommandContext(ctx, commandInterpreter)
	command.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: fmt.Sprintf(`/d /s /c "%s"`, strings.Join(arguments, " ")),
	}
	return command
}

func quoteBatchArgument(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
