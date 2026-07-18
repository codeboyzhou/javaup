package project

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

// ConfigStore persists one JSON document per project.
type ConfigStore struct {
	baseDir string
}

// NewDefaultConfigStore uses the operating system's user configuration root.
func NewDefaultConfigStore() (*ConfigStore, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user configuration directory: %w", err)
	}
	return NewConfigStore(filepath.Join(configDir, "javaup", "projects")), nil
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

	path := filepath.Join(s.baseDir, configFileName(config.ProjectRoot))
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

func configFileName(projectRoot string) string {
	identity := filepath.Clean(projectRoot)
	if runtime.GOOS == "windows" {
		identity = strings.ToLower(identity)
	}
	digest := sha256.Sum256([]byte(identity))
	hash := hex.EncodeToString(digest[:])[:12]
	return sanitizeName(filepath.Base(projectRoot)) + "-" + hash + ".json"
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
