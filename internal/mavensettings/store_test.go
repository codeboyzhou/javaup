package mavensettings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreAddsAndUpdatesAliases(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	registryPath := filepath.Join(root, "config", "settings.json")
	firstPath := writeSettingsFile(t, filepath.Join(root, "settings-intranet.xml"))
	secondPath := writeSettingsFile(t, filepath.Join(root, "settings-google.xml"))
	wantFirstPath := canonicalSettingsPath(t, firstPath)
	wantSecondPath := canonicalSettingsPath(t, secondPath)
	store := NewStore(registryPath)

	entry, savedPath, err := store.Add("intranet", firstPath)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if entry.Alias != "intranet" || entry.Path != wantFirstPath {
		t.Errorf("Add() entry = %#v, want intranet/%s", entry, wantFirstPath)
	}
	if savedPath != registryPath {
		t.Errorf("Add() registry path = %q, want %q", savedPath, registryPath)
	}

	if _, _, err := store.Add("google", secondPath); err != nil {
		t.Fatalf("Add() second alias error = %v", err)
	}
	if _, _, err := store.Add("intranet", secondPath); err != nil {
		t.Fatalf("Add() update alias error = %v", err)
	}

	content, err := os.ReadFile(registryPath) // #nosec G304 -- path belongs to the temporary test store.
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var got registry
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.SchemaVersion != currentSchemaVersion {
		t.Errorf("schema version = %d, want %d", got.SchemaVersion, currentSchemaVersion)
	}
	if len(got.Aliases) != 2 || got.Aliases["intranet"] != wantSecondPath || got.Aliases["google"] != wantSecondPath {
		t.Errorf("aliases = %#v, want updated intranet and google mappings", got.Aliases)
	}

	entry, err = store.Resolve("intranet")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if entry.Alias != "intranet" || entry.Path != wantSecondPath {
		t.Errorf("Resolve() entry = %#v, want intranet/%s", entry, wantSecondPath)
	}

	entries, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 2 || entries[0].Alias != "google" || entries[1].Alias != "intranet" {
		t.Errorf("List() entries = %#v, want aliases ordered as google, intranet", entries)
	}
	if err := os.Remove(secondPath); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	entries, err = store.List()
	if err != nil || len(entries) != 2 {
		t.Errorf("List() stale entries/error = %#v/%v, want both saved mappings", entries, err)
	}
}

func TestNewDefaultStoreUsesJavaupHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVAUP_HOME", home)

	store, err := NewDefaultStore()
	if err != nil {
		t.Fatalf("NewDefaultStore() error = %v", err)
	}
	want := filepath.Join(home, "config", "maven", "settings.json")
	if store.path != want {
		t.Errorf("NewDefaultStore() path = %q, want %q", store.path, want)
	}
}

func TestStoreRejectsUnknownAlias(t *testing.T) {
	t.Parallel()

	_, err := NewStore(filepath.Join(t.TempDir(), "settings.json")).Resolve("missing")
	if err == nil || !strings.Contains(err.Error(), "jup settings add") {
		t.Fatalf("Resolve() error = %v, want add alias guidance", err)
	}
}

func TestStoreRemovesAlias(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	settingsPath := writeSettingsFile(t, filepath.Join(root, "settings.xml"))
	googlePath := writeSettingsFile(t, filepath.Join(root, "settings-google.xml"))
	wantSettingsPath := canonicalSettingsPath(t, settingsPath)
	wantGooglePath := canonicalSettingsPath(t, googlePath)
	store := NewStore(filepath.Join(root, "config", "settings.json"))
	if _, _, err := store.Add("intranet", settingsPath); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if _, _, err := store.Add("google", googlePath); err != nil {
		t.Fatalf("Add() second alias error = %v", err)
	}
	if err := os.Remove(settingsPath); err != nil {
		t.Fatalf("Remove settings file error = %v", err)
	}

	entry, err := store.Remove("intranet")
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if entry.Alias != "intranet" || entry.Path != wantSettingsPath {
		t.Errorf("Remove() entry = %#v, want intranet/%s", entry, wantSettingsPath)
	}
	entries, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Alias != "google" || entries[0].Path != wantGooglePath {
		t.Errorf("List() entries = %#v, want only google mapping", entries)
	}
	if _, err := store.Remove("intranet"); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Errorf("Remove() missing error = %v, want not configured", err)
	}
}

func TestStoreRejectsInvalidAlias(t *testing.T) {
	t.Parallel()

	settingsPath := writeSettingsFile(t, filepath.Join(t.TempDir(), "settings.xml"))
	_, _, err := NewStore(filepath.Join(t.TempDir(), "settings.json")).Add("invalid alias", settingsPath)
	if err == nil || !strings.Contains(err.Error(), "must contain only") {
		t.Fatalf("Add() error = %v, want invalid alias error", err)
	}
}

func TestStoreRejectsInvalidMavenSettingsFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "not-settings.xml")
	if err := os.WriteFile(path, []byte(`<project/>`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, _, err := NewStore(filepath.Join(root, "settings.json")).Add("invalid", path)
	if err == nil || !strings.Contains(err.Error(), `root element "project"`) {
		t.Fatalf("Add() error = %v, want invalid root element error", err)
	}
}

func writeSettingsFile(t *testing.T, path string) string {
	t.Helper()
	content := `<settings xmlns="http://maven.apache.org/SETTINGS/1.2.0"><mirrors/></settings>`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}
	return filepath.Clean(absolutePath)
}

func canonicalSettingsPath(t *testing.T, path string) string {
	t.Helper()
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks() error = %v", err)
	}
	return filepath.Clean(resolvedPath)
}
