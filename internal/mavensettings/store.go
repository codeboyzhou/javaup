// Package mavensettings manages named Maven settings files.
package mavensettings

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/codeboyzhou/javaup/internal/apphome"
)

const currentSchemaVersion = 1

// Entry describes a saved Maven settings alias.
type Entry struct {
	Alias string
	Path  string
}

type registry struct {
	SchemaVersion int               `json:"schemaVersion"`
	Aliases       map[string]string `json:"aliases"`
}

// Store persists Maven settings aliases in a single JSON document.
type Store struct {
	path string
}

// NewDefaultStore uses the configured javaup application directory.
func NewDefaultStore() (*Store, error) {
	home, err := apphome.Resolve()
	if err != nil {
		return nil, err
	}
	return NewStore(filepath.Join(home, "config", "maven", "settings.json")), nil
}

// NewStore creates a Maven settings store at path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Add saves alias as a reference to settingsPath. Existing aliases are updated.
func (s *Store) Add(alias, settingsPath string) (Entry, string, error) {
	alias, err := validateAlias(alias)
	if err != nil {
		return Entry{}, s.path, err
	}
	settingsPath, err = validateSettingsFile(settingsPath)
	if err != nil {
		return Entry{}, s.path, err
	}

	config, err := s.load()
	if err != nil {
		return Entry{}, s.path, err
	}
	config.Aliases[alias] = settingsPath
	if err := s.save(config); err != nil {
		return Entry{}, s.path, err
	}

	return Entry{Alias: alias, Path: settingsPath}, s.path, nil
}

// Resolve returns the valid Maven settings file currently saved for alias.
func (s *Store) Resolve(alias string) (Entry, error) {
	alias, err := validateAlias(alias)
	if err != nil {
		return Entry{}, err
	}
	config, err := s.load()
	if err != nil {
		return Entry{}, err
	}
	settingsPath, found := config.Aliases[alias]
	if !found {
		return Entry{}, fmt.Errorf("maven settings alias %q is not configured; run jup settings add", alias)
	}
	settingsPath, err = validateSettingsFile(settingsPath)
	if err != nil {
		return Entry{}, fmt.Errorf("resolve Maven settings alias %q: %w", alias, err)
	}
	return Entry{Alias: alias, Path: settingsPath}, nil
}

// List returns all saved Maven settings aliases ordered by alias.
func (s *Store) List() ([]Entry, error) {
	config, err := s.load()
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(config.Aliases))
	for alias, settingsPath := range config.Aliases {
		entries = append(entries, Entry{Alias: alias, Path: settingsPath})
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].Alias < entries[right].Alias
	})
	return entries, nil
}

// Remove deletes the saved Maven settings alias and returns its former mapping.
func (s *Store) Remove(alias string) (Entry, error) {
	alias, err := validateAlias(alias)
	if err != nil {
		return Entry{}, err
	}
	config, err := s.load()
	if err != nil {
		return Entry{}, err
	}
	settingsPath, found := config.Aliases[alias]
	if !found {
		return Entry{}, fmt.Errorf("maven settings alias %q is not configured", alias)
	}

	delete(config.Aliases, alias)
	if err := s.save(config); err != nil {
		return Entry{}, err
	}
	return Entry{Alias: alias, Path: settingsPath}, nil
}

func (s *Store) load() (registry, error) {
	content, err := os.ReadFile(s.path) // #nosec G304 -- path belongs to the configured Maven settings store.
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return registry{SchemaVersion: currentSchemaVersion, Aliases: make(map[string]string)}, nil
		}
		return registry{}, fmt.Errorf("read Maven settings aliases: %w", err)
	}

	var config registry
	if err := json.Unmarshal(content, &config); err != nil {
		return registry{}, fmt.Errorf("decode Maven settings aliases %s: %w", s.path, err)
	}
	if config.SchemaVersion != currentSchemaVersion {
		return registry{}, fmt.Errorf(
			"maven settings aliases schema %d is unsupported",
			config.SchemaVersion,
		)
	}
	if config.Aliases == nil {
		config.Aliases = make(map[string]string)
	}
	return config, nil
}

func (s *Store) save(config registry) error {
	directory := filepath.Dir(s.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create Maven settings configuration directory: %w", err)
	}

	temporary, err := os.CreateTemp(directory, ".settings-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary Maven settings configuration: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()

	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(config); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("encode Maven settings aliases: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync Maven settings aliases: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close Maven settings aliases: %w", err)
	}
	if err := os.Rename(temporaryPath, s.path); err != nil {
		return fmt.Errorf("save Maven settings aliases: %w", err)
	}
	return nil
}

func validateAlias(alias string) (string, error) {
	if alias == "" {
		return "", fmt.Errorf("maven settings alias must not be empty")
	}
	if strings.TrimSpace(alias) != alias {
		return "", fmt.Errorf("maven settings alias must not start or end with whitespace")
	}
	for index, character := range alias {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || (index > 0 && strings.ContainsRune("-_.", character)) {
			continue
		}
		return "", fmt.Errorf("maven settings alias %q must contain only letters, digits, dots, dashes, or underscores", alias)
	}
	return alias, nil
}

func validateSettingsFile(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("maven settings path must not be empty")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve Maven settings path: %w", err)
	}
	if resolvedPath, resolveErr := filepath.EvalSymlinks(absolutePath); resolveErr == nil {
		absolutePath = resolvedPath
	}
	absolutePath = filepath.Clean(absolutePath)

	info, err := os.Stat(absolutePath)
	if err != nil {
		return "", fmt.Errorf("inspect Maven settings file %s: %w", absolutePath, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("maven settings path is not a regular file: %s", absolutePath)
	}

	content, err := os.ReadFile(absolutePath) // #nosec G304 -- path is explicitly supplied by the user.
	if err != nil {
		return "", fmt.Errorf("read Maven settings file %s: %w", absolutePath, err)
	}
	var document struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal(content, &document); err != nil {
		return "", fmt.Errorf("parse Maven settings file %s: %w", absolutePath, err)
	}
	if document.XMLName.Local != "settings" {
		return "", fmt.Errorf("maven settings file %s has root element %q, want %q", absolutePath, document.XMLName.Local, "settings")
	}
	return absolutePath, nil
}
