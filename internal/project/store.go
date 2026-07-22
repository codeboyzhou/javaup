package project

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"github.com/codeboyzhou/javaup/internal/apphome"
)

// ConfigStore persists one JSON document per project.
type ConfigStore struct {
	baseDir string
}

type configFinder interface {
	Find(start string) (config Config, path string, found bool, err error)
}

// NewDefaultConfigStore uses the configured javaup application directory.
func NewDefaultConfigStore() (*ConfigStore, error) {
	home, err := apphome.Resolve()
	if err != nil {
		return nil, err
	}
	return NewConfigStore(filepath.Join(home, "config", "projects")), nil
}

// NewConfigStore creates a store rooted at baseDir.
func NewConfigStore(baseDir string) *ConfigStore {
	return &ConfigStore{baseDir: baseDir}
}

// Save atomically writes a human-readable JSON project configuration.
func (s *ConfigStore) Save(config Config) (string, error) {
	if err := os.MkdirAll(s.baseDir, 0o700); err != nil {
		return "", fmt.Errorf("create project configuration directory: %w", err)
	}

	canonicalRoot, err := canonicalProjectRoot(config.ProjectRoot)
	if err != nil {
		return "", err
	}
	path := filepath.Join(s.baseDir, configFileName(canonicalRoot))
	temporary, err := os.CreateTemp(s.baseDir, ".project-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temporary project configuration: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()

	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(config); err != nil {
		_ = temporary.Close()
		return "", fmt.Errorf("encode project configuration: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return "", fmt.Errorf("sync project configuration: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close project configuration: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return "", fmt.Errorf("save project configuration: %w", err)
	}

	return path, nil
}

// Load reads the configuration associated with projectRoot.
func (s *ConfigStore) Load(projectRoot string) (config Config, path string, found bool, err error) {
	canonicalRoot, err := canonicalProjectRoot(projectRoot)
	if err != nil {
		return Config{}, "", false, err
	}
	path = filepath.Join(s.baseDir, configFileName(canonicalRoot))
	config, err = readProjectConfig(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, path, false, nil
		}
		return Config{}, path, true, err
	}

	configuredRoot, err := canonicalProjectRoot(config.ProjectRoot)
	if err != nil {
		return Config{}, path, true, fmt.Errorf("resolve configured project root: %w", err)
	}
	if !samePath(canonicalRoot, configuredRoot) {
		return Config{}, path, true, fmt.Errorf("project configuration root is %s, want %s", configuredRoot, canonicalRoot)
	}
	config.ProjectRoot = configuredRoot
	return config, path, true, nil
}

// List reads every valid project configuration. Invalid entries are returned as
// warnings so one stale project does not hide the rest of the catalog.
func (s *ConfigStore) List() (configs []Config, warnings []error, err error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("list project configurations: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(s.baseDir, entry.Name())
		config, readErr := readProjectConfig(path)
		if readErr != nil {
			warnings = append(warnings, readErr)
			continue
		}
		canonicalRoot, rootErr := canonicalProjectRoot(config.ProjectRoot)
		if rootErr != nil {
			warnings = append(warnings, fmt.Errorf("resolve configured project root in %s: %w", path, rootErr))
			continue
		}
		if entry.Name() != configFileName(canonicalRoot) {
			warnings = append(warnings, fmt.Errorf("project configuration filename does not match its root: %s", path))
			continue
		}
		info, statErr := os.Stat(canonicalRoot)
		if statErr != nil {
			warnings = append(warnings, fmt.Errorf("inspect configured project %s: %w", canonicalRoot, statErr))
			continue
		}
		if !info.IsDir() {
			warnings = append(warnings, fmt.Errorf("configured project root is not a directory: %s", canonicalRoot))
			continue
		}
		config.ProjectRoot = canonicalRoot
		configs = append(configs, config)
	}
	return configs, warnings, nil
}

func readProjectConfig(path string) (Config, error) {
	// #nosec G304 -- callers restrict paths to the configured store directory.
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read project configuration %s: %w", path, err)
	}
	var config Config
	if err := json.Unmarshal(content, &config); err != nil {
		return Config{}, fmt.Errorf("decode project configuration %s: %w", path, err)
	}
	if config.SchemaVersion != currentSchemaVersion {
		return Config{}, fmt.Errorf(
			"project configuration schema %d in %s is unsupported; run jup init again",
			config.SchemaVersion,
			path,
		)
	}
	return config, nil
}

// Find searches start and its parents for an initialized project.
func (s *ConfigStore) Find(start string) (config Config, path string, found bool, err error) {
	directory, err := canonicalProjectRoot(start)
	if err != nil {
		return Config{}, "", false, err
	}
	for {
		config, path, found, err = s.Load(directory)
		if err != nil || found {
			return config, path, found, err
		}
		parent := filepath.Dir(directory)
		if samePath(parent, directory) {
			return Config{}, path, false, nil
		}
		directory = parent
	}
}

// Delete removes the configuration associated with projectRoot.
func (s *ConfigStore) Delete(projectRoot string) (path string, removed bool, err error) {
	canonicalRoot, err := canonicalProjectRoot(projectRoot)
	if err != nil {
		return "", false, err
	}
	path = filepath.Join(s.baseDir, configFileName(canonicalRoot))
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return path, false, nil
		}
		return path, false, fmt.Errorf("remove project configuration: %w", err)
	}
	return path, true, nil
}

func configFileName(projectRoot string) string {
	digest := sha256.Sum256([]byte(projectPathIdentity(projectRoot)))
	hash := hex.EncodeToString(digest[:])[:12]
	return sanitizeName(filepath.Base(projectRoot)) + "-" + hash + ".json"
}

func projectPathIdentity(projectRoot string) string {
	identity := filepath.Clean(projectRoot)
	if runtime.GOOS == "windows" {
		identity = strings.ToLower(identity)
	}
	return identity
}

func sanitizeName(value string) string {
	value = strings.Map(func(character rune) rune {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || character == '-' || character == '_' {
			return character
		}
		return '-'
	}, value)
	value = strings.Trim(value, "-")
	if value == "" {
		return "project"
	}
	return value
}

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if left == right {
		return true
	}
	if runtime.GOOS == "windows" {
		if strings.EqualFold(left, right) {
			return true
		}
	}

	resolvedLeft, leftErr := filepath.EvalSymlinks(left)
	resolvedRight, rightErr := filepath.EvalSymlinks(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	resolvedLeft = filepath.Clean(resolvedLeft)
	resolvedRight = filepath.Clean(resolvedRight)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(resolvedLeft, resolvedRight)
	}
	return resolvedLeft == resolvedRight
}
