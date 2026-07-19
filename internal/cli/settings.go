package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/codeboyzhou/javaup/internal/mavensettings"
)

type mavenSettingsAdder interface {
	Add(alias, path string) (entry mavensettings.Entry, registryPath string, err error)
}

type mavenSettingsFactory func() (mavenSettingsAdder, error)

func newSettingsCommand(factory mavenSettingsFactory) *cobra.Command {
	command := &cobra.Command{
		Use:   "settings",
		Short: "Manage named Maven settings files",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
	}
	command.AddCommand(newSettingsAddCommand(factory))
	return command
}

func newSettingsAddCommand(factory mavenSettingsFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "add <alias> <path>",
		Short: "Add or update a Maven settings alias",
		Args:  cobra.ExactArgs(2),
		RunE: func(command *cobra.Command, args []string) error {
			store, err := factory()
			if err != nil {
				return err
			}
			entry, registryPath, err := store.Add(args[0], args[1])
			if err != nil {
				return err
			}
			writer := command.OutOrStdout()
			success := newOutputStyle(writer, color.FgGreen)
			message := fmt.Sprintf(
				"Saved Maven settings alias %q for %s in %s.",
				entry.Alias,
				entry.Path,
				registryPath,
			)
			_, err = fmt.Fprintln(writer, success.Sprint(message))
			return err
		},
	}
}

func defaultMavenSettingsFactory() (mavenSettingsAdder, error) {
	return mavensettings.NewDefaultStore()
}
