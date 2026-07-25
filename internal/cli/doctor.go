package cli

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/codeboyzhou/javaup/internal/project"
)

type projectDoctor interface {
	Diagnose(ctx context.Context, root string) (project.DoctorReport, error)
}

type doctorFactory func() (projectDoctor, error)

type unhealthyProjectError struct {
	failed int
}

func (e unhealthyProjectError) Error() string {
	return fmt.Sprintf("project has %d failed health checks", e.failed)
}

func (e unhealthyProjectError) ExitCode() int {
	return exitFailure
}

func newDoctorCommand(factory doctorFactory, workingDirectory func() (string, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose the current project's saved toolchain",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			root, err := workingDirectory()
			if err != nil {
				return fmt.Errorf("resolve current directory: %w", err)
			}
			doctor, err := factory()
			if err != nil {
				return err
			}
			report, err := doctor.Diagnose(command.Context(), root)
			if err != nil {
				return err
			}
			if err := renderDoctorReport(command, report); err != nil {
				return err
			}
			if failed := report.Failed(); failed > 0 {
				return unhealthyProjectError{failed: failed}
			}
			return nil
		},
	}
}

func renderDoctorReport(command *cobra.Command, report project.DoctorReport) error {
	writer := command.OutOrStdout()
	label := newOutputStyle(writer, color.FgCyan)
	passed := newOutputStyle(writer, color.FgGreen)
	warning := newOutputStyle(writer, color.FgYellow)
	failed := newOutputStyle(writer, color.FgRed)

	if _, err := fmt.Fprintf(writer, "%s %s\n\n", label.Sprint("Project:"), report.ProjectRoot); err != nil {
		return err
	}
	for _, check := range report.Checks {
		var marker string
		switch check.Status {
		case project.CheckPassed:
			marker = passed.Sprint("[PASS]")
		case project.CheckWarning:
			marker = warning.Sprint("[WARN]")
		case project.CheckFailed:
			marker = failed.Sprint("[FAIL]")
		default:
			continue
		}
		if _, err := fmt.Fprintf(writer, "%s %s - %s\n", marker, check.Name, check.Message); err != nil {
			return err
		}
		if check.Suggestion != "" {
			if _, err := fmt.Fprintf(writer, "       Fix: %s\n", check.Suggestion); err != nil {
				return err
			}
		}
	}

	failureCount := report.Failed()
	warningCount := report.Warnings()
	if _, err := fmt.Fprintln(writer); err != nil {
		return err
	}
	switch {
	case failureCount > 0:
		_, err := fmt.Fprintf(
			writer,
			"%s\n",
			failed.Sprintf(
				"Unhealthy: %s, %s.",
				formatDoctorCount(failureCount, "failure"),
				formatDoctorCount(warningCount, "warning"),
			),
		)
		return err
	case warningCount > 0:
		_, err := fmt.Fprintf(
			writer,
			"%s\n",
			warning.Sprintf("Healthy with %s.", formatDoctorCount(warningCount, "warning")),
		)
		return err
	default:
		_, err := fmt.Fprintln(writer, passed.Sprint("Healthy: all checks passed."))
		return err
	}
}

func formatDoctorCount(count int, noun string) string {
	if count != 1 {
		noun += "s"
	}
	return fmt.Sprintf("%d %s", count, noun)
}

func defaultDoctorFactory() (projectDoctor, error) {
	return project.NewDefaultDoctor()
}
