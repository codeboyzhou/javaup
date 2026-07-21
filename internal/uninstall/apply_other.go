//go:build !windows

package uninstall

import (
	"fmt"
	"os"
)

func applyUninstall(spec plan) (bool, error) {
	if err := cleanInstallerProfiles(spec); err != nil {
		return false, err
	}
	if err := os.Remove(spec.Target); err != nil {
		return false, fmt.Errorf("remove executable: %w", err)
	}
	if spec.Purge {
		if err := os.RemoveAll(spec.Home); err != nil {
			return false, fmt.Errorf("purge %s: %w", spec.Home, err)
		}
	} else {
		entries, err := os.ReadDir(spec.BinDir)
		if err != nil && !os.IsNotExist(err) {
			return false, fmt.Errorf("inspect bin directory: %w", err)
		}
		if err == nil && len(entries) == 0 {
			if err := os.Remove(spec.BinDir); err != nil {
				return false, fmt.Errorf("remove empty bin directory: %w", err)
			}
		}
	}
	return false, nil
}
