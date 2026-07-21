//go:build !windows

package uninstall

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyUninstallPreservesConfiguration(t *testing.T) {
	t.Parallel()

	userHome := t.TempDir()
	home := filepath.Join(userHome, ".javaup")
	binDir := filepath.Join(home, "bin")
	target := filepath.Join(binDir, "jup")
	config := filepath.Join(home, "config", "projects", "saved.json")
	if err := os.MkdirAll(filepath.Dir(config), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(userHome, ".profile")
	homeLine := "export JAVAUP_HOME=" + shellQuote(home)
	pathLine := "export PATH=" + shellQuote(binDir) + ":$PATH"
	if err := os.WriteFile(
		profile,
		[]byte(installerProfileHeader+"\n"+homeLine+"\n"+pathLine+"\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	pending, err := applyUninstall(plan{
		Home: home, BinDir: binDir, Target: target, UserHome: userHome,
	})
	if err != nil {
		t.Fatalf("applyUninstall() error = %v", err)
	}
	if pending {
		t.Error("applyUninstall() should not be pending on this platform")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("target still exists or returned unexpected error: %v", err)
	}
	if _, err := os.Stat(config); err != nil {
		t.Errorf("configuration was not preserved: %v", err)
	}
	profileContents, err := os.ReadFile(profile) // #nosec G304 -- profile is beneath t.TempDir().
	if err != nil {
		t.Fatal(err)
	}
	wantProfile := installerProfileHeader + "\n" + homeLine + "\n"
	if string(profileContents) != wantProfile {
		t.Errorf("profile = %q, want %q", profileContents, wantProfile)
	}
}

func TestApplyUninstallPurgesData(t *testing.T) {
	t.Parallel()

	userHome := t.TempDir()
	home := filepath.Join(userHome, ".javaup")
	binDir := filepath.Join(home, "bin")
	target := filepath.Join(binDir, "jup")
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := applyUninstall(plan{
		Home: home, BinDir: binDir, Target: target, UserHome: userHome, Purge: true,
	}); err != nil {
		t.Fatalf("applyUninstall() error = %v", err)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Errorf("home still exists or returned unexpected error: %v", err)
	}
}
