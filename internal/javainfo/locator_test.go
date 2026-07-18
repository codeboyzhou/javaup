package javainfo

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNormalizeVersion(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"1.8.0_442": "8",
		"11.0.26":   "11",
		"17":        "17",
		"21-ea":     "21",
	}
	for input, want := range tests {
		input, want := input, want
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeVersion(input)
			if err != nil {
				t.Fatalf("normalizeVersion() error = %v", err)
			}
			if got != want {
				t.Errorf("normalizeVersion() = %q, want %q", got, want)
			}
		})
	}
}

func TestInspectCandidateUsesReleaseMetadata(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "bin"), 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "release"), []byte("JAVA_VERSION=\"17.0.12\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(release) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "bin", javacExecutable()), nil, 0o600); err != nil {
		t.Fatalf("WriteFile(javac) error = %v", err)
	}

	installation, ok := inspectCandidate(context.Background(), home)
	if !ok {
		t.Fatal("inspectCandidate() ok = false, want true")
	}
	if installation.Version != "17" {
		t.Errorf("Version = %q, want %q", installation.Version, "17")
	}
	wantHome, _ := filepath.EvalSymlinks(home)
	if installation.Home != filepath.Clean(wantHome) {
		t.Errorf("Home = %q, want %q on %s", installation.Home, wantHome, runtime.GOOS)
	}
}
