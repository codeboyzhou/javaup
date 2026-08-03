package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/buildtool/maven"
	"github.com/codeboyzhou/javaup/internal/javainfo"
	"github.com/codeboyzhou/javaup/internal/mavensettings"
	"github.com/codeboyzhou/javaup/internal/pathutil"
)

// CheckStatus describes the outcome of one project health check.
type CheckStatus string

const (
	// CheckPassed means the configured resource is healthy.
	CheckPassed CheckStatus = "pass"
	// CheckWarning means the project is usable but could be more explicit.
	CheckWarning CheckStatus = "warn"
	// CheckFailed means the saved toolchain cannot be used safely.
	CheckFailed CheckStatus = "fail"
)

// DoctorCheck is one actionable project health result.
type DoctorCheck struct {
	Name       string
	Status     CheckStatus
	Message    string
	Suggestion string
}

// DoctorReport contains all health checks for one initialized project.
type DoctorReport struct {
	ProjectRoot string
	Checks      []DoctorCheck
}

// Failed returns the number of failed checks.
func (r DoctorReport) Failed() int {
	total := 0
	for _, check := range r.Checks {
		if check.Status == CheckFailed {
			total++
		}
	}
	return total
}

// Warnings returns the number of warning checks.
func (r DoctorReport) Warnings() int {
	total := 0
	for _, check := range r.Checks {
		if check.Status == CheckWarning {
			total++
		}
	}
	return total
}

type doctorJDKInspector func(ctx context.Context, home string) (javainfo.Installation, error)
type doctorJavaVersionDetector func(root string) (string, error)

// Doctor validates the saved environment for the project containing a directory.
type Doctor struct {
	store             configFinder
	settings          mavenSettingsResolver
	inspectJDK        doctorJDKInspector
	detectJavaVersion doctorJavaVersionDetector
}

// NewDefaultDoctor creates a doctor backed by the user's saved configuration.
func NewDefaultDoctor() (*Doctor, error) {
	projects, err := NewDefaultConfigStore()
	if err != nil {
		return nil, err
	}
	settings, err := mavensettings.NewDefaultStore()
	if err != nil {
		return nil, err
	}
	return NewDoctor(projects, settings), nil
}

// NewDoctor creates a doctor from replaceable project and settings stores.
func NewDoctor(store configFinder, settings mavenSettingsResolver) *Doctor {
	return &Doctor{
		store:             store,
		settings:          settings,
		inspectJDK:        javainfo.Inspect,
		detectJavaVersion: maven.DetectBuildJavaVersion,
	}
}

// Diagnose checks the saved toolchain without changing project configuration.
func (d *Doctor) Diagnose(ctx context.Context, start string) (DoctorReport, error) {
	config, _, found, err := d.store.Find(start)
	if err != nil {
		root, rootErr := pathutil.Canonical(start)
		if rootErr != nil {
			return DoctorReport{}, rootErr
		}
		return DoctorReport{
			ProjectRoot: root,
			Checks: []DoctorCheck{{
				Name:       "Configuration",
				Status:     CheckFailed,
				Message:    err.Error(),
				Suggestion: "Run `jup init` from the project root to recreate the configuration",
			}},
		}, nil
	}
	if !found {
		root, rootErr := pathutil.Canonical(start)
		if rootErr != nil {
			return DoctorReport{}, rootErr
		}
		return DoctorReport{
			ProjectRoot: root,
			Checks: []DoctorCheck{{
				Name:       "Configuration",
				Status:     CheckFailed,
				Message:    "No initialized javaup project was found",
				Suggestion: "Run `jup init` from the project root",
			}},
		}, nil
	}

	report := DoctorReport{ProjectRoot: config.ProjectRoot}
	report.Checks = append(report.Checks, DoctorCheck{
		Name:    "Configuration",
		Status:  CheckPassed,
		Message: "Saved project configuration is readable",
	})

	requiredJava := d.checkProject(&report, config)
	d.checkBuildTool(&report, config)
	d.checkJDK(ctx, &report, config, requiredJava)
	d.checkMavenSettings(&report, config)
	return report, nil
}

