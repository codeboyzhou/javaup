package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/codeboyzhou/javaup/internal/project"
)

type projectInitializer interface {
	Initialize(ctx context.Context, root string, progress project.ProgressFunc) (project.Config, string, error)
}

type initializerFactory func() (projectInitializer, error)

func newInitCommand(factory initializerFactory, workingDirectory func() (string, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Detect and initialize the current Java project",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			root, err := workingDirectory()
			if err != nil {
				return fmt.Errorf("resolve current directory: %w", err)
			}
			initializer, err := factory()
			if err != nil {
				return err
			}

			progress := newInitProgressRenderer(command.OutOrStdout())
			_, _, err = initializer.Initialize(command.Context(), root, progress.Report)
			if progress.Err() != nil {
				return progress.Err()
			}
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(command.OutOrStdout(), progress.Success("Initialized javaup project."))
			return err
		},
	}
}

type initProgressRenderer struct {
	writer  io.Writer
	stage   *color.Color
	info    *color.Color
	success *color.Color
	failure *color.Color
	err     error
}

func newInitProgressRenderer(writer io.Writer) *initProgressRenderer {
	style := func(attributes ...color.Attribute) *color.Color {
		value := color.New(attributes...)
		if _, interactive := writer.(*os.File); !interactive {
			value.DisableColor()
		}
		return value
	}

	return &initProgressRenderer{
		writer:  writer,
		stage:   style(color.FgBlue),
		info:    style(color.FgCyan),
		success: style(color.FgGreen),
		failure: style(color.FgRed),
	}
}

func (r *initProgressRenderer) Report(event project.ProgressEvent) {
	if r.err != nil {
		return
	}

	stage := fmt.Sprintf("[%d/%d] %s", event.Step, event.Total, event.Name)
	var line string
	switch event.State {
	case project.ProgressStarted:
		line = fmt.Sprintf("%s - %s\n", r.stage.Sprint(stage), event.Message)
	case project.ProgressInfo:
		line = fmt.Sprintf("      %s\n", r.info.Sprint(event.Message))
	case project.ProgressSucceeded:
		line = fmt.Sprintf("%s - %s\n", r.success.Sprint(stage+" OK"), event.Message)
	case project.ProgressFailed:
		line = fmt.Sprintf("%s - %s\n", r.failure.Sprint(stage+" Failed"), event.Message)
	default:
		return
	}
	_, r.err = fmt.Fprint(r.writer, line)
}

func (r *initProgressRenderer) Success(message string) string {
	return r.success.Sprint(message)
}

func (r *initProgressRenderer) Err() error {
	return r.err
}

func defaultInitializerFactory() (projectInitializer, error) {
	return project.NewDefaultInitializer()
}

func defaultWorkingDirectory() (string, error) {
	return os.Getwd()
}
