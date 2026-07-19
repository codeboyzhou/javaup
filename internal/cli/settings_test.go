package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/codeboyzhou/javaup/internal/mavensettings"
	"github.com/codeboyzhou/javaup/internal/project"
)

type recordingMavenSettingsAdder struct {
	alias        string
	path         string
	registryPath string
}

type recordingProjectMavenSettingsUser struct {
	root  string
	alias string
}

func (u *recordingProjectMavenSettingsUser) Use(
	root, alias string,
) (project.Config, mavensettings.Entry, error) {
	u.root = root
	u.alias = alias
	return project.Config{ProjectRoot: "/projects/demo"}, mavensettings.Entry{
		Alias: alias,
		Path:  "/maven/settings-intranet.xml",
	}, nil
}

func (a *recordingMavenSettingsAdder) Add(alias, path string) (mavensettings.Entry, string, error) {
	a.alias = alias
	a.path = path
	return mavensettings.Entry{Alias: alias, Path: path}, a.registryPath, nil
}

func TestSettingsAddCommandSavesAlias(t *testing.T) {
	t.Parallel()

	adder := &recordingMavenSettingsAdder{registryPath: "/config/javaup/maven/settings.json"}
	command := newSettingsCommand(
		func() (mavenSettingsAdder, error) { return adder, nil },
		func() (projectMavenSettingsUser, error) { return &recordingProjectMavenSettingsUser{}, nil },
		func() (string, error) { return "/projects/demo", nil },
	)
	command.SetArgs([]string{"add", "intranet", "/maven/settings-intranet.xml"})
	var output bytes.Buffer
	command.SetOut(&output)

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if adder.alias != "intranet" || adder.path != "/maven/settings-intranet.xml" {
		t.Errorf("Add() arguments = %q/%q, want intranet settings path", adder.alias, adder.path)
	}
	const want = "Saved Maven settings alias \"intranet\" for /maven/settings-intranet.xml " +
		"in /config/javaup/maven/settings.json.\n"
	if got := normalizedOutput(output.String()); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestSettingsAddCommandRequiresAliasAndPath(t *testing.T) {
	t.Parallel()

	command := newSettingsCommand(func() (mavenSettingsAdder, error) {
		return &recordingMavenSettingsAdder{}, nil
	}, func() (projectMavenSettingsUser, error) {
		return &recordingProjectMavenSettingsUser{}, nil
	}, func() (string, error) { return "/projects/demo", nil })
	command.SetArgs([]string{"add", "intranet"})

	if err := command.ExecuteContext(context.Background()); err == nil {
		t.Fatal("ExecuteContext() error = nil, want argument validation error")
	}
}

func TestSettingsUseCommandAssociatesCurrentProject(t *testing.T) {
	t.Parallel()

	user := &recordingProjectMavenSettingsUser{}
	command := newSettingsCommand(
		func() (mavenSettingsAdder, error) { return &recordingMavenSettingsAdder{}, nil },
		func() (projectMavenSettingsUser, error) { return user, nil },
		func() (string, error) { return "/projects/demo/module", nil },
	)
	command.SetArgs([]string{"use", "intranet"})
	var output bytes.Buffer
	command.SetOut(&output)

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if user.root != "/projects/demo/module" || user.alias != "intranet" {
		t.Errorf("Use() arguments = %q/%q, want current project and intranet", user.root, user.alias)
	}
	const want = "Configured project /projects/demo to use Maven settings alias \"intranet\" " +
		"at /maven/settings-intranet.xml.\n"
	if got := normalizedOutput(output.String()); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}
