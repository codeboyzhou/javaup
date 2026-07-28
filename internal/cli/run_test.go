package cli

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/project"
)

type recordingProjectRunner struct {
	root    string
	tool    buildtool.Type
	args    []string
	streams project.Streams
	err     error
}

type recordingProjectPicker struct {
	root        string
	tool        buildtool.Type
	keyword     string
	currentDir  string
	interactive bool
	streams     project.Streams
	err         error
}

func (p *recordingProjectPicker) Pick(
	_ context.Context,
	tool buildtool.Type,
	keyword string,
	currentDirectory string,
	interactive bool,
	streams project.Streams,
) (string, error) {
	p.tool = tool
	p.keyword = keyword
	p.currentDir = currentDirectory
	p.interactive = interactive
	p.streams = streams
	return p.root, p.err
}

func (r *recordingProjectRunner) Run(
	_ context.Context,
	root string,
	tool buildtool.Type,
	args []string,
	streams project.Streams,
) error {
	r.root = root
	r.tool = tool
	r.args = append([]string(nil), args...)
	r.streams = streams
	return r.err
}

func TestRunCommandForwardsMavenArgumentsAndStreams(t *testing.T) {
	t.Parallel()

	runner := &recordingProjectRunner{}
	command := newRunCommand(func() (projectRunner, error) { return runner, nil }, func() (string, error) {
		return "project-root", nil
	})
	input := bytes.NewBufferString("input")
	output := &bytes.Buffer{}
	errors := &bytes.Buffer{}
	command.SetIn(input)
	command.SetOut(output)
	command.SetErr(errors)
	command.SetArgs([]string{"mvn", "--version"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if runner.root != "project-root" || runner.tool != buildtool.Maven {
		t.Errorf("Run() root/tool = %q/%q", runner.root, runner.tool)
	}
	if !reflect.DeepEqual(runner.args, []string{"--version"}) {
		t.Errorf("Run() args = %#v, want %#v", runner.args, []string{"--version"})
	}
	if runner.streams.Stdin != io.Reader(input) || runner.streams.Stdout != io.Writer(output) || runner.streams.Stderr != io.Writer(errors) {
		t.Error("Run() streams were not forwarded")
	}
	if output.Len() != 0 {
		t.Errorf("Run() output = %q, want no jup output", output.String())
	}
}

func TestRunCommandForwardsMavenArguments(t *testing.T) {
	t.Parallel()

	runner := &recordingProjectRunner{}
	command := newRunCommand(func() (projectRunner, error) { return runner, nil }, func() (string, error) {
		return "project-root", nil
	})
	command.SetArgs([]string{"mvn", "clean", "package", "-DskipTests"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if runner.tool != buildtool.Maven || !reflect.DeepEqual(runner.args, []string{"clean", "package", "-DskipTests"}) {
		t.Errorf("Run() tool/args = %q/%#v", runner.tool, runner.args)
	}
}

func TestRunCommandUsesInteractiveProjectSelectionOutsideProjectRoot(t *testing.T) {
	t.Parallel()

	runner := &recordingProjectRunner{}
	picker := &recordingProjectPicker{root: "selected-project"}
	command := newRunCommandWithPicker(
		func() (projectRunner, error) { return runner, nil },
		func() (string, error) {
			return "current-directory", nil
		},
		func() (projectPicker, error) { return picker, nil },
		func(io.Reader, io.Writer) bool { return true },
	)
	input := bytes.NewBufferString("input")
	output := &bytes.Buffer{}
	errors := &bytes.Buffer{}
	command.SetIn(input)
	command.SetOut(output)
	command.SetErr(errors)
	command.SetArgs([]string{"mvn", "clean", "package"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if picker.tool != buildtool.Maven || runner.root != "selected-project" {
		t.Errorf("picker tool/runner root = %q/%q", picker.tool, runner.root)
	}
	if picker.keyword != "" || !picker.interactive {
		t.Errorf("picker keyword/interactive = %q/%t, want empty/true", picker.keyword, picker.interactive)
	}
	if picker.currentDir != "current-directory" {
		t.Errorf("picker current directory = %q, want current-directory", picker.currentDir)
	}
	if !reflect.DeepEqual(runner.args, []string{"clean", "package"}) {
		t.Errorf("Run() args = %#v", runner.args)
	}
	if picker.streams.Stdin != io.Reader(input) || picker.streams.Stdout != io.Writer(output) || picker.streams.Stderr != io.Writer(errors) {
		t.Error("picker streams were not forwarded")
	}
}

func TestRunCommandFiltersProjectAndDoesNotForwardKeywordOption(t *testing.T) {
	t.Parallel()

	runner := &recordingProjectRunner{}
	picker := &recordingProjectPicker{root: "example-project"}
	workingDirectoryCalled := false
	command := newRunCommandWithPicker(
		func() (projectRunner, error) { return runner, nil },
		func() (string, error) {
			workingDirectoryCalled = true
			return "current-project", nil
		},
		func() (projectPicker, error) { return picker, nil },
		func(io.Reader, io.Writer) bool { return false },
	)
	command.SetArgs([]string{"mvn", "clean", "package", "--project", "example"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if workingDirectoryCalled {
		t.Error("working directory was resolved when a project keyword was provided")
	}
	if picker.keyword != "example" || picker.interactive {
		t.Errorf("picker keyword/interactive = %q/%t, want example/false", picker.keyword, picker.interactive)
	}
	if runner.root != "example-project" || !reflect.DeepEqual(runner.args, []string{"clean", "package"}) {
		t.Errorf("Run() root/args = %q/%#v", runner.root, runner.args)
	}
}

func TestSplitProjectKeyword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		wantArgs    []string
		wantKeyword string
		wantError   string
	}{
		{name: "separate option", args: []string{"clean", "package", "--project", "example"}, wantArgs: []string{"clean", "package"}, wantKeyword: "example"},
		{name: "equals option", args: []string{"test", "--project=Demo"}, wantArgs: []string{"test"}, wantKeyword: "Demo"},
		{name: "ordinary Maven arguments", args: []string{"clean", "package"}, wantArgs: []string{"clean", "package"}},
		{name: "missing keyword", args: []string{"clean", "--project"}, wantError: "requires a keyword"},
		{name: "empty keyword", args: []string{"clean", "--project="}, wantError: "cannot be empty"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			gotArgs, gotKeyword, err := splitProjectKeyword(test.args)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("splitProjectKeyword() error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("splitProjectKeyword() error = %v", err)
			}
			if !reflect.DeepEqual(gotArgs, test.wantArgs) || gotKeyword != test.wantKeyword {
				t.Errorf("splitProjectKeyword() = %#v, %q; want %#v, %q", gotArgs, gotKeyword, test.wantArgs, test.wantKeyword)
			}
		})
	}
}

func TestRunCommandRejectsJava(t *testing.T) {
	t.Parallel()

	command := newRunCommand(func() (projectRunner, error) {
		return &recordingProjectRunner{}, nil
	}, func() (string, error) {
		return "project-root", nil
	})
	command.SetArgs([]string{"java"})

	err := command.ExecuteContext(context.Background())
	if err == nil || !strings.Contains(err.Error(), `unknown command "java"`) {
		t.Fatalf("ExecuteContext() error = %v, want unsupported java command", err)
	}
}
