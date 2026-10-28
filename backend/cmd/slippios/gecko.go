package main

import (
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/SlippiOS/SlippiOS/backend/internal/gecko"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/spf13/cobra"
)

func newGeckoCommand(mode *modes) *cobra.Command {
	command := &cobra.Command{
		Use:   "gecko",
		Short: "List or change Melee Gecko codes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGeckoList(mode)
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List Gecko codes and their enabled state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGeckoList(mode)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "set <name> <on|off>",
		Short: "Enable or disable one Gecko code",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, err := strconv.ParseBool(args[1])
			if err != nil {
				switch strings.ToLower(args[1]) {
				case "on":
					enabled = true
				case "off":
					enabled = false
				default:
					return mode.fail("state must be on or off")
				}
			}
			return runGeckoSet(mode, args[0], enabled)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "add <file>",
		Short: "Add custom Gecko codes from a file (use - for stdin)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGeckoAdd(mode, args[0])
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a custom Gecko code",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGeckoRemove(mode, args[0])
		},
	})
	return command
}

func runGeckoList(mode *modes) error {
	paths := settings.DefaultPaths()
	codes, err := gecko.Load(paths.ProfileDir)
	if err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	return mode.emit(map[string]any{"ok": true, "codes": codes}, func() {
		for _, code := range codes {
			state := "off"
			if code.Enabled {
				state = "on"
			}
			mode.printf("%-3s %s\n", state, code.Name)
		}
	})
}

func runGeckoSet(mode *modes, name string, enabled bool) error {
	paths := settings.DefaultPaths()
	if err := gecko.SetEnabled(paths.ProfileDir, name, enabled); err != nil {
		return mode.fail(err.Error())
	}
	codes, err := gecko.Load(paths.ProfileDir)
	if err != nil {
		return mode.fail(err.Error())
	}
	return mode.emit(map[string]any{"ok": true, "codes": codes}, func() {
		state := "disabled"
		if enabled {
			state = "enabled"
		}
		mode.printf("%s: %s\n", name, state)
	})
}

func runGeckoAdd(mode *modes, path string) error {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return mode.fail(err.Error())
	}
	paths := settings.DefaultPaths()
	added, err := gecko.Add(paths.ProfileDir, string(data))
	if err != nil {
		return mode.fail(err.Error())
	}
	return mode.emit(map[string]any{"ok": true, "added": added}, func() {
		for _, code := range added {
			mode.println("Added", code.Name)
		}
	})
}

func runGeckoRemove(mode *modes, name string) error {
	paths := settings.DefaultPaths()
	if err := gecko.Remove(paths.ProfileDir, name); err != nil {
		return mode.fail(err.Error())
	}
	return mode.emit(map[string]any{"ok": true}, func() {
		mode.println("Removed", name)
	})
}
