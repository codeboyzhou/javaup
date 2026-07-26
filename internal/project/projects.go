package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/codeboyzhou/javaup/internal/buildtool"
)

// RegistryStatus describes whether a saved project can still be found.
type RegistryStatus string

const (
	// RegistryAvailable means the saved project root exists.
	RegistryAvailable RegistryStatus = "available"
	// RegistryMissing means the saved project root no longer exists.
	RegistryMissing RegistryStatus = "missing"
	// RegistryInvalid means the saved configuration cannot be trusted.
	RegistryInvalid RegistryStatus = "invalid"
)

// RegistryEntry describes one configuration in the global project registry.
type RegistryEntry struct {
	Name        string
	ProjectRoot string
	ConfigPath  string
	BuildTool   buildtool.Type
	JavaVersion string
	Status      RegistryStatus
	Detail      string
	LastUsedAt  time.Time
	UseCount    uint64
}

// DisplayPath returns the project root, or the configuration path when a
// corrupt entry does not contain a usable root.
func (e RegistryEntry) DisplayPath() string {
	if e.ProjectRoot != "" {
		return e.ProjectRoot
	}
	return e.ConfigPath
}

// PruneResult summarizes stale project and usage records.
type PruneResult struct {
	Projects     []RegistryEntry
	UsageRecords int
	DryRun       bool
}

// Registry provides global project registry operations.
type Registry struct {
	configs *ConfigStore
	usage   *UsageStore
}

// NewDefaultRegistry uses javaup's configured project and usage stores.
func NewDefaultRegistry() (*Registry, error) {
	configs, err := NewDefaultConfigStore()
	if err != nil {
		return nil, err
	}
	usage, err := NewDefaultUsageStore()
	if err != nil {
		return nil, err
	}
	return NewRegistry(configs, usage), nil
}

// NewRegistry creates a global project registry from explicit stores.
func NewRegistry(configs *ConfigStore, usage *UsageStore) *Registry {
	return &Registry{configs: configs, usage: usage}
}

// List returns every saved project, including missing and invalid entries.
func (m *Registry) List() ([]RegistryEntry, []error, error) {
	entries, err := m.scan()
	if err != nil {
		return nil, nil, err
	}
	usage, err := m.usage.Load()
	var warnings []error
	if err != nil {
		warnings = append(warnings, err)
		usage = make(map[string]Usage)
	}
	for index := range entries {
		if entries[index].ProjectRoot == "" {
			continue
		}
		record := usage[projectPathIdentity(entries[index].ProjectRoot)]
		entries[index].LastUsedAt = record.LastUsedAt
		entries[index].UseCount = record.UseCount
	}
	sortProjectEntries(entries)
	return entries, warnings, nil
}

// Remove deletes one project's configuration and usage record by path.
func (m *Registry) Remove(ctx context.Context, root string) (RegistryEntry, bool, error) {
	canonicalRoot, err := canonicalProjectRoot(root)
	if err != nil {
		return RegistryEntry{}, false, err
	}
	config, configPath, found, loadErr := m.configs.Load(canonicalRoot)
	entry := RegistryEntry{
		Name:        filepath.Base(canonicalRoot),
		ProjectRoot: canonicalRoot,
		ConfigPath:  configPath,
	}
	if loadErr == nil && found {
		entry.BuildTool = config.BuildTool.Type
		entry.JavaVersion = config.Java.Version
	}

	path, removed, err := m.configs.Delete(canonicalRoot)
	entry.ConfigPath = path
	if err != nil {
		return entry, false, err
	}
	if !removed {
		if loadErr != nil {
			return entry, false, loadErr
		}
		return entry, false, nil
	}
	if err := m.usage.Delete(ctx, canonicalRoot); err != nil {
		return entry, true, err
	}
	return entry, true, nil
}

// Prune removes missing or invalid project configurations and orphaned usage
// records. A dry run returns the same summary without changing either store.
func (m *Registry) Prune(ctx context.Context, dryRun bool) (PruneResult, error) {
	entries, err := m.scan()
	if err != nil {
		return PruneResult{DryRun: dryRun}, err
	}
	result := PruneResult{DryRun: dryRun}
	keep := make(map[string]struct{})
	for _, entry := range entries {
		if entry.Status == RegistryAvailable {
			keep[projectPathIdentity(entry.ProjectRoot)] = struct{}{}
			continue
		}
		result.Projects = append(result.Projects, entry)
	}

	if !dryRun {
		for _, entry := range result.Projects {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			if err := m.removeConfigPath(entry.ConfigPath); err != nil {
				return result, err
			}
		}
	}
	usageRecords, err := m.usage.Prune(ctx, keep, dryRun)
	if err != nil {
		return result, err
	}
	result.UsageRecords = usageRecords
	sortProjectEntries(result.Projects)
	return result, nil
}

func (m *Registry) scan() ([]RegistryEntry, error) {
	records, err := scanProjectConfigurations(m.configs.baseDir)
	if err != nil {
		return nil, err
	}
	entries := make([]RegistryEntry, 0, len(records))
	for _, record := range records {
		entry := RegistryEntry{
			Name:        record.Name,
			ProjectRoot: record.ProjectRoot,
			ConfigPath:  record.ConfigPath,
			BuildTool:   record.Config.BuildTool.Type,
			JavaVersion: record.Config.Java.Version,
			Status:      record.Status,
		}
		if record.Err != nil {
			entry.Detail = record.Err.Error()
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (m *Registry) removeConfigPath(path string) error {
	baseDir, err := filepath.Abs(m.configs.baseDir)
	if err != nil {
		return fmt.Errorf("resolve project configuration directory: %w", err)
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve project configuration path: %w", err)
	}
	relative, err := filepath.Rel(baseDir, target)
	if err != nil || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refuse to remove project configuration outside %s: %s", baseDir, target)
	}
	if filepath.Ext(target) != ".json" {
		return fmt.Errorf("refuse to remove non-JSON project configuration: %s", target)
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove project configuration %s: %w", target, err)
	}
	return nil
}

func sortProjectEntries(entries []RegistryEntry) {
	sort.Slice(entries, func(left, right int) bool {
		if entries[left].Status != entries[right].Status {
			return entries[left].Status < entries[right].Status
		}
		if entries[left].Name != entries[right].Name {
			return entries[left].Name < entries[right].Name
		}
		return entries[left].DisplayPath() < entries[right].DisplayPath()
	})
}
