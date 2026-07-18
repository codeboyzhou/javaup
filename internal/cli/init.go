package cli

import (
	"context"
	"fmt"
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
			root, err := workingDirectory()
			if err != nil {
				return fmt.Errorf("resolve current directory: %w", err)
			}
			initializer, err := factory()
			if err != nil {
				return err
			}

			progress := newProgressRenderer(command.OutOrStdout())
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

func defaultInitializerFactory() (projectInitializer, error) {
	return project.NewDefaultInitializer()
}

func defaultWorkingDirectory() (string, error) {
	return os.Getwd()
}
