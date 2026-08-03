package project

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/pathutil"
)

func TestUsageStoreAppliesFourteenDayHalfLife(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewUsageStore(filepath.Join(t.TempDir(), "state", "project-usage.json"))
	started := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	if err := store.Touch(context.Background(), root, started); err != nil {
		t.Fatalf("Touch(first) error = %v", err)
	}
	if err := store.Touch(context.Background(), root, started.Add(usageHalfLife)); err != nil {
		t.Fatalf("Touch(second) error = %v", err)
	}

	usage, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	canonicalRoot, err := pathutil.Canonical(root)
	if err != nil {
		t.Fatalf("pathutil.Canonical() error = %v", err)
	}
	record := usage[pathutil.Identity(canonicalRoot)]
	if record.UseCount != 2 {
		t.Errorf("UseCount = %d, want 2", record.UseCount)
	}
	if math.Abs(record.Score-1.5) > 0.000001 {
		t.Errorf("Score = %f, want 1.5", record.Score)
	}
}

func TestCatalogRanksByDecayedFrequencyAndFiltersBuildTool(t *testing.T) {
	t.Parallel()

	baseDir := filepath.Join(t.TempDir(), "projects")
	configs := NewConfigStore(baseDir)
	recentRoot := filepath.Join(t.TempDir(), "same-name")
	frequentRoot := filepath.Join(t.TempDir(), "same-name")
	otherRoot := filepath.Join(t.TempDir(), "other")
	for _, root := range []string{recentRoot, frequentRoot, otherRoot} {
		if err := os.Mkdir(root, 0o750); err != nil {
			t.Fatalf("Mkdir(%s) error = %v", root, err)
		}
		tool := buildtool.Maven
		if root == otherRoot {
			tool = buildtool.Type("gradle")
		}
		if _, err := configs.Save(Config{
			SchemaVersion: currentSchemaVersion,
			ProjectRoot:   root,
			BuildTool:     buildtool.Info{Type: tool},
		}); err != nil {
			t.Fatalf("Save(%s) error = %v", root, err)
		}
	}

	usage := NewUsageStore(filepath.Join(t.TempDir(), "state", "project-usage.json"))
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	if err := usage.Touch(context.Background(), recentRoot, now.Add(-time.Hour)); err != nil {
		t.Fatalf("Touch(recent) error = %v", err)
	}
	for range 3 {
		if err := usage.Touch(context.Background(), frequentRoot, now.Add(-7*24*time.Hour)); err != nil {
			t.Fatalf("Touch(frequent) error = %v", err)
		}
	}

	catalog := NewCatalog(configs, usage)
	catalog.now = func() time.Time { return now }
	candidates, warnings, err := catalog.List(buildtool.Maven)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("List() warnings = %v", warnings)
	}
	if len(candidates) != 2 {
		t.Fatalf("List() candidates = %d, want 2", len(candidates))
	}
	if !pathutil.Same(candidates[0].ProjectRoot, frequentRoot) || !pathutil.Same(candidates[1].ProjectRoot, recentRoot) {
		t.Errorf("List() order = %q, %q; want frequent then recent", candidates[0].ProjectRoot, candidates[1].ProjectRoot)
	}
	if candidates[0].Name != candidates[1].Name {
		t.Errorf("candidate names = %q/%q, want duplicate display names", candidates[0].Name, candidates[1].Name)
	}
}

func TestUsageStoreDeleteRemovesProjectRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(t.TempDir(), "state", "project-usage.json")
	store := NewUsageStore(path)
	if err := store.Touch(context.Background(), root, time.Now()); err != nil {
		t.Fatalf("Touch() error = %v", err)
	}
	if err := store.Delete(context.Background(), root); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Stat() error = %v, want usage file removed", err)
	}
}

func TestUsageStorePruneSupportsDryRun(t *testing.T) {
	t.Parallel()

	keepRoot := t.TempDir()
	removeRoot := t.TempDir()
	path := filepath.Join(t.TempDir(), "state", "project-usage.json")
	store := NewUsageStore(path)
	for _, root := range []string{keepRoot, removeRoot} {
		if err := store.Touch(context.Background(), root, time.Now()); err != nil {
			t.Fatalf("Touch(%s) error = %v", root, err)
		}
	}
	keep := map[string]struct{}{pathutil.Identity(keepRoot): {}}

	removed, err := store.Prune(context.Background(), keep, true)
	if err != nil {
		t.Fatalf("Prune(dry run) error = %v", err)
	}
	if removed != 1 {
		t.Fatalf("Prune(dry run) removed = %d, want 1", removed)
	}
	records, err := store.Load()
	if err != nil {
		t.Fatalf("Load(after dry run) error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("records after dry run = %d, want 2", len(records))
	}

	removed, err = store.Prune(context.Background(), keep, false)
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	if removed != 1 {
		t.Fatalf("Prune() removed = %d, want 1", removed)
	}
	records, err = store.Load()
	if err != nil {
		t.Fatalf("Load(after prune) error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records after prune = %d, want 1", len(records))
	}
	if _, exists := records[pathutil.Identity(keepRoot)]; !exists {
		t.Errorf("kept record is missing: %#v", records)
	}
}
