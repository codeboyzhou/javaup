package apphome

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveUsesConfiguredHome(t *testing.T) {
	configured := filepath.Join(t.TempDir(), "custom-home")
	t.Setenv(EnvironmentVariable, configured)

	got, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got != filepath.Clean(configured) {
		t.Errorf("Resolve() = %q, want %q", got, filepath.Clean(configured))
	}
}

func TestResolveDefaultsToHiddenUserDirectory(t *testing.T) {
	t.Setenv(EnvironmentVariable, "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}

	got, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	want := filepath.Join(home, ".javaup")
	if got != want {
		t.Errorf("Resolve() = %q, want %q", got, want)
	}
}

func TestResolveRejectsRelativeConfiguredHome(t *testing.T) {
	t.Setenv(EnvironmentVariable, filepath.Join("relative", "home"))

	_, err := Resolve()
	if err == nil || !strings.Contains(err.Error(), "absolute path") {
		t.Fatalf("Resolve() error = %v, want absolute path error", err)
	}
}
