package project

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/javainfo"
)

func TestConfigStoreSavesProjectJSONStructure(t *testing.T) {
	t.Parallel()

	store := NewConfigStore(filepath.Join(t.TempDir(), "projects"))
	config := Config{
		SchemaVersion: currentSchemaVersion,
		ProjectRoot:   filepath.Join("projects", "demo"),
		BuildTool: buildtool.Info{
			Type:          buildtool.Maven,
			Version:       "3.9.11",
			Executable:    filepath.Join("maven", "bin", "mvn"),
			Wrapper:       true,
			SettingsAlias: "intranet",
		},
		Java:          javainfo.Installation{Version: "17.0.12", Home: filepath.Join("jdks", "17")},
		InitializedAt: time.Date(2026, 7, 19, 1, 29, 8, 0, time.FixedZone("test", 8*60*60)),
	}

	path, err := store.Save(config)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// #nosec G304 -- path is returned by the temporary test store.
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	want := map[string]any{
		"schemaVersion": float64(1),
		"projectRoot":   filepath.Join("projects", "demo"),
		"buildTool": map[string]any{
			"type":          "maven",
			"version":       "3.9.11",
			"executable":    filepath.Join("maven", "bin", "mvn"),
			"wrapper":       true,
			"settingsAlias": "intranet",
		},
		"java": map[string]any{
			"version": "17.0.12",
			"home":    filepath.Join("jdks", "17"),
		},
		"initializedAt": "2026-07-19T01:29:08+08:00",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("saved JSON structure = %#v, want %#v", got, want)
	}
}

func TestNewDefaultConfigStoreUsesJavaupHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVAUP_HOME", home)

	store, err := NewDefaultConfigStore()
	if err != nil {
		t.Fatalf("NewDefaultConfigStore() error = %v", err)
	}
	want := filepath.Join(home, "config", "projects")
	if store.baseDir != want {
		t.Errorf("NewDefaultConfigStore() baseDir = %q, want %q", store.baseDir, want)
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

func TestConfigStoreFindsConfigurationFromProjectDescendant(t *testing.T) {
	t.Parallel()

	store := NewConfigStore(filepath.Join(t.TempDir(), "projects"))
	root := t.TempDir()
	config := Config{
		SchemaVersion: currentSchemaVersion,
		ProjectRoot:   root,
		BuildTool: buildtool.Info{
			Type:       buildtool.Maven,
			Version:    "3.9.11",
			Executable: filepath.Join(root, "mvnw"),
		},
		Java:          javainfo.Installation{Version: "21", Home: filepath.Join(root, "jdk")},
		InitializedAt: time.Now().Truncate(time.Second),
	}
	savedPath, err := store.Save(config)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	descendant := filepath.Join(root, "module", "src")
	if err := os.MkdirAll(descendant, 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	got, path, found, err := store.Find(descendant)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if !found {
		t.Fatal("Find() found = false, want true")
	}
	if path != savedPath {
		t.Errorf("Find() path = %q, want %q", path, savedPath)
	}
	if !samePath(got.ProjectRoot, root) || got.Java != config.Java {
		t.Errorf("Find() config = %#v, want %#v", got, config)
	}
}

func TestSamePathResolvesSymbolicLinks(t *testing.T) {
	t.Parallel()

	temporary := t.TempDir()
	target := filepath.Join(temporary, "target")
	if err := os.Mkdir(target, 0o750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	link := filepath.Join(temporary, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("Symlink() error = %v", err)
	}

	if !samePath(link, target) {
		t.Errorf("samePath(%q, %q) = false, want true", link, target)
	}
}

func TestConfigStoreRejectsOutdatedConfiguration(t *testing.T) {
	t.Parallel()

	store := NewConfigStore(filepath.Join(t.TempDir(), "projects"))
	root := t.TempDir()
	path, err := store.Save(Config{SchemaVersion: currentSchemaVersion - 1, ProjectRoot: root})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	_, gotPath, found, err := store.Load(root)
	if err == nil || !strings.Contains(err.Error(), "run jup init again") {
		t.Fatalf("Load() error = %v, want schema guidance", err)
	}
	if !found || gotPath != path {
		t.Errorf("Load() path/found = %q/%t, want %q/true", gotPath, found, path)
	}
}
