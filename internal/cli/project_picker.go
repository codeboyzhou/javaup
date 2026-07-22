package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/mattn/go-isatty"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/project"
)

type projectCatalog interface {
	List(tool buildtool.Type) ([]project.Candidate, []error, error)
}

type terminalProjectPicker struct {
	catalog projectCatalog
}

func newTerminalProjectPicker(catalog projectCatalog) *terminalProjectPicker {
	return &terminalProjectPicker{catalog: catalog}
}

func (p *terminalProjectPicker) Pick(
	_ context.Context,
	tool buildtool.Type,
	streams project.Streams,
) (string, error) {
	candidates, warnings, err := p.catalog.List(tool)
	if err != nil {
		return "", err
	}
	for _, warning := range warnings {
		if streams.Stderr != nil {
			_, _ = fmt.Fprintf(streams.Stderr, "jup: warning: %v\n", warning)
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no configured %s projects found; run jup init in a project first", tool.DisplayName())
	}

	selector := promptui.Select{
		Label: "Select a " + tool.DisplayName() + " project",
		Items: candidates,
		Size:  min(len(candidates), 10),
		Templates: &promptui.SelectTemplates{
			Active:   `> {{ .Name | cyan }}  {{ .ProjectRoot }}`,
			Inactive: `  {{ .Name }}  {{ .ProjectRoot }}`,
			Selected: `{{ "Selected" | green }}: {{ .Name }}  {{ .ProjectRoot }}`,
			Help:     "↑/↓ move • enter select • ctrl+c cancel",
		},
		Stdin:  promptInput(streams.Stdin),
		Stdout: promptOutput(streams.Stdout),
	}
	index, _, err := selector.Run()
	if err != nil {
		if errors.Is(err, promptui.ErrInterrupt) || errors.Is(err, promptui.ErrAbort) || errors.Is(err, promptui.ErrEOF) {
			return "", canceledCommandError{}
		}
		return "", fmt.Errorf("select project: %w", err)
	}
	return candidates[index].ProjectRoot, nil
}

type readCloser struct {
	io.Reader
}

func (readCloser) Close() error { return nil }

type writeCloser struct {
	io.Writer
}

func (writeCloser) Close() error { return nil }

func promptInput(input io.Reader) io.ReadCloser {
	if file, ok := input.(*os.File); ok && file == os.Stdin {
		// A nil prompt input selects readline's platform-specific stdin. On
		// Windows this is a console event reader that translates arrow keys.
		return nil
	}
	return readCloser{Reader: input}
}

func promptOutput(output io.Writer) io.WriteCloser {
	if file, ok := output.(*os.File); ok && file == os.Stdout {
		return nil
	}
	return writeCloser{Writer: output}
}

type canceledCommandError struct{}

func (canceledCommandError) Error() string { return "project selection canceled" }

func (canceledCommandError) ExitCode() int { return 130 }

func isInteractiveTerminal(stdin io.Reader, stdout io.Writer) bool {
	input, inputOK := stdin.(*os.File)
	output, outputOK := stdout.(*os.File)
	if !inputOK || !outputOK {
		return false
	}
	return isTerminalDescriptor(input.Fd()) && isTerminalDescriptor(output.Fd())
}

func isTerminalDescriptor(descriptor uintptr) bool {
	return isatty.IsTerminal(descriptor) || isatty.IsCygwinTerminal(descriptor)
}
