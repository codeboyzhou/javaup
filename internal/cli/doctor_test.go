package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/codeboyzhou/javaup/internal/project"
)

type recordingDoctor struct {
	root   string
	report project.DoctorReport
	err    error
}

func (d *recordingDoctor) Diagnose(_ context.Context, root string) (project.DoctorReport, error) {
	d.root = root
	return d.report, d.err
}

func TestDoctorCommandShowsHealthyReport(t *testing.T) {
	t.Parallel()

	doctor := &recordingDoctor{report: project.DoctorReport{
		ProjectRoot: "/projects/demo",
		Checks: []project.DoctorCheck{
			{Name: "Configuration", Status: project.CheckPassed, Message: "Configuration is readable"},
			{Name: "Java requirement", Status: project.CheckWarning, Message: "No Java version", Suggestion: "Declare one"},
		},
	}}
	command := newDoctorCommand(func() (projectDoctor, error) { return doctor, nil }, func() (string, error) {
		return "/projects/demo/module", nil
	})
	var output bytes.Buffer
	command.SetOut(&output)

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if doctor.root != "/projects/demo/module" {
		t.Errorf("Diagnose() root = %q, want module path", doctor.root)
	}
	assertContains(t, output.String(), []string{
		"Project: /projects/demo",
		"[PASS] Configuration - Configuration is readable",
		"[WARN] Java requirement - No Java version",
		"Fix: Declare one",
		"Healthy with 1 warning.",
	})
}

func TestDoctorCommandReturnsFailureAfterRenderingReport(t *testing.T) {
	t.Parallel()

	doctor := &recordingDoctor{report: project.DoctorReport{
		ProjectRoot: "/projects/demo",
		Checks: []project.DoctorCheck{{
			Name:       "JDK",
			Status:     project.CheckFailed,
			Message:    "Saved JDK is missing",
			Suggestion: "Run jup init again",
		}},
	}}
	command := newDoctorCommand(
		func() (projectDoctor, error) { return doctor, nil },
		func() (string, error) { return "/projects/demo", nil },
	)
	var output bytes.Buffer
	command.SetOut(&output)

	err := command.ExecuteContext(context.Background())
	var unhealthy unhealthyProjectError
	if !errors.As(err, &unhealthy) {
		t.Fatalf("ExecuteContext() error = %v, want unhealthyProjectError", err)
	}
	if unhealthy.ExitCode() != exitFailure {
		t.Errorf("ExitCode() = %d, want %d", unhealthy.ExitCode(), exitFailure)
	}
	assertContains(t, output.String(), []string{
		"[FAIL] JDK - Saved JDK is missing",
		"Fix: Run jup init again",
		"Unhealthy: 1 failure, 0 warnings.",
	})
}

func TestDoctorCommandReturnsDiagnosisError(t *testing.T) {
	t.Parallel()

	want := errors.New("diagnosis failure")
	command := newDoctorCommand(
		func() (projectDoctor, error) { return &recordingDoctor{err: want}, nil },
		func() (string, error) { return "/projects/demo", nil },
	)

	if err := command.ExecuteContext(context.Background()); !errors.Is(err, want) {
		t.Fatalf("ExecuteContext() error = %v, want %v", err, want)
	}
}
