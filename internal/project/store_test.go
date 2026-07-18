package project

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/javainfo"
)

func TestConfigStoreSavesStableProjectJSON(t *testing.T) {
	t.Parallel()

	store := NewConfigStore(filepath.Join(t.TempDir(), "projects"))
	config := Config{
		SchemaVersion: currentSchemaVersion,
		ProjectRoot:   filepath.Join("projects", "demo"),
		BuildTool: buildtool.Info{
			Type:    buildtool.Maven,
			Version: "3.9.11",
		},
		Java:          javainfo.Installation{Version: "17", Home: filepath.Join("jdks", "17")},
		InitializedAt: NewLocalTimestamp(time.Date(2026, 7, 18, 12, 0, 0, 0, time.FixedZone("test", 8*60*60))),
	}

	firstPath, err := store.Save(config)
	if err != nil {
		t.Fatalf("Save() first error = %v", err)
	}
	secondPath, err := store.Save(config)
	if err != nil {
		t.Fatalf("Save() second error = %v", err)
	}
	if firstPath != secondPath {
		t.Errorf("Save() paths differ: %q and %q", firstPath, secondPath)
	}

	// #nosec G304 -- firstPath is returned by the temporary test store.
	content, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var saved Config
	if err := json.Unmarshal(content, &saved); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if saved.BuildTool.Version != config.BuildTool.Version || saved.Java != config.Java {
		t.Errorf("saved config = %#v, want %#v", saved, config)
	}
	if got, want := saved.InitializedAt.Format(localTimestampLayout), "2026-07-18 12:00:00"; got != want {
		t.Errorf("saved initializedAt = %q, want %q", got, want)
	}
	if !strings.Contains(string(content), `"initializedAt": "2026-07-18 12:00:00"`) {
		t.Errorf("saved JSON contains an unexpected initializedAt value: %s", content)
	}
}

func TestConfigStoreDeletesProjectConfigurationIdempotently(t *testing.T) {
	t.Parallel()

	store := NewConfigStore(filepath.Join(t.TempDir(), "projects"))
	config := Config{ProjectRoot: filepath.Join("projects", "demo")}
	savedPath, err := store.Save(config)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	deletedPath, removed, err := store.Delete(config.ProjectRoot)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !removed {
		t.Fatal("Delete() removed = false, want true")
	}
	if deletedPath != savedPath {
		t.Errorf("Delete() path = %q, want %q", deletedPath, savedPath)
	}
	if _, err := os.Stat(savedPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Stat() error = %v, want os.ErrNotExist", err)
	}

	_, removed, err = store.Delete(config.ProjectRoot)
	if err != nil {
		t.Fatalf("Delete() repeated error = %v", err)
	}
	if removed {
		t.Fatal("Delete() repeated removed = true, want false")
	}
}
