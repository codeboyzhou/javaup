package cli

import (
	"context"
	"fmt"

	"github.com/codeboyzhou/javaup/internal/uninstall"
	"github.com/spf13/cobra"
)

type applicationUninstaller interface {
	Run(context.Context) (uninstall.Result, error)
}

func newUninstallCommand(factory func(bool) applicationUninstaller) *cobra.Command {
	var purge bool
	command := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall jup from this computer",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			result, err := factory(purge).Run(command.Context())
			if err != nil {
				return err
			}
			if result.Pending {
				if result.Purged {
					_, err = fmt.Fprintln(
						command.OutOrStdout(),
						"Uninstall scheduled; jup and all javaup data will be removed after this process exits.",
					)
					return err
				}
				_, err = fmt.Fprintf(
					command.OutOrStdout(),
					"Uninstall scheduled; jup will be removed after this process exits. Configuration will remain at %s.\n",
					result.Home,
				)
				return err
			}
			if result.Purged {
				_, err = fmt.Fprintln(command.OutOrStdout(), "Uninstalled jup and removed all javaup data.")
				return err
			}
			_, err = fmt.Fprintf(
				command.OutOrStdout(),
				"Uninstalled jup. Configuration remains at %s.\n",
				result.Home,
			)
			return err
		},
	}
	command.Flags().BoolVar(
		&purge,
		"purge",
		false,
		"Also remove all project configurations and Maven settings aliases",
	)
	return command
}

func defaultApplicationUninstaller(purge bool) applicationUninstaller {
	return uninstall.New(purge)
}
