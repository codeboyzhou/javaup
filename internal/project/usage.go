package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/codeboyzhou/javaup/internal/apphome"
	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/gofrs/flock"
)

const (
	usageSchemaVersion = 1
	usageHalfLife      = 14 * 24 * time.Hour
)

// Usage describes how often and how recently a project has been built.
type Usage struct {
	ProjectRoot string    `json:"projectRoot"`
	LastUsedAt  time.Time `json:"lastUsedAt"`
	UseCount    uint64    `json:"useCount"`
	Score       float64   `json:"score"`
}

type usageRegistry struct {
	SchemaVersion int              `json:"schemaVersion"`
	Projects      map[string]Usage `json:"projects"`
}

// UsageStore persists project build usage independently from toolchain configuration.
type UsageStore struct {
	path string
}

// NewDefaultUsageStore uses the configured javaup application directory.
func NewDefaultUsageStore() (*UsageStore, error) {
	home, err := apphome.Resolve()
	if err != nil {
		return nil, err
	}
	return NewUsageStore(filepath.Join(home, "state", "project-usage.json")), nil
}

// NewUsageStore creates a usage store at path.
func NewUsageStore(path string) *UsageStore {
	return &UsageStore{path: path}
}

// Load returns all saved project usage records.
func (s *UsageStore) Load() (map[string]Usage, error) {
	registry, err := s.loadRegistry()
	if err != nil {
		return nil, err
	}
	return registry.Projects, nil
}

// Touch records one build attempt for projectRoot using a decaying frequency score.
func (s *UsageStore) Touch(ctx context.Context, projectRoot string, at time.Time) error {
	canonicalRoot, err := canonicalProjectRoot(projectRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create project usage directory: %w", err)
	}
	lock := flock.New(s.path + ".lock")
	locked, err := lock.TryLockContext(ctx, 25*time.Millisecond)
	if err != nil {
		return fmt.Errorf("lock project usage: %w", err)
	}
	if !locked {
		return fmt.Errorf("lock project usage: canceled")
	}
	defer func() { _ = lock.Unlock() }()

	registry, err := s.loadRegistry()
	if err != nil {
		return err
	}
	key := projectPathIdentity(canonicalRoot)
	usage := registry.Projects[key]
	usage.Score = decayedUsageScore(usage, at) + 1
	usage.ProjectRoot = canonicalRoot
	usage.LastUsedAt = at.Truncate(time.Second)
	usage.UseCount++
	registry.Projects[key] = usage
	return s.saveRegistry(registry)
}

// Delete removes the usage record for projectRoot.
func (s *UsageStore) Delete(ctx context.Context, projectRoot string) error {
	canonicalRoot, err := canonicalProjectRoot(projectRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create project usage directory: %w", err)
	}
	lock := flock.New(s.path + ".lock")
	locked, err := lock.TryLockContext(ctx, 25*time.Millisecond)
	if err != nil {
		return fmt.Errorf("lock project usage: %w", err)
	}
	if !locked {
		return fmt.Errorf("lock project usage: canceled")
	}
	defer func() { _ = lock.Unlock() }()

	registry, err := s.loadRegistry()
	if err != nil {
		return err
	}
	delete(registry.Projects, projectPathIdentity(canonicalRoot))
	if len(registry.Projects) == 0 {
		if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove project usage: %w", err)
		}
		return nil
	}
	return s.saveRegistry(registry)
}

func (s *UsageStore) loadRegistry() (usageRegistry, error) {
	registry := usageRegistry{SchemaVersion: usageSchemaVersion, Projects: make(map[string]Usage)}
	// #nosec G304 -- path is fixed when the store is created.
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return registry, nil
		}
		return usageRegistry{}, fmt.Errorf("read project usage: %w", err)
	}
	if err := json.Unmarshal(content, &registry); err != nil {
		return usageRegistry{}, fmt.Errorf("decode project usage %s: %w", s.path, err)
	}
	if registry.SchemaVersion != usageSchemaVersion {
		return usageRegistry{}, fmt.Errorf("project usage schema %d is unsupported", registry.SchemaVersion)
	}
	if registry.Projects == nil {
		registry.Projects = make(map[string]Usage)
	}
	return registry, nil
}

func (s *UsageStore) saveRegistry(registry usageRegistry) error {
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".project-usage-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary project usage: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()

	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(registry); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("encode project usage: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync project usage: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close project usage: %w", err)
	}
	if err := os.Rename(temporaryPath, s.path); err != nil {
		return fmt.Errorf("save project usage: %w", err)
	}
	return nil
}

func decayedUsageScore(usage Usage, at time.Time) float64 {
	if usage.Score <= 0 || usage.LastUsedAt.IsZero() {
		return 0
	}
	elapsed := at.Sub(usage.LastUsedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	return usage.Score * math.Pow(0.5, float64(elapsed)/float64(usageHalfLife))
}

// Candidate is one configured project available to the interactive selector.
type Candidate struct {
	Name        string
	ProjectRoot string
	Config      Config
	Usage       Usage
	Rank        float64
}

// Catalog combines saved project configurations with their usage ranking.
type Catalog struct {
	configs *ConfigStore
	usage   *UsageStore
	now     func() time.Time
}

// NewDefaultCatalog creates a catalog backed by javaup's application directory.
func NewDefaultCatalog() (*Catalog, error) {
	configs, err := NewDefaultConfigStore()
	if err != nil {
		return nil, err
	}
	usage, err := NewDefaultUsageStore()
	if err != nil {
		return nil, err
	}
	return NewCatalog(configs, usage), nil
}

// NewCatalog creates a project catalog from replaceable stores.
func NewCatalog(configs *ConfigStore, usage *UsageStore) *Catalog {
	return &Catalog{configs: configs, usage: usage, now: time.Now}
}

// List returns configured projects for tool ordered by recent-frequency rank.
func (c *Catalog) List(tool buildtool.Type) ([]Candidate, []error, error) {
	configs, warnings, err := c.configs.List()
	if err != nil {
		return nil, nil, err
	}
	usage, usageErr := c.usage.Load()
	if usageErr != nil {
		warnings = append(warnings, usageErr)
		usage = make(map[string]Usage)
	}
	now := c.now()
	candidates := make([]Candidate, 0, len(configs))
	for _, config := range configs {
		if config.BuildTool.Type != tool {
			continue
		}
		record := usage[projectPathIdentity(config.ProjectRoot)]
		candidates = append(candidates, Candidate{
			Name:        filepath.Base(config.ProjectRoot),
			ProjectRoot: config.ProjectRoot,
			Config:      config,
			Usage:       record,
			Rank:        decayedUsageScore(record, now),
		})
	}
	sort.SliceStable(candidates, func(left, right int) bool {
		if candidates[left].Rank != candidates[right].Rank {
			return candidates[left].Rank > candidates[right].Rank
		}
		if !candidates[left].Usage.LastUsedAt.Equal(candidates[right].Usage.LastUsedAt) {
			return candidates[left].Usage.LastUsedAt.After(candidates[right].Usage.LastUsedAt)
		}
		if candidates[left].Name != candidates[right].Name {
			return candidates[left].Name < candidates[right].Name
		}
		return candidates[left].ProjectRoot < candidates[right].ProjectRoot
	})
	return candidates, warnings, nil
}
