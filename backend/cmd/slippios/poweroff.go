package main

import (
	"os/exec"

	"github.com/spf13/cobra"
)

func newPoweroffCommand(mode *modes) *cobra.Command {
	return &cobra.Command{
		Use:   "poweroff",
		Short: "Power off the console",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPoweroff(mode)
		},
	}
}

func runPoweroff(mode *modes) error {
	if err := exec.Command("systemctl", "poweroff", "--no-block").Run(); err != nil {
		return mode.fail(err.Error())
	}
	return mode.emit(map[string]any{"ok": true}, func() {
		mode.println("Powering off.")
	})
}
