package pathutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCanonicalMakesPathAbsoluteAndClean(t *testing.T) {
	t.Parallel()

	path := filepath.Join("a", "b", "..", "b", string(filepath.Separator))
	want, err := filepath.Abs(filepath.Join("a", "b"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Canonical(path)
	if err != nil {
		t.Fatalf("Canonical() error = %v", err)
	}
	if got != want {
		t.Errorf("Canonical(%q) = %q, want %q", path, got, want)
	}
}

func TestResolvedFailsForMissingPath(t *testing.T) {
	t.Parallel()

	if _, err := Resolved(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("Resolved() error = nil, want not-exist error")
	}
}

func TestResolvedOrCleanFallsBackForMissingPath(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing")
	got, err := ResolvedOrClean(path)
	if err != nil {
		t.Fatalf("ResolvedOrClean() error = %v", err)
	}
	if got != path {
		t.Errorf("ResolvedOrClean(%q) = %q, want %q", path, got, path)
	}
}

func TestSameResolvesSymlinks(t *testing.T) {
	t.Parallel()

	temporary := t.TempDir()
	target := filepath.Join(temporary, "target")
	if err := os.Mkdir(target, 0o750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	link := filepath.Join(temporary, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("Symlink() error = %v", err)
	}

	if !Same(link, target) {
		t.Errorf("Same(%q, %q) = false, want true", link, target)
	}
}

func TestSameMatchesCleanedSpellingWithoutFilesystemAccess(t *testing.T) {
	t.Parallel()

	path := filepath.Join("a", "b", string(filepath.Separator))
	if !Same(path, filepath.Clean(path)) {
		t.Errorf("Same(%q, %q) = false, want true", path, filepath.Clean(path))
	}
}

func TestSameFoldsCaseOnWindows(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific case folding")
	}
	path := filepath.Join(t.TempDir(), "MixedCase")
	if err := os.Mkdir(path, 0o750); err != nil {
		t.Fatal(err)
	}
	if !Same(path, strings.ToLower(path)) {
		t.Errorf("Same(%q, %q) = false, want true", path, strings.ToLower(path))
	}
}

func TestIdentityResolvesSymlinks(t *testing.T) {
	t.Parallel()

	temporary := t.TempDir()
	target := filepath.Join(temporary, "target")
	if err := os.Mkdir(target, 0o750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	link := filepath.Join(temporary, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("Symlink() error = %v", err)
	}

	if Identity(link) != Identity(target) {
		t.Errorf("Identity(%q) = %q, want %q", link, Identity(link), Identity(target))
	}
}

func TestIdentityFoldsCaseOnWindows(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific case folding")
	}
	path := filepath.Join(t.TempDir(), "MixedCase")
	if Identity(path) != strings.ToLower(path) {
		t.Errorf("Identity(%q) = %q, want %q", path, Identity(path), strings.ToLower(path))
	}
}
