// Package uninstall removes an installer-managed javaup installation.
package uninstall

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/codeboyzhou/javaup/internal/apphome"
	"github.com/codeboyzhou/javaup/internal/pathutil"
)

// Result describes an uninstall operation.
type Result struct {
	Home    string
	Purged  bool
	Pending bool
}

type plan struct {
	Home     string
	BinDir   string
	Target   string
	UserHome string
	Purge    bool
}

// Uninstaller removes the running executable and installer-managed settings.
type Uninstaller struct {
	Purge          bool
	Home           string
	ExecutablePath string
	UserHome       string
	apply          func(plan) (bool, error)
}

// New returns an uninstaller for the current javaup installation.
func New(purge bool) *Uninstaller {
	return &Uninstaller{Purge: purge, apply: applyUninstall}
}

// Run validates the installation and removes it.
func (u *Uninstaller) Run(ctx context.Context) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	home := u.Home
	var err error
	if home == "" {
		home, err = apphome.Resolve()
		if err != nil {
			return Result{}, err
		}
	}
	home, err = pathutil.Canonical(home)
	if err != nil {
		return Result{}, fmt.Errorf("resolve %s: %w", apphome.EnvironmentVariable, err)
	}

	userHome := u.UserHome
	if userHome == "" {
		userHome, err = os.UserHomeDir()
		if err != nil {
			return Result{}, fmt.Errorf("resolve user home directory: %w", err)
		}
	}
	userHome, err = pathutil.Canonical(userHome)
	if err != nil {
		return Result{}, fmt.Errorf("resolve user home directory: %w", err)
	}
	if err := validateHome(home, userHome, u.Purge); err != nil {
		return Result{}, err
	}

	target := u.ExecutablePath
	if target == "" {
		target, err = os.Executable()
		if err != nil {
			return Result{}, fmt.Errorf("locate current executable: %w", err)
		}
	}
	target, err = pathutil.Resolved(target)
	if err != nil {
		return Result{}, fmt.Errorf("resolve current executable: %w", err)
	}

	binaryName := "jup"
	if runtime.GOOS == "windows" {
		binaryName = "jup.exe"
	}
	expected, err := pathutil.ResolvedOrClean(filepath.Join(home, "bin", binaryName))
	if err != nil {
		return Result{}, fmt.Errorf("resolve managed executable: %w", err)
	}
	if !pathutil.Same(target, expected) {
		return Result{}, fmt.Errorf(
			"current executable is not managed by the javaup installer: %s (expected %s)",
			target,
			expected,
		)
	}

	pending, err := u.apply(plan{
		Home:     home,
		BinDir:   filepath.Dir(expected),
		Target:   target,
		UserHome: userHome,
		Purge:    u.Purge,
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Home: home, Purged: u.Purge, Pending: pending}, nil
}

func validateHome(home, userHome string, purge bool) error {
	volumeRoot := filepath.Clean(filepath.VolumeName(home) + string(filepath.Separator))
	if pathutil.Same(home, volumeRoot) {
		return fmt.Errorf("refusing to use a filesystem root as %s: %s", apphome.EnvironmentVariable, home)
	}
	if purge && pathContains(home, userHome) {
		return fmt.Errorf("refusing to purge %s because it contains the user home directory", home)
	}
	return nil
}

func pathContains(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}
