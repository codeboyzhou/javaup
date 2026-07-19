package mavensettings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreAddsAndUpdatesMavenSettingsAliases(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	registryPath := filepath.Join(root, "config", "settings.json")
	firstPath := writeSettingsFile(t, filepath.Join(root, "settings-intranet.xml"))
	secondPath := writeSettingsFile(t, filepath.Join(root, "settings-google.xml"))
	store := NewStore(registryPath)

	entry, savedPath, err := store.Add("intranet", firstPath)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if entry.Alias != "intranet" || entry.Path != firstPath {
		t.Errorf("Add() entry = %#v, want intranet/%s", entry, firstPath)
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
	if len(got.Aliases) != 2 || got.Aliases["intranet"] != secondPath || got.Aliases["google"] != secondPath {
		t.Errorf("aliases = %#v, want updated intranet and google mappings", got.Aliases)
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
