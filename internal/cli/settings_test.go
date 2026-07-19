package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/codeboyzhou/javaup/internal/mavensettings"
)

type recordingMavenSettingsAdder struct {
	alias        string
	path         string
	registryPath string
}

func (a *recordingMavenSettingsAdder) Add(alias, path string) (mavensettings.Entry, string, error) {
	a.alias = alias
	a.path = path
	return mavensettings.Entry{Alias: alias, Path: path}, a.registryPath, nil
}

func TestSettingsAddCommandSavesAlias(t *testing.T) {
	t.Parallel()

	adder := &recordingMavenSettingsAdder{registryPath: "/config/javaup/maven/settings.json"}
	command := newSettingsCommand(func() (mavenSettingsAdder, error) { return adder, nil })
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
	})
	command.SetArgs([]string{"add", "intranet"})

	if err := command.ExecuteContext(context.Background()); err == nil {
		t.Fatal("ExecuteContext() error = nil, want argument validation error")
	}
}
