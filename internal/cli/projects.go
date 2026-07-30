package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"

	"github.com/codeboyzhou/javaup/internal/project"
)

type projectsManager interface {
	List() ([]project.RegistryEntry, []error, error)
	Remove(ctx context.Context, root string) (project.RegistryEntry, bool, error)
	Prune(ctx context.Context, dryRun bool) (project.PruneResult, error)
}

type projectsFactory func() (projectsManager, error)

func newProjectsCommand(factory projectsFactory) *cobra.Command {
	command := &cobra.Command{
		Use:   "projects",
		Short: "Manage initialized projects",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
	}
	command.AddCommand(newProjectsListCommand(factory))
	command.AddCommand(newProjectsPruneCommand(factory))
	command.AddCommand(newProjectsRemoveCommand(factory))
	return command
}

func newProjectsListCommand(factory projectsFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all initialized projects",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			manager, err := factory()
			if err != nil {
				return err
			}
			entries, warnings, err := manager.List()
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				writeProjectsWarnings(command.ErrOrStderr(), warnings, entries)
				_, err = fmt.Fprintln(command.OutOrStdout(), "No initialized projects found.")
				return err
			}
			if err := writeProjectsTable(command.OutOrStdout(), entries); err != nil {
				return err
			}
			writeProjectsWarnings(command.ErrOrStderr(), warnings, entries)
			return nil
		},
	}
}

func writeProjectsTable(writer io.Writer, entries []project.RegistryEntry) error {
	return writeProjectsTableWithStyle(
		writer,
		entries,
		newOutputStyle(writer, color.FgYellow),
	)
}

func writeProjectsTableWithStyle(
	writer io.Writer,
	entries []project.RegistryEntry,
	warning *color.Color,
) error {
	table := tablewriter.NewTable(
		writer,
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleASCII),
		})),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	table.Header("NAME", "TOOL", "JAVA", "STATUS", "USES", "LAST USED", "PATH")

	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		tool := entry.BuildTool.DisplayName()
		if tool == "" {
			tool = "-"
		}
		javaVersion := entry.JavaVersion
		if javaVersion == "" {
			javaVersion = "-"
		}
		lastUsed := "-"
		if !entry.LastUsedAt.IsZero() {
			lastUsed = entry.LastUsedAt.Local().Format("2006-01-02 15:04")
		}
		row := []string{
			entry.Name,
			tool,
			javaVersion,
			string(entry.Status),
			strconv.FormatUint(entry.UseCount, 10),
			lastUsed,
			entry.DisplayPath(),
		}
		if entry.Status != project.RegistryAvailable {
			for index := range row {
				row[index] = warning.Sprint(row[index])
			}
		}
		rows = append(rows, row)
	}
	if err := table.Bulk(rows); err != nil {
		return err
	}
	return table.Render()
}

func writeProjectsWarnings(
	writer io.Writer,
	warnings []error,
	entries []project.RegistryEntry,
) {
	writeProjectsWarningsWithStyle(
		writer,
		warnings,
		entries,
		newOutputStyle(writer, color.FgYellow),
		newOutputStyle(writer, color.FgGreen),
	)
}

func writeProjectsWarningsWithStyle(
	writer io.Writer,
	warnings []error,
	entries []project.RegistryEntry,
	warningStyle *color.Color,
	hintStyle *color.Color,
) {
	for _, warning := range warnings {
		_, _ = fmt.Fprintln(writer, warningStyle.Sprintf("jup: warning: %v", warning))
	}

	hasProblem := false
	for _, entry := range entries {
		if entry.Status != project.RegistryAvailable {
			hasProblem = true
		}
		if entry.Detail != "" {
			_, _ = fmt.Fprintln(
				writer,
				warningStyle.Sprintf(
					"jup: warning: %s: %s",
					entry.DisplayPath(),
					entry.Detail,
				),
			)
		}
	}
	if hasProblem {
		_, _ = fmt.Fprintln(
			writer,
			hintStyle.Sprint(
				"jup: hint: run `jup projects prune` to remove missing or invalid project records.",
			),
		)
	}
}

func newProjectsRemoveCommand(factory projectsFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <path>",
		Short: "Remove a project by path",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			manager, err := factory()
			if err != nil {
				return err
			}
			entry, removed, err := manager.Remove(command.Context(), args[0])
			if err != nil {
				return err
			}
			if !removed {
				_, err = fmt.Fprintf(
					command.OutOrStdout(),
					"No saved project configuration found for %s.\n",
					entry.ProjectRoot,
				)
				return err
			}
			success := newOutputStyle(command.OutOrStdout(), color.FgGreen)
			_, err = fmt.Fprintln(
				command.OutOrStdout(),
				success.Sprintf("Removed project %s.", entry.ProjectRoot),
			)
			return err
		},
	}
}

func newProjectsPruneCommand(factory projectsFactory) *cobra.Command {
	var dryRun bool
	command := &cobra.Command{
		Use:   "prune",
		Short: "Remove missing or invalid project records",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			manager, err := factory()
			if err != nil {
				return err
			}
			result, err := manager.Prune(command.Context(), dryRun)
			if err != nil {
				return err
			}
			writer := command.OutOrStdout()
			if len(result.Projects) == 0 && result.UsageRecords == 0 {
				_, err = fmt.Fprintln(writer, "No stale project records found.")
				return err
			}

			action := "Removed"
			if result.DryRun {
				action = "Would remove"
			}
			for _, entry := range result.Projects {
				if _, err := fmt.Fprintf(
					writer,
					"%s %s project record %s.\n",
					action,
					entry.Status,
					entry.DisplayPath(),
				); err != nil {
					return err
				}
			}
			if result.UsageRecords > 0 {
				if _, err := fmt.Fprintf(
					writer,
					"%s %s.\n",
					action,
					formatProjectsCount(result.UsageRecords, "orphaned usage record"),
				); err != nil {
					return err
				}
			}
			if result.DryRun {
				_, err = fmt.Fprintln(writer, "Dry run complete; no files were changed.")
				return err
			}
			success := newOutputStyle(writer, color.FgGreen)
			_, err = fmt.Fprintln(writer, success.Sprint("Pruned stale project records."))
			return err
		},
	}
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be removed without changing files")
	return command
}

func formatProjectsCount(count int, noun string) string {
	if count != 1 {
		noun += "s"
	}
	return fmt.Sprintf("%d %s", count, noun)
}

func defaultProjectsFactory() (projectsManager, error) {
	return project.NewDefaultRegistry()
}
