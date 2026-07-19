package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/codeboyzhou/javaup/internal/project"
)

type projectInspector interface {
	Inspect(root string) (project.Config, error)
}

type inspectorFactory func() (projectInspector, error)

func newStatusCommand(factory inspectorFactory, workingDirectory func() (string, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the current project's detected toolchain",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			root, err := workingDirectory()
			if err != nil {
				return fmt.Errorf("resolve current directory: %w", err)
			}
			inspector, err := factory()
			if err != nil {
				return err
			}
			config, err := inspector.Inspect(root)
			if err != nil {
				return err
			}

			writer := command.OutOrStdout()
			label := newOutputStyle(writer, color.FgCyan)
			_, err = fmt.Fprintf(
				writer,
				"%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n",
				label.Sprint("Project:"),
				config.ProjectRoot,
				label.Sprint("Build tool:"),
				config.BuildTool.Summary(),
				label.Sprint("Build executable:"),
				config.BuildTool.Executable,
				label.Sprint("Java version:"),
				config.Java.Version,
				label.Sprint("Java home:"),
				config.Java.Home,
			)
			if err != nil {
				return err
			}
			if config.BuildTool.SettingsAlias != "" {
				_, err = fmt.Fprintf(
					writer,
					"%s %s\n",
					label.Sprint("Maven settings:"),
					config.BuildTool.SettingsAlias,
				)
			}
			return err
		},
	}
}

func defaultInspectorFactory() (projectInspector, error) {
	return project.NewDefaultInspector()
}
