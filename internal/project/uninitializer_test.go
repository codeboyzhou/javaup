package project

import (
	"context"
	"path/filepath"
	"testing"
)

type fakeConfigRemover struct {
	root    string
	path    string
	removed bool
}

func (r *fakeConfigRemover) Delete(root string) (string, bool, error) {
	r.root = root
	return r.path, r.removed, nil
}

func TestUninitializerRemovesCurrentProjectConfiguration(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := &fakeConfigRemover{path: filepath.Join("config", "project.json"), removed: true}
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
	if lastEvent.State != ProgressSucceeded || lastEvent.Message != "No saved configuration found" {
		t.Errorf("last progress event = %#v", lastEvent)
	}
}
