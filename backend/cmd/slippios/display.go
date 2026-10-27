package main

import (
	"github.com/SlippiOS/SlippiOS/backend/internal/desktop"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/spf13/cobra"
)

func newDisplayCommand(mode *modes) *cobra.Command {
	command := &cobra.Command{
		Use:   "display",
		Short: "Change the display mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mode.fail("usage: slippios display apply|modes|mode|custom")
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "apply",
		Short: "Apply the saved display and audio settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDisplayApply(mode)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "modes",
		Short: "List the modes the display advertises",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDisplayModes(mode)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "mode <WxH> <Hz>",
		Short: "Switch to an advertised mode",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDisplayMode(mode, args[0], args[1])
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "custom <WxH[@Hz]>",
		Short: "Add and switch to a custom mode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDisplayCustom(mode, args[0])
		},
	})
	return command
}

func runDisplayApply(mode *modes) error {
	paths := settings.DefaultPaths()
	values, err := paths.Load()
	if err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	if err := desktop.ApplyDisplay(values); err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	// Audio is best effort: a missing sink must not fail the display change.
	_ = desktop.ApplyAudio(values)
	return mode.emit(map[string]any{"ok": true}, func() {
		mode.println("Display settings applied.")
	})
}

func runDisplayModes(mode *modes) error {
	available, err := desktop.Modes()
	if err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	return mode.emit(map[string]any{
		"ok":        true,
		"output":    available.Output,
		"current":   available.Current,
		"rate":      available.Rate,
		"edid_name": available.EDIDName,
		"modes":     available.Modes,
	}, func() {
		mode.printf("Display: %s (%s)\n", available.Output, available.EDIDName)
		for _, entry := range available.Modes {
			mode.printf("  %s  %v\n", entry.Resolution, entry.Rates)
		}
	})
}

func runDisplayMode(mode *modes, resolution, rate string) error {
	if err := desktop.SetMode(resolution, rate); err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	paths := settings.DefaultPaths()
	values, err := paths.Load()
	if err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	display, _ := values["display"].(map[string]any)
	display["resolution"] = resolution
	display["rate"] = rate
	display["custom"] = ""
	if err := paths.Save(values); err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	return mode.emit(map[string]any{"ok": true, "resolution": resolution, "rate": rate}, func() {
		mode.printf("Display set to %s @ %s Hz.\n", resolution, rate)
	})
}

func runDisplayCustom(mode *modes, spec string) error {
	if err := desktop.ApplyCustomMode(spec); err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	paths := settings.DefaultPaths()
	values, err := paths.Load()
	if err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	display, _ := values["display"].(map[string]any)
	display["resolution"] = "custom"
	display["custom"] = spec
	if err := paths.Save(values); err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	return mode.emit(map[string]any{"ok": true, "custom": spec}, func() {
		mode.println("Custom mode applied:", spec)
	})
}
