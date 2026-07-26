package cli

import (
	"context"
	"os"

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
			return runProjectProgressCommand(command, workingDirectory, func() (projectProgressAction, error) {
				initializer, err := factory()
				if err != nil {
					return nil, err
				}
				return func(ctx context.Context, root string, progress project.ProgressFunc) error {
					_, _, err := initializer.Initialize(ctx, root, progress)
					return err
				}, nil
			}, "Initialized javaup project.")
		},
	}
}

func defaultInitializerFactory() (projectInitializer, error) {
	return project.NewDefaultInitializer()
}

func defaultWorkingDirectory() (string, error) {
	return os.Getwd()
}
