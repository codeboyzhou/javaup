package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/codeboyzhou/javaup/internal/project"
)

type fakeProjectUninitializer struct {
	path string
}

func (u fakeProjectUninitializer) Uninitialize(
	_ context.Context,
	root string,
	progress project.ProgressFunc,
) (string, bool, error) {
	progress(project.ProgressEvent{
		Step: 1, Total: 2, Name: "Project", Message: "Resolving current project directory", State: project.ProgressStarted,
	})
	progress(project.ProgressEvent{
		Step: 1, Total: 2, Name: "Project", Message: root, State: project.ProgressSucceeded,
	})
	progress(project.ProgressEvent{
		Step: 2, Total: 2, Name: "Config", Message: "Removing local project configuration", State: project.ProgressStarted,
	})
	progress(project.ProgressEvent{
		Step: 2, Total: 2, Name: "Config", Message: u.path, State: project.ProgressSucceeded,
	})
	return u.path, true, nil
}

func TestUninitCommandPrintsConciseProgress(t *testing.T) {
	t.Parallel()

	root := filepath.Join("projects", "demo")
	uninitializer := fakeProjectUninitializer{path: filepath.Join("config", "project.json")}
	command := newUninitCommand(
		func() (projectUninitializer, error) { return uninitializer, nil },
		func() (string, error) { return root, nil },
	)
	var output bytes.Buffer
	command.SetOut(&output)

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	want := "[1/2] Project - Resolving current project directory\n" +
		"[1/2] Project OK - " + root + "\n" +
		"[2/2] Config - Removing local project configuration\n" +
		"[2/2] Config OK - " + uninitializer.path + "\n" +
		"Uninitialized javaup project.\n"
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}
