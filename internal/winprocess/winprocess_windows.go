//go:build windows

// Package winprocess centralizes Windows command interpreter and detached
// PowerShell helper process behavior.
package winprocess

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// BatchCommand returns a cmd.exe process that invokes a batch file with args.
func BatchCommand(ctx context.Context, executable string, args ...string) *exec.Cmd {
	commandInterpreter := os.Getenv("ComSpec")
	if commandInterpreter == "" {
		commandInterpreter = "cmd.exe"
	}
	arguments := make([]string, 0, len(args)+1)
	arguments = append(arguments, quoteBatchArgument(executable))
	for _, argument := range args {
		arguments = append(arguments, quoteBatchArgument(argument))
	}
	// #nosec G204,G702 -- ComSpec is the required Windows command interpreter.
	command := exec.CommandContext(ctx, commandInterpreter)
	command.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: fmt.Sprintf(`/d /s /c "%s"`, strings.Join(arguments, " ")),
	}
	return command
}

// StartPowerShellHelper writes and starts a hidden, detached PowerShell script.
// arguments receives the generated script path and returns script parameters.
func StartPowerShellHelper(
	directory string,
	pattern string,
	contents string,
	arguments func(scriptPath string) []string,
) error {
	script, err := os.CreateTemp(directory, pattern)
	if err != nil {
		return fmt.Errorf("create helper script: %w", err)
	}
	scriptPath := script.Name()
	contents = strings.ReplaceAll(contents, "\n", "\r\n")
	if _, err := script.WriteString(contents); err != nil {
		_ = script.Close()
		_ = os.Remove(scriptPath)
		return fmt.Errorf("write helper script: %w", err)
	}
	if err := script.Close(); err != nil {
		_ = os.Remove(scriptPath)
		return fmt.Errorf("close helper script: %w", err)
	}

	commandArgs := []string{
		"-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
	}
	commandArgs = append(commandArgs, arguments(scriptPath)...)
	// #nosec G204 -- arguments are validated local paths passed as distinct values.
	command := exec.Command("powershell.exe", commandArgs...)
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x08000000,
	}
	if err := command.Start(); err != nil {
		_ = os.Remove(scriptPath)
		return fmt.Errorf("start helper script: %w", err)
	}
	if err := command.Process.Release(); err != nil {
		return fmt.Errorf("detach helper script: %w", err)
	}
	return nil
}

func quoteBatchArgument(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
