package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/javainfo"
	"github.com/codeboyzhou/javaup/internal/mavensettings"
)

type doctorConfigFinder struct {
	config Config
	found  bool
	err    error
	start  string
}

func (f *doctorConfigFinder) Find(start string) (Config, string, bool, error) {
	f.start = start
	return f.config, "", f.found, f.err
}

type doctorSettingsResolver struct {
	entry mavensettings.Entry
	err   error
	alias string
}

func (r *doctorSettingsResolver) Resolve(alias string) (mavensettings.Entry, error) {
	r.alias = alias
	return r.entry, r.err
}

func TestDoctorReportsHealthyMavenProject(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeDoctorFile(t, filepath.Join(root, "pom.xml"), `
<project>
  <modelVersion>4.0.0</modelVersion>
  <properties><maven.compiler.release>17</maven.compiler.release></properties>
</project>`, 0o600)
	executable := filepath.Join(root, "mvn")
	writeDoctorFile(t, executable, "#!/bin/sh\n", 0o755)

	settings := &doctorSettingsResolver{
		entry: mavensettings.Entry{Alias: "intranet", Path: filepath.Join(root, "settings.xml")},
	}
	store := &doctorConfigFinder{
		found: true,
		config: Config{
			ProjectRoot: root,
			BuildTool: buildtool.Info{
				Type:          buildtool.Maven,
				Version:       "3.9.11",
				Executable:    executable,
				SettingsAlias: "intranet",
			},
			Java: javainfo.Installation{Version: "17.0.12", Home: filepath.Join(root, "jdk-17")},
		},
	}
	doctor := NewDoctor(store, settings)
	doctor.inspectJDK = func(context.Context, string) (javainfo.Installation, error) {
		return javainfo.Installation{Version: "17.0.12", Home: filepath.Join(root, "jdk-17")}, nil
	}

	report, err := doctor.Diagnose(context.Background(), filepath.Join(root, "module"))
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	if got := report.Failed(); got != 0 {
		t.Fatalf("Failed() = %d, want 0: %#v", got, report.Checks)
	}
	if got := report.Warnings(); got != 0 {
		t.Errorf("Warnings() = %d, want 0", got)
	}
	if len(report.Checks) != 5 {
		t.Fatalf("checks = %d, want 5: %#v", len(report.Checks), report.Checks)
	}
	if settings.alias != "intranet" {
		t.Errorf("resolved settings alias = %q, want intranet", settings.alias)
	}
	assertDoctorCheck(t, report, "Java requirement", CheckPassed, "requires Java 17")
	assertDoctorCheck(t, report, "Maven", CheckPassed, "Maven 3.9.11")
	assertDoctorCheck(t, report, "JDK", CheckPassed, "Java 17.0.12")
	assertDoctorCheck(t, report, "Maven settings", CheckPassed, "intranet")
}

func TestDoctorReportsActionableBrokenResources(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeDoctorFile(t, filepath.Join(root, "pom.xml"), `
<project>
  <modelVersion>4.0.0</modelVersion>
  <properties><java.version>17</java.version></properties>
</project>`, 0o600)
	store := &doctorConfigFinder{
		found: true,
		config: Config{
			ProjectRoot: root,
			BuildTool: buildtool.Info{
				Type:          buildtool.Maven,
				Version:       "3.9.11",
				Executable:    filepath.Join(root, "missing-mvn"),
				SettingsAlias: "intranet",
			},
			Java: javainfo.Installation{Version: "11.0.25", Home: filepath.Join(root, "jdk-11")},
		},
	}
	settings := &doctorSettingsResolver{err: errors.New("settings file is missing")}
	doctor := NewDoctor(store, settings)
	doctor.inspectJDK = func(context.Context, string) (javainfo.Installation, error) {
		return javainfo.Installation{Version: "11.0.25", Home: filepath.Join(root, "jdk-11")}, nil
	}

	report, err := doctor.Diagnose(context.Background(), root)
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	if got := report.Failed(); got != 3 {
		t.Fatalf("Failed() = %d, want 3: %#v", got, report.Checks)
	}
	assertDoctorCheck(t, report, "Maven", CheckFailed, "unavailable")
	assertDoctorCheck(t, report, "JDK", CheckFailed, "requires Java 17")
	assertDoctorCheck(t, report, "Maven settings", CheckFailed, "settings file is missing")
	for _, name := range []string{"Maven", "JDK", "Maven settings"} {
		check := findDoctorCheck(t, report, name)
		if check.Suggestion == "" {
			t.Errorf("%s suggestion is empty", name)
		}
	}
}

