package project

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type fakeConfigRemover struct {
	config  Config
	found   bool
	start   string
	root    string
	path    string
	removed bool
}

func (r *fakeConfigRemover) Find(start string) (Config, string, bool, error) {
	r.start = start
	return r.config, r.path, r.found, nil
}

func (r *fakeConfigRemover) Delete(root string) (string, bool, error) {
	r.root = root
	return r.path, r.removed, nil
}

func TestUninitializerRemovesCurrentProjectConfiguration(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := &fakeConfigRemover{
		config:  Config{ProjectRoot: root},
		found:   true,
		path:    filepath.Join("config", "project.json"),
		removed: true,
	}
	uninitializer := NewUninitializer(store)
	var events []ProgressEvent

	path, removed, err := uninitializer.Uninitialize(context.Background(), root, func(event ProgressEvent) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatalf("Uninitialize() error = %v", err)
	}
	if !removed {
		t.Fatal("Uninitialize() removed = false, want true")
	}
	if path != store.path {
		t.Errorf("Uninitialize() path = %q, want %q", path, store.path)
	}
	wantRoot, err := canonicalProjectRoot(root)
	if err != nil {
		t.Fatalf("canonicalProjectRoot() error = %v", err)
	}
	if store.root != wantRoot {
		t.Errorf("Delete() root = %q, want %q", store.root, wantRoot)
	}
	if store.start != wantRoot {
		t.Errorf("Find() start = %q, want %q", store.start, wantRoot)
	}

	wantEvents := []ProgressEvent{
		{Step: 1, Total: 2, Name: projectStepName, Message: "Resolving current project directory", State: ProgressStarted},
		{Step: 1, Total: 2, Name: projectStepName, Message: wantRoot, State: ProgressSucceeded},
		{Step: 2, Total: 2, Name: configStepName, Message: "Removing local project configuration", State: ProgressStarted},
		{Step: 2, Total: 2, Name: configStepName, Message: store.path, State: ProgressSucceeded},
	}
	if len(events) != len(wantEvents) {
		t.Fatalf("progress event count = %d, want %d", len(events), len(wantEvents))
	}
	for index, want := range wantEvents {
		if events[index] != want {
			t.Errorf("progress event %d = %#v, want %#v", index, events[index], want)
		}
	}
}

func TestUninitializerSucceedsWhenConfigurationDoesNotExist(t *testing.T) {
	t.Parallel()

	store := &fakeConfigRemover{path: filepath.Join("config", "project.json")}
	uninitializer := NewUninitializer(store)
	var lastEvent ProgressEvent

	_, removed, err := uninitializer.Uninitialize(context.Background(), t.TempDir(), func(event ProgressEvent) {
		lastEvent = event
	})
	if err != nil {
		t.Fatalf("Uninitialize() error = %v", err)
	}
	if removed {
		t.Fatal("Uninitialize() removed = true, want false")
	}
	if store.root != "" {
		t.Errorf("Delete() root = %q, want no call", store.root)
	}
	if lastEvent.State != ProgressSucceeded || lastEvent.Message != "No saved configuration found" {
		t.Errorf("last progress event = %#v", lastEvent)
	}
}

func TestUninitializerRemovesProjectConfigurationFromDescendant(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	descendant := filepath.Join(root, "module", "src")
	if err := os.MkdirAll(descendant, 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	store := NewConfigStore(filepath.Join(t.TempDir(), "projects"))
	if _, err := store.Save(Config{
		SchemaVersion: currentSchemaVersion,
		ProjectRoot:   root,
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	uninitializer := NewUninitializer(store)
	_, removed, err := uninitializer.Uninitialize(context.Background(), descendant, nil)
	if err != nil {
		t.Fatalf("Uninitialize() error = %v", err)
	}
	if !removed {
		t.Fatal("Uninitialize() removed = false, want true")
	}
	_, _, found, err := store.Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if found {
		t.Fatal("Load() found = true, want removed parent project configuration")
	}
}
