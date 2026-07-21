package uninstall

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunValidatesAndAppliesManagedInstall(t *testing.T) {
	t.Parallel()

	userHome := t.TempDir()
	home := filepath.Join(userHome, ".javaup")
	binaryName := "jup"
	if runtime.GOOS == "windows" {
		binaryName = "jup.exe"
	}
	target := filepath.Join(home, "bin", binaryName)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("binary"), 0o600); err != nil {
		t.Fatal(err)
	}

	var applied plan
	uninstaller := New(true)
	uninstaller.Home = home
	uninstaller.UserHome = userHome
	uninstaller.ExecutablePath = target
	uninstaller.apply = func(spec plan) (bool, error) {
		applied = spec
		return true, nil
	}
	result, err := uninstaller.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !result.Purged || !result.Pending || result.Home != home {
		t.Errorf("Run() = %+v", result)
	}
	if applied.Home != home || applied.Target != target || !applied.Purge {
		t.Errorf("applied plan = %+v", applied)
	}
}

func TestRunRejectsUnmanagedExecutable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	home := filepath.Join(root, ".javaup")
	target := filepath.Join(root, "other", "jup.exe")
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("binary"), 0o600); err != nil {
		t.Fatal(err)
	}
	uninstaller := New(false)
	uninstaller.Home = home
	uninstaller.UserHome = root
	uninstaller.ExecutablePath = target
	uninstaller.apply = func(plan) (bool, error) {
		t.Fatal("apply should not be called for an unmanaged executable")
		return false, nil
	}
	_, err := uninstaller.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "not managed") {
		t.Fatalf("Run() error = %v, want unmanaged executable error", err)
	}
}

func TestRunRejectsPurgingUserHome(t *testing.T) {
	t.Parallel()

	userHome := t.TempDir()
	uninstaller := New(true)
	uninstaller.Home = userHome
	uninstaller.UserHome = userHome
	_, err := uninstaller.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "contains the user home") {
		t.Fatalf("Run() error = %v, want user home protection", err)
	}
}

func TestCleanInstallerProfileRemovesOnlyManagedBlock(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	profile := filepath.Join(directory, ".profile")
	managed := "export PATH='/home/test/.javaup/bin':$PATH"
	contents := "export EDITOR=vim\n\n" + installerProfileHeader + "\n" + managed +
		"\n# user content\nexport CUSTOM=1\n"
	if err := os.WriteFile(profile, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	lines := map[string]bool{managed: true}
	if err := cleanInstallerProfile(profile, lines, lines); err != nil {
		t.Fatalf("cleanInstallerProfile() error = %v", err)
	}
	got, err := os.ReadFile(profile) // #nosec G304 -- profile is created inside t.TempDir().
	if err != nil {
		t.Fatal(err)
	}
	want := "export EDITOR=vim\n# user content\nexport CUSTOM=1\n"
	if string(got) != want {
		t.Errorf("cleaned profile = %q, want %q", got, want)
	}
}

func TestCleanInstallerProfilePreservesUnrecognizedHeader(t *testing.T) {
	t.Parallel()

	profile := filepath.Join(t.TempDir(), ".profile")
	contents := installerProfileHeader + "\nexport SOMETHING_ELSE=1\n"
	if err := os.WriteFile(profile, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	lines := map[string]bool{"managed": true}
	if err := cleanInstallerProfile(profile, lines, lines); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(profile) // #nosec G304 -- profile is created inside t.TempDir().
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != contents {
		t.Errorf("profile changed unexpectedly: %q", got)
	}
}

func TestCleanInstallerProfileCanPreserveJavaupHome(t *testing.T) {
	t.Parallel()

	profile := filepath.Join(t.TempDir(), ".profile")
	homeLine := "export JAVAUP_HOME='/custom/javaup'"
	pathLine := "export PATH='/custom/javaup/bin':$PATH"
	contents := installerProfileHeader + "\n" + homeLine + "\n" + pathLine + "\n"
	if err := os.WriteFile(profile, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	recognized := map[string]bool{homeLine: true, pathLine: true}
	remove := map[string]bool{pathLine: true}
	if err := cleanInstallerProfile(profile, recognized, remove); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(profile) // #nosec G304 -- profile is created inside t.TempDir().
	if err != nil {
		t.Fatal(err)
	}
	want := installerProfileHeader + "\n" + homeLine + "\n"
	if string(got) != want {
		t.Errorf("cleaned profile = %q, want %q", got, want)
	}
}
