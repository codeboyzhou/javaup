package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/codeboyzhou/javaup/internal/project"
)

type projectUninitializer interface {
	Uninitialize(ctx context.Context, root string, progress project.ProgressFunc) (path string, removed bool, err error)
}

type uninitializerFactory func() (projectUninitializer, error)

func newUninitCommand(factory uninitializerFactory, workingDirectory func() (string, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "uninit",
		Short: "Remove javaup configuration for the current project",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			root, err := workingDirectory()
			if err != nil {
				return fmt.Errorf("resolve current directory: %w", err)
			}
			uninitializer, err := factory()
			if err != nil {
				return err
			}

			progress := newProgressRenderer(command.OutOrStdout())
			_, _, err = uninitializer.Uninitialize(command.Context(), root, progress.Report)
			if progress.Err() != nil {
				return progress.Err()
			}
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(command.OutOrStdout(), progress.Success("Uninitialized javaup project."))
			return err
		},
	}
}

func defaultUninitializerFactory() (projectUninitializer, error) {
	return project.NewDefaultUninitializer()
}
