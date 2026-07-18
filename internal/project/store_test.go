package project

import (
	"encoding/json"
	"os"
	"path/filepath"
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
		InitializedAt: time.Date(2026, 7, 18, 4, 0, 0, 0, time.UTC),
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
}
