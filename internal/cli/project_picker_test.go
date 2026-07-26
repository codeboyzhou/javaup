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

	root, err := picker.Pick(context.Background(), buildtool.Maven, "", true, project.Streams{
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
	_, err := picker.Pick(context.Background(), buildtool.Maven, "", true, project.Streams{})
	if err == nil || !strings.Contains(err.Error(), "run jup init") {
		t.Fatalf("Pick() error = %v, want initialization guidance", err)
	}
}

func TestTerminalProjectPickerFiltersByNameAndPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		keyword string
		want    string
	}{
		{name: "name", keyword: "EXAMPLE", want: "/projects/example-service"},
		{name: "absolute path", keyword: "workspace/team-a", want: "/workspace/team-a/service"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			picker := newTerminalProjectPicker(fakeProjectCatalog{candidates: []project.Candidate{
				{Name: "example-service", ProjectRoot: "/projects/example-service"},
				{Name: "service", ProjectRoot: "/workspace/team-a/service"},
				{Name: "other", ProjectRoot: "/projects/other"},
			}})

			root, err := picker.Pick(context.Background(), buildtool.Maven, test.keyword, false, project.Streams{})
			if err != nil {
				t.Fatalf("Pick() error = %v", err)
			}
			if root != test.want {
				t.Errorf("Pick() root = %q, want %q", root, test.want)
			}
		})
	}
}

func TestTerminalProjectPickerShowsAutomaticallySelectedProject(t *testing.T) {
	t.Parallel()

	output := &bytes.Buffer{}
	picker := newTerminalProjectPicker(fakeProjectCatalog{candidates: []project.Candidate{
		{Name: "mcp-java-sdk-examples", ProjectRoot: `C:\Users\home\code\mcp-java-sdk-examples`},
		{Name: "other", ProjectRoot: `C:\Users\home\code\other`},
	}})

	root, err := picker.Pick(context.Background(), buildtool.Maven, "mcp-java", false, project.Streams{Stdout: output})
	if err != nil {
		t.Fatalf("Pick() error = %v", err)
	}
	if root != `C:\Users\home\code\mcp-java-sdk-examples` {
		t.Errorf("Pick() root = %q", root)
	}
	want := "Selected: mcp-java-sdk-examples  C:\\Users\\home\\code\\mcp-java-sdk-examples\n"
	if output.String() != want {
		t.Errorf("Pick() output = %q, want %q", output.String(), want)
	}
}

func TestTerminalProjectPickerRequiresMoreSpecificKeywordForMultipleNonInteractiveMatches(t *testing.T) {
	t.Parallel()

	picker := newTerminalProjectPicker(fakeProjectCatalog{candidates: []project.Candidate{
		{Name: "example-api", ProjectRoot: "/projects/example-api"},
		{Name: "example-web", ProjectRoot: "/projects/example-web"},
	}})
	_, err := picker.Pick(context.Background(), buildtool.Maven, "example", false, project.Streams{})
	if err == nil || !strings.Contains(err.Error(), "use a more specific keyword") {
		t.Fatalf("Pick() error = %v, want ambiguity guidance", err)
	}
}

func TestTerminalProjectPickerReportsUnmatchedKeyword(t *testing.T) {
	t.Parallel()

	picker := newTerminalProjectPicker(fakeProjectCatalog{candidates: []project.Candidate{
		{Name: "service", ProjectRoot: "/projects/service"},
	}})
	_, err := picker.Pick(context.Background(), buildtool.Maven, "example", false, project.Streams{})
	if err == nil || !strings.Contains(err.Error(), `match "example"`) {
		t.Fatalf("Pick() error = %v, want unmatched keyword error", err)
	}
}
