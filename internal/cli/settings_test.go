package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/codeboyzhou/javaup/internal/mavensettings"
	"github.com/codeboyzhou/javaup/internal/project"
)

type recordingMavenSettingsStore struct {
	alias        string
	path         string
	registryPath string
	entries      []mavensettings.Entry
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

func (s *recordingMavenSettingsStore) Add(alias, path string) (mavensettings.Entry, string, error) {
	s.alias = alias
	s.path = path
	return mavensettings.Entry{Alias: alias, Path: path}, s.registryPath, nil
}

func (s *recordingMavenSettingsStore) List() ([]mavensettings.Entry, error) {
	return append([]mavensettings.Entry(nil), s.entries...), nil
}

func TestSettingsAddCommandSavesAlias(t *testing.T) {
	t.Parallel()

	adder := &recordingMavenSettingsStore{registryPath: "/config/javaup/maven/settings.json"}
	command := newSettingsCommand(
		func() (mavenSettingsStore, error) { return adder, nil },
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

	command := newSettingsCommand(func() (mavenSettingsStore, error) {
		return &recordingMavenSettingsStore{}, nil
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
		func() (mavenSettingsStore, error) { return &recordingMavenSettingsStore{}, nil },
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

func TestSettingsListCommandPrintsAliasTable(t *testing.T) {
	t.Parallel()

	store := &recordingMavenSettingsStore{entries: []mavensettings.Entry{
		{Alias: "google", Path: "/maven/settings-google.xml"},
		{Alias: "intranet", Path: "/maven/settings-intranet.xml"},
	}}
	command := newSettingsCommand(
		func() (mavenSettingsStore, error) { return store, nil },
		func() (projectMavenSettingsUser, error) { return &recordingProjectMavenSettingsUser{}, nil },
		func() (string, error) { return "/projects/demo", nil },
	)
	command.SetArgs([]string{"list"})
	var output bytes.Buffer
	command.SetOut(&output)

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(normalizedOutput(output.String())), "\n")
	if len(lines) != 6 {
		t.Fatalf("output lines = %#v, want bordered header and two aliases", lines)
	}
	if lines[0] != lines[2] || lines[0] != lines[5] || !strings.HasPrefix(lines[0], "+-") {
		t.Errorf("table borders = %q/%q/%q, want matching dashed borders", lines[0], lines[2], lines[5])
	}
	if !strings.Contains(lines[1], "| ALIAS") || !strings.Contains(lines[1], "| PATH") {
		t.Errorf("header = %q, want ALIAS and PATH columns", lines[1])
	}
	if !strings.Contains(lines[3], "| google") || !strings.Contains(lines[3], "/maven/settings-google.xml") {
		t.Errorf("first row = %q, want google mapping", lines[3])
	}
	if !strings.Contains(lines[4], "| intranet") || !strings.Contains(lines[4], "/maven/settings-intranet.xml") {
		t.Errorf("second row = %q, want intranet mapping", lines[4])
	}
}

func TestSettingsListCommandHandlesEmptyStore(t *testing.T) {
	t.Parallel()

	command := newSettingsCommand(
		func() (mavenSettingsStore, error) { return &recordingMavenSettingsStore{}, nil },
		func() (projectMavenSettingsUser, error) { return &recordingProjectMavenSettingsUser{}, nil },
		func() (string, error) { return "/projects/demo", nil },
	)
	command.SetArgs([]string{"list"})
	var output bytes.Buffer
	command.SetOut(&output)

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if got, want := normalizedOutput(output.String()), "No Maven settings aliases configured.\n"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}