func (d *Doctor) checkProject(report *DoctorReport, config Config) string {
	if config.BuildTool.Type != buildtool.Maven {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:       "Project",
			Status:     CheckFailed,
			Message:    fmt.Sprintf("Unsupported saved build tool %q", config.BuildTool.Type),
			Suggestion: "Run `jup init` after installing a supported build tool",
		})
		return ""
	}

	version, err := d.detectJavaVersion(config.ProjectRoot)
	if err != nil {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:       "Project",
			Status:     CheckFailed,
			Message:    err.Error(),
			Suggestion: "Fix pom.xml, then run `jup init` again",
		})
		return ""
	}
	if version == "" {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:       "Java requirement",
			Status:     CheckWarning,
			Message:    "pom.xml does not declare a Java build version",
			Suggestion: "Declare maven.compiler.release or java.version in pom.xml",
		})
		return ""
	}
	report.Checks = append(report.Checks, DoctorCheck{
		Name:    "Java requirement",
		Status:  CheckPassed,
		Message: "pom.xml requires Java " + version,
	})
	return version
}

func (d *Doctor) checkBuildTool(report *DoctorReport, config Config) {
	name := config.BuildTool.Type.DisplayName()
	executable := config.BuildTool.Executable
	info, err := os.Stat(executable)
	if err != nil {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:       name,
			Status:     CheckFailed,
			Message:    fmt.Sprintf("Saved executable is unavailable: %s", executable),
			Suggestion: "Restore the executable or run `jup init` again",
		})
		return
	}
	if !info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0) {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:       name,
			Status:     CheckFailed,
			Message:    fmt.Sprintf("Saved executable is not runnable: %s", executable),
			Suggestion: "Fix its permissions or run `jup init` again",
		})
		return
	}

	source := "PATH"
	if config.BuildTool.Wrapper {
		source = "wrapper"
	}
	report.Checks = append(report.Checks, DoctorCheck{
		Name:   name,
		Status: CheckPassed,
		Message: fmt.Sprintf(
			"%s %s (%s) is available at %s",
			name,
			config.BuildTool.Version,
			source,
			executable,
		),
	})
}

func (d *Doctor) checkJDK(
	ctx context.Context,
	report *DoctorReport,
	config Config,
	requiredVersion string,
) {
	installation, err := d.inspectJDK(ctx, config.Java.Home)
	if err != nil {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:       "JDK",
			Status:     CheckFailed,
			Message:    err.Error(),
			Suggestion: "Restore the saved JDK or run `jup init` again",
		})
		return
	}
	if requiredVersion != "" && !javainfo.MatchesVersion(installation.Version, requiredVersion) {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:   "JDK",
			Status: CheckFailed,
			Message: fmt.Sprintf(
				"Saved JDK is Java %s, but pom.xml requires Java %s",
				installation.Version,
				requiredVersion,
			),
			Suggestion: "Install a matching JDK and run `jup init` again",
		})
		return
	}

	message := fmt.Sprintf("Java %s is available at %s", installation.Version, installation.Home)
	if config.Java.Version != "" && config.Java.Version != installation.Version {
		message = fmt.Sprintf(
			"Java %s is available at %s; saved version was %s",
			installation.Version,
			installation.Home,
			config.Java.Version,
		)
		report.Checks = append(report.Checks, DoctorCheck{
			Name:       "JDK",
			Status:     CheckWarning,
			Message:    message,
			Suggestion: "Run `jup init` again to refresh the saved JDK version",
		})
		return
	}
	report.Checks = append(report.Checks, DoctorCheck{Name: "JDK", Status: CheckPassed, Message: message})
}

func (d *Doctor) checkMavenSettings(report *DoctorReport, config Config) {
	if config.BuildTool.Type != buildtool.Maven {
		return
	}
	alias := config.BuildTool.SettingsAlias
	if alias == "" {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:    "Maven settings",
			Status:  CheckPassed,
			Message: "Using Maven default settings",
		})
		return
	}
	if d.settings == nil {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:       "Maven settings",
			Status:     CheckFailed,
			Message:    fmt.Sprintf("Cannot resolve settings alias %q", alias),
			Suggestion: "Run `jup settings unset` or register the alias again",
		})
		return
	}
	entry, err := d.settings.Resolve(alias)
	if err != nil {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:    "Maven settings",
			Status:  CheckFailed,
			Message: err.Error(),
			Suggestion: fmt.Sprintf(
				"Run `jup settings add %s <file>`, or `jup settings unset`",
				alias,
			),
		})
		return
	}
	report.Checks = append(report.Checks, DoctorCheck{
		Name:    "Maven settings",
		Status:  CheckPassed,
		Message: fmt.Sprintf("Alias %q resolves to %s", entry.Alias, filepath.Clean(entry.Path)),
	})
}