func TestDoctorWarnsWhenPOMHasNoJavaRequirement(t *testing.T) {
	t.Parallel()

	report := diagnoseDoctorProject(t, `
<project><modelVersion>4.0.0</modelVersion></project>`, "21.0.8", "21.0.8")
	if report.Failed() != 0 || report.Warnings() != 1 {
		t.Fatalf("failed/warnings = %d/%d, want 0/1", report.Failed(), report.Warnings())
	}
	assertDoctorCheck(t, report, "Java requirement", CheckWarning, "does not declare")
}

func TestDoctorReportsUninitializedProject(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := &doctorConfigFinder{}
	report, err := NewDoctor(store, nil).Diagnose(context.Background(), root)
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	if store.start != root {
		t.Errorf("Find() start = %q, want %q", store.start, root)
	}
	if report.Failed() != 1 {
		t.Fatalf("Failed() = %d, want 1", report.Failed())
	}
	check := findDoctorCheck(t, report, "Configuration")
	if !strings.Contains(check.Suggestion, "jup init") {
		t.Errorf("suggestion = %q, want jup init guidance", check.Suggestion)
	}
}

func TestDoctorReportsUnreadableConfiguration(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := &doctorConfigFinder{err: errors.New("decode project configuration: invalid JSON")}
	report, err := NewDoctor(store, nil).Diagnose(context.Background(), root)
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	if report.Failed() != 1 {
		t.Fatalf("Failed() = %d, want 1", report.Failed())
	}
	assertDoctorCheck(t, report, "Configuration", CheckFailed, "invalid JSON")
	check := findDoctorCheck(t, report, "Configuration")
	if !strings.Contains(check.Suggestion, "jup init") {
		t.Errorf("suggestion = %q, want jup init guidance", check.Suggestion)
	}
}

func TestDoctorWarnsWhenJDKVersionChangedInPlace(t *testing.T) {
	t.Parallel()

	report := diagnoseDoctorProject(t, `
<project>
  <modelVersion>4.0.0</modelVersion>
  <properties><maven.compiler.release>17</maven.compiler.release></properties>
</project>`, "17.0.12", "17.0.13")
	if report.Failed() != 0 || report.Warnings() != 1 {
		t.Fatalf("failed/warnings = %d/%d, want 0/1", report.Failed(), report.Warnings())
	}
	assertDoctorCheck(t, report, "JDK", CheckWarning, "saved version was 17.0.12")
}

func diagnoseDoctorProject(t *testing.T, pom, savedVersion, inspectedVersion string) DoctorReport {
	t.Helper()
	root := t.TempDir()
	writeDoctorFile(t, filepath.Join(root, "pom.xml"), pom, 0o600)
	executable := filepath.Join(root, "mvn")
	writeDoctorFile(t, executable, "#!/bin/sh\n", 0o755)
	javaHome := filepath.Join(root, "jdk-"+strings.SplitN(savedVersion, ".", 2)[0])
	doctor := NewDoctor(&doctorConfigFinder{
		found: true,
		config: Config{
			ProjectRoot: root,
			BuildTool: buildtool.Info{
				Type:       buildtool.Maven,
				Version:    "3.9.11",
				Executable: executable,
			},
			Java: javainfo.Installation{Version: savedVersion, Home: javaHome},
		},
	}, &doctorSettingsResolver{})
	doctor.inspectJDK = func(context.Context, string) (javainfo.Installation, error) {
		return javainfo.Installation{Version: inspectedVersion, Home: javaHome}, nil
	}

	report, err := doctor.Diagnose(context.Background(), root)
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	return report
}

func assertDoctorCheck(
	t *testing.T,
	report DoctorReport,
	name string,
	status CheckStatus,
	messagePart string,
) {
	t.Helper()
	check := findDoctorCheck(t, report, name)
	if check.Status != status {
		t.Errorf("%s status = %q, want %q", name, check.Status, status)
	}
	if !strings.Contains(check.Message, messagePart) {
		t.Errorf("%s message = %q, want substring %q", name, check.Message, messagePart)
	}
}

func findDoctorCheck(t *testing.T, report DoctorReport, name string) DoctorCheck {
	t.Helper()
	for _, check := range report.Checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("check %q not found in %#v", name, report.Checks)
	return DoctorCheck{}
}

func writeDoctorFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
