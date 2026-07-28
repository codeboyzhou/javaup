package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

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
	Pick(
		ctx context.Context,
		tool buildtool.Type,
		keyword string,
		currentDirectory string,
		interactive bool,
		streams project.Streams,
	) (string, error)
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
		Short: "Run Maven projects with their saved toolchains",
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
		{name: "mvn", tool: buildtool.Maven, description: "Run Maven with a saved project toolchain"},
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
		Use:                name + " [arguments...] [--project <keyword>]",
		Short:              description,
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(command *cobra.Command, args []string) error {
			mavenArgs, keyword, err := splitProjectKeyword(args)
			if err != nil {
				return err
			}
			streams := project.Streams{
				Stdin:  command.InOrStdin(),
				Stdout: command.OutOrStdout(),
				Stderr: command.ErrOrStderr(),
			}
			var root, currentDirectory string
			if keyword == "" {
				currentDirectory, err = workingDirectory()
				if err != nil {
					return fmt.Errorf("resolve current directory: %w", err)
				}
			}
			interactive := isInteractive(streams.Stdin, streams.Stdout)
			if interactive || keyword != "" {
				picker, pickerErr := pickerFactory()
				if pickerErr != nil {
					return pickerErr
				}
				root, err = picker.Pick(
					command.Context(), tool, keyword, currentDirectory, interactive, streams,
				)
			} else {
				root = currentDirectory
			}
			if err != nil {
				return err
			}
			runner, err := factory()
			if err != nil {
				return err
			}
			return runner.Run(command.Context(), root, tool, mavenArgs, streams)
		},
	}
}

func splitProjectKeyword(args []string) ([]string, string, error) {
	if len(args) == 0 {
		return args, "", nil
	}
	last := args[len(args)-1]
	if strings.HasPrefix(last, "--project=") {
		keyword := strings.TrimSpace(strings.TrimPrefix(last, "--project="))
		if keyword == "" {
			return nil, "", fmt.Errorf("project keyword cannot be empty")
		}
		return args[:len(args)-1], keyword, nil
	}
	if len(args) >= 2 && args[len(args)-2] == "--project" {
		keyword := strings.TrimSpace(last)
		if keyword == "" {
			return nil, "", fmt.Errorf("project keyword cannot be empty")
		}
		return args[:len(args)-2], keyword, nil
	}
	if last == "--project" {
		return nil, "", fmt.Errorf("--project requires a keyword")
	}
	return args, "", nil
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
