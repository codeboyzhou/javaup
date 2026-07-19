package project

import (
	"fmt"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/mavensettings"
)

type mavenSettingsResolver interface {
	Resolve(alias string) (mavensettings.Entry, error)
}

type settingsProjectStore interface {
	configFinder
	configStore
}

// MavenSettingsManager associates initialized Maven projects with settings aliases.
type MavenSettingsManager struct {
	projects settingsProjectStore
	settings mavenSettingsResolver
}

// NewDefaultMavenSettingsManager uses the platform-specific project and settings stores.
func NewDefaultMavenSettingsManager() (*MavenSettingsManager, error) {
	projects, err := NewDefaultConfigStore()
	if err != nil {
		return nil, err
	}
	settings, err := mavensettings.NewDefaultStore()
	if err != nil {
		return nil, err
	}
	return NewMavenSettingsManager(projects, settings), nil
}

// NewMavenSettingsManager creates a manager from replaceable stores.
func NewMavenSettingsManager(projects settingsProjectStore, settings mavenSettingsResolver) *MavenSettingsManager {
	return &MavenSettingsManager{projects: projects, settings: settings}
}

// Use saves alias as the Maven settings selection for the project containing root.
func (m *MavenSettingsManager) Use(root, alias string) (Config, mavensettings.Entry, error) {
	config, _, found, err := m.projects.Find(root)
	if err != nil {
		return Config{}, mavensettings.Entry{}, err
	}
	if !found {
		return Config{}, mavensettings.Entry{}, fmt.Errorf(
			"no initialized javaup project found from %s; run jup init",
			root,
		)
	}
	if config.BuildTool.Type != buildtool.Maven {
		return Config{}, mavensettings.Entry{}, fmt.Errorf(
			"project build tool is %s, not Maven",
			config.BuildTool.Type.DisplayName(),
		)
	}

	entry, err := m.settings.Resolve(alias)
	if err != nil {
		return Config{}, mavensettings.Entry{}, err
	}
	config.BuildTool.SettingsAlias = entry.Alias
	if _, err := m.projects.Save(config); err != nil {
		return Config{}, mavensettings.Entry{}, err
	}
	return config, entry, nil
}
