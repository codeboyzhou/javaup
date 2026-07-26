package cli

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

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
	})
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
