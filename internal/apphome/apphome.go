// Package apphome resolves javaup's per-user application directory.
package apphome

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnvironmentVariable overrides the default javaup application directory.
const EnvironmentVariable = "JAVAUP_HOME"

// Resolve returns the root shared by javaup binaries and configuration.
func Resolve() (string, error) {
	if configured := os.Getenv(EnvironmentVariable); configured != "" {
		if !filepath.IsAbs(configured) {
			return "", fmt.Errorf("%s must be an absolute path", EnvironmentVariable)
		}
		return filepath.Clean(configured), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	return filepath.Join(home, ".javaup"), nil
}
