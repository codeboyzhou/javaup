package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/project"
)

type recordingProjectsManager struct {
	entries     []project.RegistryEntry
	warnings    []error
	listErr     error
	removeEntry project.RegistryEntry
	removed     bool
	removeErr   error
	removeRoot  string
	pruneResult project.PruneResult
	pruneErr    error
	pruneDryRun bool
}

func (m *recordingProjectsManager) List() ([]project.RegistryEntry, []error, error) {
	return m.entries, m.warnings, m.listErr
}

func (m *recordingProjectsManager) Remove(
	_ context.Context,
	root string,
) (project.RegistryEntry, bool, error) {
	m.removeRoot = root
	return m.removeEntry, m.removed, m.removeErr
}

func (m *recordingProjectsManager) Prune(
	_ context.Context,
	dryRun bool,
) (project.PruneResult, error) {
	m.pruneDryRun = dryRun
	return m.pruneResult, m.pruneErr
}

func TestProjectsListCommandShowsAllRegistryStates(t *testing.T) {
	t.Parallel()

	usedAt := time.Date(2026, 7, 25, 12, 30, 0, 0, time.Local)
	manager := &recordingProjectsManager{
		entries: []project.RegistryEntry{
			{
				Name:        "demo",
				ProjectRoot: "/projects/demo",
				BuildTool:   buildtool.Maven,
				JavaVersion: "17",
				Status:      project.RegistryAvailable,
				UseCount:    3,
				LastUsedAt:  usedAt,
			},
			{
				Name:       "broken",
				ConfigPath: "/config/broken.json",
				Status:     project.RegistryInvalid,
				Detail:     "invalid JSON",
			},
		},
		warnings: []error{errors.New("usage data unavailable")},
	}
	command := newProjectsListCommand(func() (projectsManager, error) { return manager, nil })
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	assertContains(t, stdout.String(), []string{
		"NAME", "TOOL", "JAVA", "STATUS", "USES", "LAST USED", "PATH",
		"demo", "Maven", "17", "available", "3", "/projects/demo",
		"broken", "invalid", "/config/broken.json",
	})
	assertContains(t, stderr.String(), []string{
		"jup: warning: usage data unavailable",
		"jup: warning: /config/broken.json: invalid JSON",
		"jup: hint: run `jup projects prune` to remove missing or invalid project records.",
	})
}

func TestWriteProjectsTableHighlightsProblemRowsInYellow(t *testing.T) {
	t.Parallel()

	output := openProjectsOutputFile(t)
	entries := []project.RegistryEntry{
		{Name: "demo", ProjectRoot: "/projects/demo", Status: project.RegistryAvailable},
		{Name: "missing", ProjectRoot: "/projects/missing", Status: project.RegistryMissing},
	}
	if err := writeProjectsTableWithStyle(output, entries, enabledYellow()); err != nil {
		t.Fatalf("writeProjectsTable() error = %v", err)
	}

	contents := readProjectsOutputFile(t, output)
	if strings.Contains(contents, "\x1b[33mdemo\x1b[0m") {
		t.Errorf("available row was highlighted:\n%s", contents)
	}
	for _, value := range []string{"missing", "-", "0", "/projects/missing"} {
		if !strings.Contains(contents, "\x1b[33m"+value+"\x1b[0m") {
			t.Errorf("problem row value %q was not highlighted:\n%s", value, contents)
		}
	}
}

func TestWriteProjectsWarningsUsesYellowAndSuggestsPrune(t *testing.T) {
	t.Parallel()

	output := openProjectsOutputFile(t)
	writeProjectsWarningsWithStyle(
		output,
		[]error{errors.New("usage data unavailable")},
		[]project.RegistryEntry{{
			ProjectRoot: "/projects/missing",
			Status:      project.RegistryMissing,
			Detail:      "directory does not exist",
		}},
		enabledYellow(),
		enabledColor(color.FgGreen),
	)

	contents := readProjectsOutputFile(t, output)
	for _, line := range []string{
		"jup: warning: usage data unavailable",
		"jup: warning: /projects/missing: directory does not exist",
	} {
		if !strings.Contains(contents, "\x1b[33m"+line+"\x1b[0m") {
			t.Errorf("line %q was not highlighted:\n%s", line, contents)
		}
	}
	hint := "jup: hint: run `jup projects prune` to remove missing or invalid project records."
	if !strings.Contains(contents, "\x1b[32m"+hint+"\x1b[0m") {
		t.Errorf("hint %q was not highlighted in green:\n%s", hint, contents)
	}
}

func enabledYellow() *color.Color {
	return enabledColor(color.FgYellow)
}

func enabledColor(attribute color.Attribute) *color.Color {
	style := color.New(attribute)
	style.EnableColor()
	return style
}

func openProjectsOutputFile(t *testing.T) *os.File {
	t.Helper()
	output, err := os.CreateTemp(t.TempDir(), "projects-output-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	t.Cleanup(func() { _ = output.Close() })
	return output
}

func readProjectsOutputFile(t *testing.T, output *os.File) string {
	t.Helper()
	if _, err := output.Seek(0, 0); err != nil {
		t.Fatalf("Seek() error = %v", err)
	}
	contents, err := os.ReadFile(output.Name())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	return string(contents)
}

func TestProjectsRemoveCommandUsesExplicitPath(t *testing.T) {
	t.Parallel()

	root := filepath.Join("projects", "demo")
	manager := &recordingProjectsManager{
		removeEntry: project.RegistryEntry{ProjectRoot: root},
		removed:     true,
	}
	command := newProjectsRemoveCommand(func() (projectsManager, error) { return manager, nil })
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{root})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if manager.removeRoot != root {
		t.Errorf("Remove() root = %q, want %q", manager.removeRoot, root)
	}
	assertContains(t, output.String(), []string{"Removed project " + root})
}

func TestProjectsPruneCommandSupportsDryRun(t *testing.T) {
	t.Parallel()

	manager := &recordingProjectsManager{pruneResult: project.PruneResult{
		DryRun: true,
		Projects: []project.RegistryEntry{{
			ProjectRoot: "/projects/missing",
			Status:      project.RegistryMissing,
		}},
		UsageRecords: 2,
	}}
	command := newProjectsPruneCommand(func() (projectsManager, error) { return manager, nil })
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"--dry-run"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if !manager.pruneDryRun {
		t.Fatal("Prune() dryRun = false, want true")
	}
	assertContains(t, output.String(), []string{
		"Would remove missing project record /projects/missing.",
		"Would remove 2 orphaned usage records.",
		"Dry run complete; no files were changed.",
	})
}
