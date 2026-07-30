package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/mattn/go-isatty"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/project"
)

type projectCatalog interface {
	List(tool buildtool.Type) ([]project.Candidate, []error, error)
}

const selectedProjectTemplate = `{{ "Selected" | green }}: {{ .Name }}  {{ .ProjectRoot }}`

type terminalProjectPicker struct {
	catalog projectCatalog
}

func newTerminalProjectPicker(catalog projectCatalog) *terminalProjectPicker {
	return &terminalProjectPicker{catalog: catalog}
}

func (p *terminalProjectPicker) Pick(
	_ context.Context,
	tool buildtool.Type,
	keyword string,
	currentDirectory string,
	interactive bool,
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
	if keyword == "" {
		for _, candidate := range candidates {
			if sameProjectRoot(currentDirectory, candidate.ProjectRoot) {
				return candidate.ProjectRoot, nil
			}
		}
	}
	candidates = filterProjectCandidates(candidates, keyword)
	if len(candidates) == 0 {
		if keyword != "" {
			return "", fmt.Errorf("no configured %s projects match %q", tool.DisplayName(), keyword)
		}
		return "", fmt.Errorf("no configured %s projects found; run `jup init` in a project first", tool.DisplayName())
	}
	if keyword != "" && len(candidates) == 1 {
		if err := writeSelectedProject(streams.Stdout, candidates[0]); err != nil {
			return "", err
		}
		return candidates[0].ProjectRoot, nil
	}
	if !interactive {
		return "", fmt.Errorf(
			"project keyword %q matches %d configured %s projects; use a more specific keyword",
			keyword,
			len(candidates),
			tool.DisplayName(),
		)
	}

	selector := promptui.Select{
		Label:     "Select a " + tool.DisplayName() + " project",
		Items:     candidates,
		Size:      min(len(candidates), 10),
		Templates: projectSelectTemplates(streams.Stdout),
		Stdin:     promptInput(streams.Stdin),
		Stdout:    promptOutput(streams.Stdout),
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

func sameProjectRoot(left, right string) bool {
	if strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
		return false
	}
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	if leftErr == nil && rightErr == nil {
		return os.SameFile(leftInfo, rightInfo)
	}
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func writeSelectedProject(output io.Writer, candidate project.Candidate) error {
	if output == nil {
		return nil
	}
	selected, err := template.New("selected-project").Funcs(projectPromptFuncMap(output)).Parse(selectedProjectTemplate)
	if err != nil {
		return fmt.Errorf("prepare selected project output: %w", err)
	}
	if err := selected.Execute(output, candidate); err != nil {
		return fmt.Errorf("show selected project: %w", err)
	}
	if _, err := fmt.Fprintln(output); err != nil {
		return fmt.Errorf("show selected project: %w", err)
	}
	return nil
}

func projectSelectTemplates(output io.Writer) *promptui.SelectTemplates {
	return &promptui.SelectTemplates{
		Active:   `> {{ .Name | cyan }}  {{ .ProjectRoot }}`,
		Inactive: `  {{ .Name }}  {{ .ProjectRoot }}`,
		Selected: selectedProjectTemplate,
		Help:     "↑/↓ move • enter select • ctrl+c cancel",
		FuncMap:  projectPromptFuncMap(output),
	}
}

func projectPromptFuncMap(output io.Writer) template.FuncMap {
	functions := make(template.FuncMap, len(promptui.FuncMap))
	for name, function := range promptui.FuncMap {
		functions[name] = function
	}
	for name, attribute := range map[string]color.Attribute{
		"cyan":  color.FgCyan,
		"green": color.FgGreen,
	} {
		style := newOutputStyle(output, attribute)
		functions[name] = func(value any) string { return style.Sprint(value) }
	}
	return functions
}

func filterProjectCandidates(candidates []project.Candidate, keyword string) []project.Candidate {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return candidates
	}
	filtered := make([]project.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(candidate.Name), keyword) ||
			strings.Contains(strings.ToLower(candidate.ProjectRoot), keyword) {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
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
