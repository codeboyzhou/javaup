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
}

func (s *fakeSettingsProjectStore) Find(start string) (Config, string, bool, error) {
	s.start = start
	return s.config, "project.json", s.found, nil
}

func (s *fakeSettingsProjectStore) Save(config Config) (string, error) {
	s.saved = config
	return "project.json", nil
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
