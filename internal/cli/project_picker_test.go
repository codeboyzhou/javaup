package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/project"
)

type fakeProjectCatalog struct {
	candidates []project.Candidate
	warnings   []error
	err        error
}

func (c fakeProjectCatalog) List(buildtool.Type) ([]project.Candidate, []error, error) {
	return c.candidates, c.warnings, c.err
}

func TestTerminalProjectPickerHandlesArrowKeySelection(t *testing.T) {
	t.Parallel()

	input := bytes.NewBufferString("\x1b[B\r")
	output := &bytes.Buffer{}
	errorsOutput := &bytes.Buffer{}
	picker := newTerminalProjectPicker(fakeProjectCatalog{
		candidates: []project.Candidate{
			{Name: "first", ProjectRoot: "/projects/first"},
			{Name: "second", ProjectRoot: "/projects/second"},
		},
		warnings: []error{errors.New("stale project ignored")},
	})

	root, err := picker.Pick(context.Background(), buildtool.Maven, project.Streams{
		Stdin: input, Stdout: output, Stderr: errorsOutput,
	})
	if err != nil {
		t.Fatalf("Pick() error = %v", err)
	}
	if root != "/projects/second" {
		t.Errorf("Pick() root = %q, want /projects/second", root)
	}
	if !strings.Contains(errorsOutput.String(), "stale project ignored") {
		t.Errorf("Pick() stderr = %q, want warning", errorsOutput.String())
	}
}

func TestPromptUsesPlatformStreamsForProcessTerminal(t *testing.T) {
	t.Parallel()

	if got := promptInput(os.Stdin); got != nil {
		t.Errorf("promptInput(os.Stdin) = %T, want nil platform input", got)
	}
	if got := promptOutput(os.Stdout); got != nil {
		t.Errorf("promptOutput(os.Stdout) = %T, want nil platform output", got)
	}
	input := bytes.NewBuffer(nil)
	output := &bytes.Buffer{}
	if got := promptInput(input); got == nil {
		t.Error("promptInput(buffer) = nil, want injected input")
	}
	if got := promptOutput(output); got == nil {
		t.Error("promptOutput(buffer) = nil, want injected output")
	}
}

func TestTerminalProjectPickerRequiresConfiguredProject(t *testing.T) {
	t.Parallel()

	picker := newTerminalProjectPicker(fakeProjectCatalog{})
	_, err := picker.Pick(context.Background(), buildtool.Maven, project.Streams{})
	if err == nil || !strings.Contains(err.Error(), "run jup init") {
		t.Fatalf("Pick() error = %v, want initialization guidance", err)
	}
}
