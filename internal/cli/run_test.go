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
	root    string
	tool    buildtool.Type
	streams project.Streams
	err     error
}

func (p *recordingProjectPicker) Pick(
	_ context.Context,
	tool buildtool.Type,
	streams project.Streams,
) (string, error) {
	p.tool = tool
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

func TestRunCommandUsesInteractiveProjectSelection(t *testing.T) {
	t.Parallel()

	runner := &recordingProjectRunner{}
	picker := &recordingProjectPicker{root: "selected-project"}
	workingDirectoryCalled := false
	command := newRunCommandWithPicker(
		func() (projectRunner, error) { return runner, nil },
		func() (string, error) {
			workingDirectoryCalled = true
			return "current-project", nil
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
	if workingDirectoryCalled {
		t.Error("working directory was resolved during interactive selection")
	}
	if picker.tool != buildtool.Maven || runner.root != "selected-project" {
		t.Errorf("picker tool/runner root = %q/%q", picker.tool, runner.root)
	}
	if !reflect.DeepEqual(runner.args, []string{"clean", "package"}) {
		t.Errorf("Run() args = %#v", runner.args)
	}
	if picker.streams.Stdin != io.Reader(input) || picker.streams.Stdout != io.Writer(output) || picker.streams.Stderr != io.Writer(errors) {
		t.Error("picker streams were not forwarded")
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
