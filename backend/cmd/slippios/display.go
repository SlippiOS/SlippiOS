package main

import (
	"fmt"

	"github.com/SlippiOS/SlippiOS/backend/internal/desktop"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

func displayCommand(args []string, mode modes) {
	if len(args) == 0 {
		fail("usage: slippios display apply|modes|custom <WxH[@Hz]>")
	}
	paths := settings.DefaultPaths()
	switch args[0] {
	case "apply":
		values, err := paths.Load()
		if err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		if err := desktop.ApplyDisplay(values); err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		// Audio is best effort: a missing sink must not fail the display change.
		_ = desktop.ApplyAudio(values)
		emit(map[string]any{"ok": true}, mode, func() {
			fmt.Println("Display settings applied.")
		})
	case "modes":
		modes, err := desktop.Modes()
		if err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		emit(map[string]any{
			"ok":        true,
			"output":    modes.Output,
			"current":   modes.Current,
			"rate":      modes.Rate,
			"edid_name": modes.EDIDName,
			"modes":     modes.Modes,
		}, mode, func() {
			fmt.Printf("Display: %s (%s)\n", modes.Output, modes.EDIDName)
			for _, entry := range modes.Modes {
				fmt.Printf("  %s  %v\n", entry.Resolution, entry.Rates)
			}
		})
	case "mode":
		if len(args) != 3 {
			fail("usage: slippios display mode <WxH> <Hz>")
		}
		if err := desktop.SetMode(args[1], args[2]); err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		values, err := paths.Load()
		if err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		display, _ := values["display"].(map[string]any)
		display["resolution"] = args[1]
		display["rate"] = args[2]
		display["custom"] = ""
		if err := paths.Save(values); err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		emit(map[string]any{"ok": true, "resolution": args[1], "rate": args[2]}, mode, func() {
			fmt.Printf("Display set to %s @ %s Hz.\n", args[1], args[2])
		})
	case "custom":
		if len(args) != 2 {
			fail("usage: slippios display custom <WxH[@Hz]>")
		}
		if err := desktop.ApplyCustomMode(args[1]); err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		values, err := paths.Load()
		if err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		display, _ := values["display"].(map[string]any)
		display["resolution"] = "custom"
		display["custom"] = args[1]
		if err := paths.Save(values); err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		emit(map[string]any{"ok": true, "custom": args[1]}, mode, func() {
			fmt.Println("Custom mode applied:", args[1])
		})
	default:
		fail("usage: slippios display apply|modes|custom <WxH[@Hz]>")
	}
}
