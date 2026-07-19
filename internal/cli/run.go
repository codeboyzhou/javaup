package cli

import (
	"context"
	"fmt"

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

func newRunCommand(factory runnerFactory, workingDirectory func() (string, error)) *cobra.Command {
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
) *cobra.Command {
	return &cobra.Command{
		Use:                name + " [arguments...]",
		Short:              description,
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(command *cobra.Command, args []string) error {
			root, err := workingDirectory()
			if err != nil {
				return fmt.Errorf("resolve current directory: %w", err)
			}
			runner, err := factory()
			if err != nil {
				return err
			}
			return runner.Run(command.Context(), root, tool, args, project.Streams{
				Stdin:  command.InOrStdin(),
				Stdout: command.OutOrStdout(),
				Stderr: command.ErrOrStderr(),
			})
		},
	}
}

func defaultRunnerFactory() (projectRunner, error) {
	return project.NewDefaultRunner()
}
