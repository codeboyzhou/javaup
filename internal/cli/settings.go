package cli

import (
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"

	"github.com/codeboyzhou/javaup/internal/mavensettings"
	"github.com/codeboyzhou/javaup/internal/project"
)

type mavenSettingsStore interface {
	Add(alias, path string) (entry mavensettings.Entry, registryPath string, err error)
	List() ([]mavensettings.Entry, error)
}

type mavenSettingsFactory func() (mavenSettingsStore, error)

type projectMavenSettingsUser interface {
	Use(root, alias string) (project.Config, mavensettings.Entry, error)
}

type projectMavenSettingsFactory func() (projectMavenSettingsUser, error)

func newSettingsCommand(
	settingsFactory mavenSettingsFactory,
	projectFactory projectMavenSettingsFactory,
	workingDirectory func() (string, error),
) *cobra.Command {
	command := &cobra.Command{
		Use:   "settings",
		Short: "Manage named Maven settings files",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
	}
	command.AddCommand(newSettingsAddCommand(settingsFactory))
	command.AddCommand(newSettingsListCommand(settingsFactory))
	command.AddCommand(newSettingsUseCommand(projectFactory, workingDirectory))
	return command
}

func newSettingsListCommand(factory mavenSettingsFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved Maven settings aliases",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			store, err := factory()
			if err != nil {
				return err
			}
			entries, err := store.List()
			if err != nil {
				return err
			}
			writer := command.OutOrStdout()
			if len(entries) == 0 {
				_, err = fmt.Fprintln(writer, "No Maven settings aliases configured.")
				return err
			}

			return writeSettingsTable(writer, entries)
		},
	}
}

func writeSettingsTable(writer io.Writer, entries []mavensettings.Entry) error {
	table := tablewriter.NewTable(
		writer,
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleASCII),
		})),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	table.Header("ALIAS", "PATH")

	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, []string{entry.Alias, entry.Path})
	}
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
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

func newSettingsUseCommand(
	factory projectMavenSettingsFactory,
	workingDirectory func() (string, error),
) *cobra.Command {
	return &cobra.Command{
		Use:   "use <alias>",
		Short: "Use a Maven settings alias for the current project",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			root, err := workingDirectory()
			if err != nil {
				return fmt.Errorf("resolve current directory: %w", err)
			}
			manager, err := factory()
			if err != nil {
				return err
			}
			config, entry, err := manager.Use(root, args[0])
			if err != nil {
				return err
			}

			writer := command.OutOrStdout()
			success := newOutputStyle(writer, color.FgGreen)
			message := fmt.Sprintf(
				"Configured project %s to use Maven settings alias %q at %s.",
				config.ProjectRoot,
				entry.Alias,
				entry.Path,
			)
			_, err = fmt.Fprintln(writer, success.Sprint(message))
			return err
		},
	}
}

func defaultMavenSettingsFactory() (mavenSettingsStore, error) {
	return mavensettings.NewDefaultStore()
}

func defaultProjectMavenSettingsFactory() (projectMavenSettingsUser, error) {
	return project.NewDefaultMavenSettingsManager()
}
