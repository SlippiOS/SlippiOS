package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func execute() int {
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		if errors.Is(err, errFailed) {
			return 1
		}
		fmt.Fprintln(root.ErrOrStderr(), "error:", err)
		return 1
	}
	return 0
}

func newRootCommand() *cobra.Command {
	mode := &modes{out: os.Stdout, err: os.Stderr}
	root := &cobra.Command{
		Use:           "slippios",
		Short:         "SlippiOS control CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mode.fail("command required")
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			mode.out = cmd.OutOrStdout()
			mode.err = cmd.ErrOrStderr()
			return nil
		},
	}
	root.PersistentFlags().BoolVar(&mode.json, "json", false, "emit JSON")
	root.PersistentFlags().BoolVar(&mode.human, "human", false, "emit human-readable output")

	root.AddCommand(
		newStatusCommand(mode),
		newSettingsCommand(mode),
		newISOsCommand(mode),
		newLoginCommand(mode),
		newNetCommand(mode),
		newUpdateCommand(mode),
		newOSCommand(mode),
		newPlayCommand(mode),
		newDolphinCommand(mode),
		newControllersCommand(mode),
		newGeckoCommand(mode),
		newReplayCommand(mode),
		newDisplayCommand(mode),
		newAudioCommand(mode),
		newPoweroffCommand(mode),
		newWiFiCommand(mode),
	)
	return root
}
