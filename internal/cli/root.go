package cli

import "github.com/spf13/cobra"

func newRootCommand(options Options) *cobra.Command {
	version := formatVersion(options)
	root := &cobra.Command{
		Use:           options.Name,
		Short:         options.Description,
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
	root.SetOut(options.Stdout)
	root.SetErr(options.Stderr)
	if options.Stdin != nil {
		root.SetIn(options.Stdin)
	}
	root.SetVersionTemplate("{{.Version}}\n")

	// Register top-level commands here. Command-specific dependencies should be
	// passed into their constructors instead of accessed through global state.
	root.AddCommand(newInitCommand(defaultInitializerFactory, defaultWorkingDirectory))
	root.AddCommand(newUninitCommand(defaultUninitializerFactory, defaultWorkingDirectory))
	root.AddCommand(newRunCommand(defaultRunnerFactory, defaultWorkingDirectory))
	root.AddCommand(newSettingsCommand(
		defaultMavenSettingsFactory,
		defaultProjectMavenSettingsFactory,
		defaultWorkingDirectory,
	))
	root.AddCommand(newStatusCommand(defaultInspectorFactory, defaultWorkingDirectory))
	root.AddCommand(newUninstallCommand(defaultApplicationUninstaller))
	root.AddCommand(newUpdateCommand(defaultUpdateService(options.Version)))
	root.AddCommand(newVersionCommand(version))

	return root
}
