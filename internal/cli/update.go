package cli

import (
	"context"
	"fmt"

	"github.com/codeboyzhou/javaup/internal/selfupdate"
	"github.com/spf13/cobra"
)

type updateService interface {
	Check(context.Context) (selfupdate.Result, error)
	Update(context.Context) (selfupdate.Result, error)
}

func newUpdateCommand(newService func() updateService) *cobra.Command {
	var checkOnly bool
	command := &cobra.Command{
		Use:   "update",
		Short: "Update jup to the latest release",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			service := newService()
			if checkOnly {
				result, err := service.Check(command.Context())
				if err != nil {
					return err
				}
				if result.Updated {
					_, err = fmt.Fprintf(
						command.OutOrStdout(),
						"Update available: %s -> %s\n",
						result.Current,
						result.Latest,
					)
					return err
				}
				_, err = fmt.Fprintf(command.OutOrStdout(), "Already up to date (%s)\n", result.Current)
				return err
			}

			result, err := service.Update(command.Context())
			if err != nil {
				return err
			}
			if !result.Updated {
				_, err = fmt.Fprintf(command.OutOrStdout(), "Already up to date (%s)\n", result.Current)
				return err
			}
			if result.Pending {
				_, err = fmt.Fprintf(
					command.OutOrStdout(),
					"Downloaded jup %s; it will be installed after this process exits.\n",
					result.Latest,
				)
				return err
			}
			_, err = fmt.Fprintf(
				command.OutOrStdout(),
				"Updated jup from %s to %s.\n",
				result.Current,
				result.Latest,
			)
			return err
		},
	}
	command.Flags().BoolVar(&checkOnly, "check", false, "Check for an update without installing it")
	return command
}

func defaultUpdateService(currentVersion string) func() updateService {
	return func() updateService {
		return selfupdate.New(currentVersion)
	}
}
