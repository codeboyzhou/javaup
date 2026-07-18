package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(command.OutOrStdout(), version)
			return err
		},
	}
}

func formatVersion(options Options) string {
	return fmt.Sprintf(
		"%s version %s %s (%s)",
		options.ProductName,
		options.Version,
		options.Platform,
		options.Commit,
	)
}
