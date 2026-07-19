package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/mavensettings"
)

type fakeSettingsProjectStore struct {
	config Config
	found  bool
	start  string
	saved  Config
	saves  int
}

func (s *fakeSettingsProjectStore) Find(start string) (Config, string, bool, error) {
	s.start = start
	return s.config, "project.json", s.found, nil
}

func (s *fakeSettingsProjectStore) Save(config Config) (string, error) {
	s.saved = config
	s.saves++
	return "project.json", nil
}

func TestMavenSettingsManagerUnsetsAliasFromCurrentProject(t *testing.T) {
	t.Parallel()

	store := &fakeSettingsProjectStore{
		found: true,
		config: Config{
			ProjectRoot: "/projects/demo",
			BuildTool: buildtool.Info{
				Type:          buildtool.Maven,
				SettingsAlias: "intranet",
			},
		},
	}
	manager := NewMavenSettingsManager(store, &fakeMavenSettingsResolver{})

	config, previousAlias, err := manager.Unset("/projects/demo/module")
	if err != nil {
		t.Fatalf("Unset() error = %v", err)
	}
	if store.start != "/projects/demo/module" {
		t.Errorf("Unset() lookup root = %q, want current project", store.start)
	}
	if previousAlias != "intranet" {
		t.Errorf("Unset() previous alias = %q, want intranet", previousAlias)
	}
	if config.BuildTool.SettingsAlias != "" || store.saved.BuildTool.SettingsAlias != "" {
		t.Errorf("saved Maven settings alias = %q, want empty", store.saved.BuildTool.SettingsAlias)
	}
}

func TestMavenSettingsManagerUnsetIsIdempotent(t *testing.T) {
	t.Parallel()

	store := &fakeSettingsProjectStore{
		found: true,
		config: Config{
			ProjectRoot: "/projects/demo",
			BuildTool:   buildtool.Info{Type: buildtool.Maven},
		},
	}
	manager := NewMavenSettingsManager(store, &fakeMavenSettingsResolver{})

	config, previousAlias, err := manager.Unset("/projects/demo")
	if err != nil {
		t.Fatalf("Unset() error = %v", err)
	}
	if previousAlias != "" || config.BuildTool.SettingsAlias != "" {
		t.Errorf("Unset() aliases = %q/%q, want empty", previousAlias, config.BuildTool.SettingsAlias)
	}
	if store.saves != 0 {
		t.Errorf("Unset() Save() calls = %d, want 0", store.saves)
	}
}

type fakeMavenSettingsResolver struct {
	alias string
	entry mavensettings.Entry
	err   error
}

func (r *fakeMavenSettingsResolver) Resolve(alias string) (mavensettings.Entry, error) {
	r.alias = alias
	return r.entry, r.err
}

func TestMavenSettingsManagerAssociatesAliasWithCurrentProject(t *testing.T) {
	t.Parallel()

	store := &fakeSettingsProjectStore{
		found: true,
		config: Config{
			ProjectRoot: "/projects/demo",
			BuildTool:   buildtool.Info{Type: buildtool.Maven},
		},
	}
	resolver := &fakeMavenSettingsResolver{
		entry: mavensettings.Entry{Alias: "intranet", Path: "/maven/settings.xml"},
	}
	manager := NewMavenSettingsManager(store, resolver)

	config, entry, err := manager.Use("/projects/demo/module", "intranet")
	if err != nil {
		t.Fatalf("Use() error = %v", err)
	}
	if store.start != "/projects/demo/module" || resolver.alias != "intranet" {
		t.Errorf("Use() lookup arguments = %q/%q", store.start, resolver.alias)
	}
	if config.BuildTool.SettingsAlias != "intranet" || store.saved.BuildTool.SettingsAlias != "intranet" {
		t.Errorf("saved Maven settings alias = %q, want intranet", store.saved.BuildTool.SettingsAlias)
	}
	if entry.Path != "/maven/settings.xml" {
		t.Errorf("Use() entry = %#v, want resolved settings path", entry)
	}
}

func TestMavenSettingsManagerRequiresInitializedProject(t *testing.T) {
	t.Parallel()

	manager := NewMavenSettingsManager(&fakeSettingsProjectStore{}, &fakeMavenSettingsResolver{})
	_, _, err := manager.Use("/projects/demo", "intranet")
	if err == nil || !strings.Contains(err.Error(), "jup init") {
		t.Fatalf("Use() error = %v, want initialization guidance", err)
	}
}

func TestMavenSettingsManagerPersistsAliasInProjectStore(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projects := NewConfigStore(filepath.Join(t.TempDir(), "projects"))
	aliases := mavensettings.NewStore(filepath.Join(t.TempDir(), "maven", "settings.json"))
	settingsPath := filepath.Join(t.TempDir(), "settings.xml")
	if err := os.WriteFile(settingsPath, []byte(`<settings/>`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, _, err := aliases.Add("intranet", settingsPath); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if _, err := projects.Save(Config{
		SchemaVersion: currentSchemaVersion,
		ProjectRoot:   root,
		BuildTool:     buildtool.Info{Type: buildtool.Maven},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	manager := NewMavenSettingsManager(projects, aliases)
	if _, _, err := manager.Use(root, "intranet"); err != nil {
		t.Fatalf("Use() error = %v", err)
	}
	config, _, found, err := projects.Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !found || config.BuildTool.SettingsAlias != "intranet" {
		t.Errorf("persisted project found/alias = %t/%q, want true/intranet", found, config.BuildTool.SettingsAlias)
	}
}

func TestMavenSettingsManagerPersistsUnsetInProjectStore(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projects := NewConfigStore(filepath.Join(t.TempDir(), "projects"))
	if _, err := projects.Save(Config{
		SchemaVersion: currentSchemaVersion,
		ProjectRoot:   root,
		BuildTool: buildtool.Info{
			Type:          buildtool.Maven,
			SettingsAlias: "intranet",
		},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	manager := NewMavenSettingsManager(projects, &fakeMavenSettingsResolver{})
	if _, previousAlias, err := manager.Unset(root); err != nil {
		t.Fatalf("Unset() error = %v", err)
	} else if previousAlias != "intranet" {
		t.Errorf("Unset() previous alias = %q, want intranet", previousAlias)
	}

	config, path, found, err := projects.Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !found || config.BuildTool.SettingsAlias != "" {
		t.Errorf("persisted project found/alias = %t/%q, want true/empty", found, config.BuildTool.SettingsAlias)
	}
	// #nosec G304 -- path is returned by the temporary project store created by this test.
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.Contains(string(content), "settingsAlias") {
		t.Errorf("persisted project still contains settingsAlias: %s", content)
	}
}
