package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SlippiOS/SlippiOS/backend/internal/controllers"
	"github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/spf13/cobra"
)

func newDolphinCommand(mode *modes) *cobra.Command {
	command := &cobra.Command{
		Use:   "dolphin",
		Short: "Open or reset Dolphin",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mode.fail("usage: slippios dolphin config|reset")
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "config",
		Short: "Open Dolphin's settings window",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDolphinConfig(mode)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "reset",
		Short: "Reset Dolphin's configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDolphinReset(mode)
		},
	})
	return command
}

func runDolphinReset(mode *modes) error {
	config := filepath.Join(settings.DefaultPaths().DolphinDir, "Config")
	for _, name := range []string{"Dolphin.ini", "GFX.ini", "GCPadNew.ini"} {
		_ = os.Remove(filepath.Join(config, name))
	}
	return mode.emit(map[string]any{"ok": true}, func() {
		mode.println("Dolphin configuration reset.")
	})
}

func runDolphinConfig(mode *modes) error {
	paths := settings.DefaultPaths()
	appimage, err := dolphin.DefaultPaths().Ensure(func(message string) {
		fmt.Fprintln(mode.stderr(), message)
	}, false)
	if err != nil {
		return mode.fail(err.Error())
	}
	values, err := settings.DefaultPaths().Load()
	if err != nil {
		return mode.fail(err.Error())
	}
	if err := controllers.ApplyConfigured(paths.DolphinDir, values); err != nil {
		return mode.fail(err.Error())
	}
	arguments := dolphinArgs(values)
	arguments = append(arguments, "-C", "Dolphin.Display.Fullscreen=False")
	returncode, err := runDolphin(paths, appimage, arguments)
	if err != nil {
		return mode.fail(err.Error())
	}
	return mode.emit(map[string]any{"ok": returncode == 0, "returncode": returncode}, func() {
		mode.println("Dolphin closed.")
	})
}
