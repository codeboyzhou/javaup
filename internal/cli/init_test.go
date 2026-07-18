package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/codeboyzhou/javaup/internal/project"
)

type fakeProjectInitializer struct {
	config project.Config
	path   string
}

func (i fakeProjectInitializer) Initialize(_ context.Context, _ string, progress project.ProgressFunc) (project.Config, string, error) {
	progress(project.ProgressEvent{
		Step: 1, Total: 5, Name: "Project", Message: i.config.ProjectRoot, State: project.ProgressStarted,
	})
	progress(project.ProgressEvent{
		Step: 1, Total: 5, Name: "Project", Message: i.config.ProjectRoot, State: project.ProgressSucceeded,
	})
	return i.config, i.path, nil
}

func TestInitCommandPrintsConciseProgress(t *testing.T) {
	t.Parallel()

	config := project.Config{
		ProjectRoot: filepath.Join("projects", "demo"),
	}
	initializer := fakeProjectInitializer{config: config, path: filepath.Join("config", "project.json")}
	command := newInitCommand(
		func() (projectInitializer, error) { return initializer, nil },
		func() (string, error) { return config.ProjectRoot, nil },
	)
	var output bytes.Buffer
	command.SetOut(&output)

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	want := "[1/5] Project - " + config.ProjectRoot + "\n" +
		"[1/5] Project OK - " + config.ProjectRoot + "\n" +
		"Initialized javaup project.\n"
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}
