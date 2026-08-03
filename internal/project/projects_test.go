package project

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/javainfo"
	"github.com/codeboyzhou/javaup/internal/pathutil"
)

func TestProjectManagerListsAvailableMissingAndInvalidProjects(t *testing.T) {
	t.Parallel()

	configs := NewConfigStore(filepath.Join(t.TempDir(), "projects"))
	usage := NewUsageStore(filepath.Join(t.TempDir(), "state", "project-usage.json"))
	availableRoot := t.TempDir()
	missingRoot := filepath.Join(t.TempDir(), "missing")
	saveProjectConfig(t, configs, availableRoot, "17")
	saveProjectConfig(t, configs, missingRoot, "8")
	invalidPath := filepath.Join(configs.baseDir, "broken.json")
	writeProjectRegistryFile(t, invalidPath, "{not-json")
	usedAt := time.Date(2026, 7, 25, 12, 30, 0, 0, time.UTC)
	if err := usage.Touch(context.Background(), availableRoot, usedAt); err != nil {
		t.Fatalf("Touch() error = %v", err)
	}

	entries, warnings, err := NewRegistry(configs, usage).List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3: %#v", len(entries), entries)
	}

	available := findProjectEntry(t, entries, RegistryAvailable)
	if available.ProjectRoot != availableRoot {
		t.Errorf("available root = %q, want %q", available.ProjectRoot, availableRoot)
	}
	if available.JavaVersion != "17" || available.UseCount != 1 || !available.LastUsedAt.Equal(usedAt) {
		t.Errorf("available entry = %#v", available)
	}
	missing := findProjectEntry(t, entries, RegistryMissing)
	if missing.ProjectRoot != missingRoot || missing.Detail == "" {
		t.Errorf("missing entry = %#v", missing)
	}
	invalid := findProjectEntry(t, entries, RegistryInvalid)
	if invalid.ConfigPath != invalidPath || invalid.ProjectRoot != "" || invalid.Detail == "" {
		t.Errorf("invalid entry = %#v", invalid)
	}
}

func TestProjectManagerRemovesProjectAndUsageByPath(t *testing.T) {
	t.Parallel()

	configs := NewConfigStore(filepath.Join(t.TempDir(), "projects"))
	usagePath := filepath.Join(t.TempDir(), "state", "project-usage.json")
	usage := NewUsageStore(usagePath)
	root := t.TempDir()
	configPath := saveProjectConfig(t, configs, root, "17")
	if err := usage.Touch(context.Background(), root, time.Now()); err != nil {
		t.Fatalf("Touch() error = %v", err)
	}
	manager := NewRegistry(configs, usage)

	entry, removed, err := manager.Remove(context.Background(), root)
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if !removed {
		t.Fatal("Remove() removed = false, want true")
	}
	if entry.ProjectRoot != root || entry.ConfigPath != configPath {
		t.Errorf("entry = %#v, want root/path %q/%q", entry, root, configPath)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Errorf("Stat(config) error = %v, want removed", err)
	}
	if _, err := os.Stat(usagePath); !os.IsNotExist(err) {
		t.Errorf("Stat(usage) error = %v, want removed", err)
	}

	_, removed, err = manager.Remove(context.Background(), root)
	if err != nil {
		t.Fatalf("Remove(second) error = %v", err)
	}
	if removed {
		t.Fatal("Remove(second) removed = true, want idempotent false")
	}
}

func TestProjectManagerPruneDryRunAndRemoval(t *testing.T) {
	t.Parallel()

	configs := NewConfigStore(filepath.Join(t.TempDir(), "projects"))
	usage := NewUsageStore(filepath.Join(t.TempDir(), "state", "project-usage.json"))
	availableRoot := t.TempDir()
	missingRoot := filepath.Join(t.TempDir(), "missing")
	orphanRoot := t.TempDir()
	availableConfig := saveProjectConfig(t, configs, availableRoot, "17")
	missingConfig := saveProjectConfig(t, configs, missingRoot, "8")
	invalidConfig := filepath.Join(configs.baseDir, "broken.json")
	writeProjectRegistryFile(t, invalidConfig, "{broken")
	for _, root := range []string{availableRoot, missingRoot, orphanRoot} {
		if err := usage.Touch(context.Background(), root, time.Now()); err != nil {
			t.Fatalf("Touch(%s) error = %v", root, err)
		}
	}
	manager := NewRegistry(configs, usage)

	preview, err := manager.Prune(context.Background(), true)
	if err != nil {
		t.Fatalf("Prune(dry run) error = %v", err)
	}
	if !preview.DryRun || len(preview.Projects) != 2 || preview.UsageRecords != 2 {
		t.Fatalf("preview = %#v, want 2 projects and 2 usage records", preview)
	}
	for _, path := range []string{availableConfig, missingConfig, invalidConfig} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("dry run removed %s: %v", path, err)
		}
	}

	result, err := manager.Prune(context.Background(), false)
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	if result.DryRun || len(result.Projects) != 2 || result.UsageRecords != 2 {
		t.Fatalf("result = %#v, want 2 projects and 2 usage records", result)
	}
	if _, err := os.Stat(availableConfig); err != nil {
		t.Errorf("available config was removed: %v", err)
	}
	for _, path := range []string{missingConfig, invalidConfig} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("Stat(%s) error = %v, want removed", path, err)
		}
	}
	records, err := usage.Load()
	if err != nil {
		t.Fatalf("usage Load() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("usage records = %d, want 1", len(records))
	}
	if _, exists := records[pathutil.Identity(availableRoot)]; !exists {
		t.Errorf("available usage was removed: %#v", records)
	}
}

func saveProjectConfig(t *testing.T, store *ConfigStore, root, javaVersion string) string {
	t.Helper()
	path, err := store.Save(Config{
		SchemaVersion: currentSchemaVersion,
		ProjectRoot:   root,
		BuildTool: buildtool.Info{
			Type:       buildtool.Maven,
			Version:    "3.9.11",
			Executable: filepath.Join(root, "mvn"),
		},
		Java: javainfo.Installation{Version: javaVersion, Home: filepath.Join(root, "jdk")},
	})
	if err != nil {
		t.Fatalf("Save(%s) error = %v", root, err)
	}
	return path
}

func writeProjectRegistryFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func findProjectEntry(t *testing.T, entries []RegistryEntry, status RegistryStatus) RegistryEntry {
	t.Helper()
	for _, entry := range entries {
		if entry.Status == status {
			return entry
		}
	}
	t.Fatalf("status %q not found in %#v", status, entries)
	return RegistryEntry{}
}
