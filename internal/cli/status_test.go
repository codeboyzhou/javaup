package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/javainfo"
	"github.com/codeboyzhou/javaup/internal/project"
)

type recordingProjectInspector struct {
	root   string
	config project.Config
	err    error
}

func (i *recordingProjectInspector) Inspect(root string) (project.Config, error) {
	i.root = root
	return i.config, i.err
}

func TestStatusCommandShowsCurrentProjectToolchain(t *testing.T) {
	t.Parallel()

	inspector := &recordingProjectInspector{config: project.Config{
		ProjectRoot: "/projects/demo",
		BuildTool: buildtool.Info{
			Type:          buildtool.Maven,
			Version:       "3.9.11",
			Executable:    "/projects/demo/mvnw",
			Wrapper:       true,
			SettingsAlias: "intranet",
		},
		Java: javainfo.Installation{Version: "17.0.12", Home: "/jdks/17"},
	}}
	command := newStatusCommand(func() (projectInspector, error) { return inspector, nil }, func() (string, error) {
		return "/projects/demo/module", nil
	})
	var output bytes.Buffer
	command.SetOut(&output)

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if inspector.root != "/projects/demo/module" {
		t.Errorf("Inspect() root = %q, want %q", inspector.root, "/projects/demo/module")
	}
	assertContains(t, output.String(), []string{
		"Project: /projects/demo",
		"Build tool: Maven 3.9.11 (wrapper)",
		"Build executable: /projects/demo/mvnw",
		"Java version: 17.0.12",
		"Java home: /jdks/17",
		"Maven settings: intranet",
	})
}

func TestStatusCommandReturnsInspectionError(t *testing.T) {
	t.Parallel()

	want := errors.New("inspect failure")
	command := newStatusCommand(func() (projectInspector, error) {
		return &recordingProjectInspector{err: want}, nil
	}, func() (string, error) {
		return "/projects/demo", nil
	})

	if err := command.ExecuteContext(context.Background()); !errors.Is(err, want) {
		t.Fatalf("ExecuteContext() error = %v, want %v", err, want)
	}
}
