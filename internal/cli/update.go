package cli

import (
	"context"
	"fmt"

	"github.com/codeboyzhou/javaup/internal/selfupdate"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type updateService interface {
	Check(context.Context) (selfupdate.Result, error)
	UpdateWithProgress(context.Context, selfupdate.ProgressFunc) (selfupdate.Result, error)
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
					output := command.OutOrStdout()
					message := fmt.Sprintf("Update available: %s -> %s", result.Current, result.Latest)
					_, err = fmt.Fprintln(output, newOutputStyle(output, color.FgYellow).Sprint(message))
					return err
				}
				output := command.OutOrStdout()
				message := fmt.Sprintf("Already up to date (%s)", result.Current)
				_, err = fmt.Fprintln(output, newOutputStyle(output, color.FgGreen).Sprint(message))
				return err
			}

			progress := newUpdateProgressRenderer(command.OutOrStdout())
			result, err := service.UpdateWithProgress(command.Context(), progress.Report)
			if finishErr := progress.Finish(); finishErr != nil {
				return finishErr
			}
			if err != nil {
				return err
			}
			if !result.Updated {
				_, err = fmt.Fprintln(
					command.OutOrStdout(),
					progress.Success(fmt.Sprintf("Already up to date (%s)", result.Current)),
				)
				return err
			}
			_, err = fmt.Fprintln(command.OutOrStdout(), progress.Success("Updated successfully"))
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
