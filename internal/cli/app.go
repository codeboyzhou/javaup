// Package cli implements the jup command-line interface.
package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

const (
	exitSuccess = 0
	exitFailure = 1
)

// Options contains the dependencies and metadata needed by the CLI.
type Options struct {
	Name        string
	ProductName string
	Description string
	Version     string
	Platform    string
	Commit      string
	Stdout      io.Writer
	Stderr      io.Writer
}

// App owns the Cobra command tree and process-level error handling.
type App struct {
	root   *cobra.Command
	stderr io.Writer
}

// New creates an application with all built-in commands registered.
func New(options Options) *App {
	root := newRootCommand(options)
	return &App{
		root:   root,
		stderr: options.Stderr,
	}
}

// Run executes the requested command and returns a process exit code.
func (a *App) Run(ctx context.Context, args []string) int {
	a.root.SetArgs(args)

	if err := a.root.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(a.stderr, "%s: %v\n", a.root.Name(), err)
		return exitFailure
	}

	return exitSuccess
}
