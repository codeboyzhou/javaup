//go:build !windows

package uninstall

import (
	"path/filepath"
	"strings"
)

func cleanInstallerProfiles(spec plan) error {
	profiles := []string{
		filepath.Join(spec.UserHome, ".zshrc"),
		filepath.Join(spec.UserHome, ".bash_profile"),
		filepath.Join(spec.UserHome, ".bashrc"),
		filepath.Join(spec.UserHome, ".profile"),
		filepath.Join(spec.UserHome, ".config", "fish", "config.fish"),
	}
	recognizedLines := map[string]bool{
		"export PATH=" + shellQuote(spec.BinDir) + ":$PATH": true,
		"export JAVAUP_HOME=" + shellQuote(spec.Home):       true,
		"fish_add_path " + shellQuote(spec.BinDir):          true,
		"set -gx JAVAUP_HOME " + shellQuote(spec.Home):      true,
	}
	removeLines := map[string]bool{
		"export PATH=" + shellQuote(spec.BinDir) + ":$PATH": true,
		"fish_add_path " + shellQuote(spec.BinDir):          true,
	}
	if spec.Purge {
		removeLines["export JAVAUP_HOME="+shellQuote(spec.Home)] = true
		removeLines["set -gx JAVAUP_HOME "+shellQuote(spec.Home)] = true
	}

	for _, profile := range profiles {
		if err := cleanInstallerProfile(profile, recognizedLines, removeLines); err != nil {
			return err
		}
	}
	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
