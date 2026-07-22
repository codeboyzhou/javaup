package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/project"
)

type projectRunner interface {
	Run(
		ctx context.Context,
		root string,
		tool buildtool.Type,
		args []string,
		streams project.Streams,
	) error
}

type runnerFactory func() (projectRunner, error)

type projectPicker interface {
	Pick(ctx context.Context, tool buildtool.Type, streams project.Streams) (string, error)
}

type projectPickerFactory func() (projectPicker, error)

type interactiveTerminal func(stdin io.Reader, stdout io.Writer) bool

func newRunCommand(factory runnerFactory, workingDirectory func() (string, error)) *cobra.Command {
	return newRunCommandWithPicker(factory, workingDirectory, defaultProjectPickerFactory, isInteractiveTerminal)
}

func newRunCommandWithPicker(
	factory runnerFactory,
	workingDirectory func() (string, error),
	pickerFactory projectPickerFactory,
	isInteractive interactiveTerminal,
) *cobra.Command {
	command := &cobra.Command{
		Use:   "run",
		Short: "Run a project build tool with its detected toolchain",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
	}
	buildTools := []struct {
		name        string
		tool        buildtool.Type
		description string
	}{
		{name: "mvn", tool: buildtool.Maven, description: "Run the detected project Maven"},
	}
	for _, buildTool := range buildTools {
		command.AddCommand(newRunBuildToolCommand(
			buildTool.name,
			buildTool.tool,
			buildTool.description,
			factory,
			workingDirectory,
			pickerFactory,
			isInteractive,
		))
	}
	return command
}

func newRunBuildToolCommand(
	name string,
	tool buildtool.Type,
	description string,
	factory runnerFactory,
	workingDirectory func() (string, error),
	pickerFactory projectPickerFactory,
	isInteractive interactiveTerminal,
) *cobra.Command {
	return &cobra.Command{
		Use:                name + " [arguments...]",
		Short:              description,
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(command *cobra.Command, args []string) error {
			streams := project.Streams{
				Stdin:  command.InOrStdin(),
				Stdout: command.OutOrStdout(),
				Stderr: command.ErrOrStderr(),
			}
			var root string
			var err error
			if isInteractive(streams.Stdin, streams.Stdout) {
				picker, pickerErr := pickerFactory()
				if pickerErr != nil {
					return pickerErr
				}
				root, err = picker.Pick(command.Context(), tool, streams)
			} else {
				root, err = workingDirectory()
				if err != nil {
					err = fmt.Errorf("resolve current directory: %w", err)
				}
			}
			if err != nil {
				return err
			}
			runner, err := factory()
			if err != nil {
				return err
			}
			return runner.Run(command.Context(), root, tool, args, streams)
		},
	}
}

func defaultProjectPickerFactory() (projectPicker, error) {
	catalog, err := project.NewDefaultCatalog()
	if err != nil {
		return nil, err
	}
	return newTerminalProjectPicker(catalog), nil
}

func defaultRunnerFactory() (projectRunner, error) {
	return project.NewDefaultRunner()
}
