package project

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/codeboyzhou/javaup/internal/buildtool"
)

// Streams are connected directly to a command started by Runner.
type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type processSpec struct {
	executable  string
	args        []string
	directory   string
	environment []string
	streams     Streams
}

type processExecutor interface {
	Execute(ctx context.Context, spec processSpec) error
}

type osProcessExecutor struct{}

func (osProcessExecutor) Execute(ctx context.Context, spec processSpec) error {
	command := platformRunCommand(ctx, spec.executable, spec.args)
	command.Dir = spec.directory
	command.Env = spec.environment
	command.Stdin = spec.streams.Stdin
	command.Stdout = spec.streams.Stdout
	command.Stderr = spec.streams.Stderr
	return command.Run()
}

// Runner executes a project's detected build tool with its saved Java environment.
type Runner struct {
	store    configFinder
	executor processExecutor
}

// NewDefaultRunner creates a runner backed by the user project configuration store.
func NewDefaultRunner() (*Runner, error) {
	store, err := NewDefaultConfigStore()
	if err != nil {
		return nil, err
	}
	return NewRunner(store, osProcessExecutor{}), nil
}

// NewRunner creates a project command runner from replaceable services.
func NewRunner(store configFinder, executor processExecutor) *Runner {
	return &Runner{store: store, executor: executor}
}

// Run executes the requested build tool saved for the project containing root.
func (r *Runner) Run(
	ctx context.Context,
	root string,
	tool buildtool.Type,
	args []string,
	streams Streams,
) error {
	config, _, found, err := r.store.Find(root)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no initialized javaup project found from %s; run jup init", root)
	}

	executable, err := configuredBuildToolExecutable(config, tool)
	if err != nil {
		return err
	}
	if err := validateExecutable(executable, tool); err != nil {
		return err
	}
	directory, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve command directory: %w", err)
	}

	err = r.executor.Execute(ctx, processSpec{
		executable:  executable,
		args:        append([]string(nil), args...),
		directory:   filepath.Clean(directory),
		environment: environmentForJava(config.Java.Home),
		streams:     streams,
	})
	if err == nil {
		return nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return err
	}
	return fmt.Errorf("run %s: %w", tool.DisplayName(), err)
}

func configuredBuildToolExecutable(config Config, tool buildtool.Type) (string, error) {
	if strings.TrimSpace(config.Java.Version) == "" {
		return "", fmt.Errorf("project configuration has no Java version; run jup init again")
	}
	javaHome := strings.TrimSpace(config.Java.Home)
	if javaHome == "" {
		return "", fmt.Errorf("project configuration has no JDK; run jup init again")
	}
	if !filepath.IsAbs(javaHome) {
		return "", fmt.Errorf("configured JDK path is not absolute; run jup init again")
	}

	if config.BuildTool.Type != tool {
		return "", fmt.Errorf(
			"project build tool is %s, not %s",
			config.BuildTool.Type.DisplayName(),
			tool.DisplayName(),
		)
	}
	if strings.TrimSpace(config.BuildTool.Version) == "" {
		return "", fmt.Errorf("project configuration has no %s version; run jup init again", tool.DisplayName())
	}
	executable := strings.TrimSpace(config.BuildTool.Executable)
	if executable == "" {
		return "", fmt.Errorf("project configuration has no %s executable; run jup init again", tool.DisplayName())
	}
	if !filepath.IsAbs(executable) {
		return "", fmt.Errorf("configured %s path is not absolute; run jup init again", tool.DisplayName())
	}
	return filepath.Clean(executable), nil
}

func validateExecutable(path string, tool buildtool.Type) error {
	displayName := tool.DisplayName()
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("configured %s executable does not exist at %s; run jup init again", displayName, path)
		}
		return fmt.Errorf("inspect configured %s executable: %w", displayName, err)
	}
	if info.IsDir() {
		return fmt.Errorf("configured %s executable is a directory at %s; run jup init again", displayName, path)
	}
	return nil
}

func environmentForJava(javaHome string) []string {
	javaBin := filepath.Join(javaHome, "bin")
	environment := os.Environ()
	filtered := make([]string, 0, len(environment)+2)
	pathValue := ""
	for _, entry := range environment {
		name, value, found := strings.Cut(entry, "=")
		if !found {
			filtered = append(filtered, entry)
			continue
		}
		if environmentNameEqual(name, "JAVA_HOME") {
			continue
		}
		if environmentNameEqual(name, "PATH") {
			pathValue = value
			continue
		}
		filtered = append(filtered, entry)
	}
	filtered = append(filtered, "JAVA_HOME="+javaHome)
	if pathValue == "" {
		filtered = append(filtered, "PATH="+javaBin)
	} else {
		filtered = append(filtered, "PATH="+javaBin+string(os.PathListSeparator)+pathValue)
	}
	return filtered
}

func environmentNameEqual(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
