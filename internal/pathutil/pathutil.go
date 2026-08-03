// Package pathutil centralizes path normalization semantics so canonical
// forms, comparable identities, and location equality follow one rule set.
package pathutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Canonical returns path as an absolute, cleaned location without resolving
// symlinks. The caller's spelling is preserved so persisted and displayed
// paths stay stable, and the path may not exist yet.
func Canonical(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}

// Resolved returns the absolute, cleaned location with every symlink resolved.
// It fails when path does not exist.
func Resolved(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

// ResolvedOrClean returns the resolved form of path, falling back to the
// absolute cleaned form when path does not exist yet.
func ResolvedOrClean(path string) (string, error) {
	resolved, err := Resolved(path)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	return Canonical(path)
}

// Identity returns a stable key for comparing an absolute path with others.
// Existing symlinks are resolved so aliases such as /var and /private/var on
// macOS and Windows short (8.3) path components share one identity; missing
// paths keep their cleaned spelling. Case is folded on Windows.
func Identity(path string) string {
	identity := filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(identity); err == nil {
		identity = filepath.Clean(resolved)
	}
	if runtime.GOOS == "windows" {
		identity = strings.ToLower(identity)
	}
	return identity
}

// Same reports whether two paths denote the same location. Cleaned and
// Windows case-folded spellings match without touching the filesystem;
// existing paths are additionally compared after resolving symlinks.
func Same(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if left == right {
		return true
	}
	if runtime.GOOS == "windows" && strings.EqualFold(left, right) {
		return true
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
