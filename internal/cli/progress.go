package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/fatih/color"

	"github.com/codeboyzhou/javaup/internal/project"
)

type progressRenderer struct {
	writer  io.Writer
	stage   *color.Color
	info    *color.Color
	success *color.Color
	failure *color.Color
	err     error
}

func newProgressRenderer(writer io.Writer) *progressRenderer {
	return &progressRenderer{
		writer:  writer,
		stage:   newOutputStyle(writer, color.FgBlue),
		info:    newOutputStyle(writer, color.FgCyan),
		success: newOutputStyle(writer, color.FgGreen),
		failure: newOutputStyle(writer, color.FgRed),
	}
}

func newOutputStyle(writer io.Writer, attributes ...color.Attribute) *color.Color {
	style := color.New(attributes...)
	if _, interactive := writer.(*os.File); !interactive {
		style.DisableColor()
	}
	return style
}

func (r *progressRenderer) Report(event project.ProgressEvent) {
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

func (r *progressRenderer) Success(message string) string {
	return r.success.Sprint(message)
}

func (r *progressRenderer) Err() error {
	return r.err
}

type projectProgressAction func(context.Context, string, project.ProgressFunc) error
type projectProgressActionFactory func() (projectProgressAction, error)

func runProjectProgressCommand(
	command *cobra.Command,
	workingDirectory func() (string, error),
	factory projectProgressActionFactory,
	successMessage string,
) error {
	root, err := workingDirectory()
	if err != nil {
		return fmt.Errorf("resolve current directory: %w", err)
	}
	action, err := factory()
	if err != nil {
		return err
	}
	progress := newProgressRenderer(command.OutOrStdout())
	err = action(command.Context(), root, progress.Report)
	if progress.Err() != nil {
		return progress.Err()
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(command.OutOrStdout(), progress.Success(successMessage))
	return err
}
