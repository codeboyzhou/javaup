package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/javainfo"
	"github.com/codeboyzhou/javaup/internal/project"
)

type fakeProjectInitializer struct {
	config project.Config
	path   string
}

func (i fakeProjectInitializer) Initialize(_ context.Context, _ string, progress project.ProgressFunc) (project.Config, string, error) {
	progress(project.ProgressEvent{
		Step: 1, Total: 5, Name: "PROJECT", Message: i.config.ProjectRoot, State: project.ProgressStarted,
	})
	progress(project.ProgressEvent{
		Step: 1, Total: 5, Name: "PROJECT", Message: i.config.ProjectRoot, State: project.ProgressSucceeded,
	})
	return i.config, i.path, nil
}

func TestInitCommandPrintsDetectedProject(t *testing.T) {
	t.Parallel()

	config := project.Config{
		ProjectRoot: filepath.Join("projects", "demo"),
		BuildTool: buildtool.Info{
			Type:    buildtool.Maven,
			Version: "3.9.11",
			Wrapper: buildtool.Wrapper{Enabled: true, Executable: "mvnw.cmd"},
		},
		Java: javainfo.Installation{Version: "17", Home: filepath.Join("jdks", "17")},
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
	for _, want := range []string{
		"[1/5] PROJECT - " + config.ProjectRoot,
		"[1/5] PROJECT OK - " + config.ProjectRoot,
		"Initialized javaup project.",
		"Build tool:    Maven 3.9.11",
		"Build wrapper: yes (mvnw.cmd)",
		"Java:          17",
		"Configuration: " + initializer.path,
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output %q does not contain %q", output.String(), want)
		}
	}
	if strings.Contains(output.String(), "\x1b[") {
		t.Errorf("redirected output contains ANSI escape sequences: %q", output.String())
	}
}
