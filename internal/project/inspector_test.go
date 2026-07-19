package project

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codeboyzhou/javaup/internal/buildtool"
)

type fakeInspectionStore struct {
	config Config
	found  bool
	err    error
	start  string
}

func (s *fakeInspectionStore) Find(start string) (Config, string, bool, error) {
	s.start = start
	return s.config, "config.json", s.found, s.err
}

func TestInspectorReturnsContainingProjectConfiguration(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	want := Config{
		ProjectRoot: root,
		BuildTool:   buildtool.Info{Type: buildtool.Maven, Version: "3.9.11"},
	}
	store := &fakeInspectionStore{config: want, found: true}
	inspector := NewInspector(store)
	start := filepath.Join(root, "module")

	got, err := inspector.Inspect(start)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if got != want {
		t.Errorf("Inspect() config = %#v, want %#v", got, want)
	}
	if store.start != start {
		t.Errorf("Find() start = %q, want %q", store.start, start)
	}
}

func TestInspectorRequiresInitializedProject(t *testing.T) {
	t.Parallel()

	_, err := NewInspector(&fakeInspectionStore{}).Inspect(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "run jup init") {
		t.Fatalf("Inspect() error = %v, want initialization guidance", err)
	}
}

func TestInspectorPropagatesStoreError(t *testing.T) {
	t.Parallel()

	want := errors.New("store failure")
	_, err := NewInspector(&fakeInspectionStore{err: want}).Inspect(t.TempDir())
	if !errors.Is(err, want) {
		t.Fatalf("Inspect() error = %v, want %v", err, want)
	}
}
