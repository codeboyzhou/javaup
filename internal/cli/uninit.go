package cli

import (
	"context"

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
			return runProjectProgressCommand(command, workingDirectory, func() (projectProgressAction, error) {
				uninitializer, err := factory()
				if err != nil {
					return nil, err
				}
				return func(ctx context.Context, root string, progress project.ProgressFunc) error {
					_, _, err := uninitializer.Uninitialize(ctx, root, progress)
					return err
				}, nil
			}, "Uninitialized javaup project.")
		},
	}
}

func defaultUninitializerFactory() (projectUninitializer, error) {
	return project.NewDefaultUninitializer()
}
