package main

import (
	"fmt"

	"github.com/SlippiOS/SlippiOS/backend/internal/osupdate"
	"github.com/spf13/cobra"
)

func newOSCommand(mode *modes) *cobra.Command {
	command := &cobra.Command{
		Use:   "os",
		Short: "Check, install, or reboot the system",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mode.fail("os action required")
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Check for a system update",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOSCheck(mode)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show update status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOSStatus(mode)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "update",
		Short: "Stage the newest system update",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOSUpdate(mode)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "reboot",
		Short: "Restart the console",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOSReboot(mode)
		},
	})
	return command
}

func runOSCheck(mode *modes) error {
	paths := osupdate.DefaultPaths()
	checked, err := paths.Check()
	if err != nil {
		return mode.fail(err.Error())
	}
	return mode.emit(map[string]any{
		"ok": true, "current": checked.Current, "latest": checked.Latest,
		"update_available": checked.Available,
	}, func() {
		mode.printf("Current:   %s\nLatest:    %s\n", checked.Current, checked.Latest)
		if checked.Available {
			mode.println("Update:    available")
		} else {
			mode.println("Update:    up to date")
		}
	})
}

func runOSStatus(mode *modes) error {
	status := osupdate.DefaultPaths().Status()
	return mode.emit(status, func() {
		mode.printf("Current:   %v\nLatest:    %v\n", status["current"], status["latest"])
		mode.printf("Phase:     %v\n", status["phase"])
		if status["error"] != nil {
			mode.printf("Error:     %v\n", status["error"])
		}
	})
}

func runOSUpdate(mode *modes) error {
	result, err := osupdate.DefaultPaths().Stage(func(message string) {
		fmt.Fprintln(mode.stderr(), message)
	})
	if err != nil {
		return mode.fail(err.Error())
	}
	return mode.emit(result, func() {
		if result["updated"] == true {
			mode.printf("Staged SlippiOS %v. Restart to install.\n", result["latest"])
		} else {
			mode.printf("SlippiOS %v is up to date.\n", result["current"])
		}
	})
}

func runOSReboot(mode *modes) error {
	if err := osupdate.DefaultPaths().RequestReboot(); err != nil {
		return mode.fail(err.Error())
	}
	return mode.emit(map[string]any{"ok": true}, func() {
		mode.println("Restarting…")
	})
}
