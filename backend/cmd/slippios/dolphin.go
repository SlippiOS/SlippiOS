package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

func dolphinCommand(args []string, mode modes) {
	if len(args) == 0 {
		fail("usage: slippios dolphin config|reset")
	}
	switch args[0] {
	case "config":
		dolphinConfigCommand(mode)
	case "reset":
		config := filepath.Join(settings.DefaultPaths().DolphinDir, "Config")
		for _, name := range []string{"Dolphin.ini", "GFX.ini", "GCPadNew.ini"} {
			_ = os.Remove(filepath.Join(config, name))
		}
		emit(map[string]any{"ok": true}, mode, func() {
			fmt.Println("Dolphin configuration reset.")
		})
	default:
		fail("usage: slippios dolphin config|reset")
	}
}

// dolphinConfigCommand opens Dolphin's own settings window so users can bind
// controllers or change graphics options.
func dolphinConfigCommand(mode modes) {
	paths := settings.DefaultPaths()
	appimage, err := dolphin.DefaultPaths().Ensure(func(message string) {
		fmt.Fprintln(os.Stderr, message)
	}, false)
	if err != nil {
		fail(err.Error())
	}
	values, err := settings.DefaultPaths().Load()
	if err != nil {
		fail(err.Error())
	}
	arguments := dolphinArgs(values)
	arguments = append(arguments, "-C", "Dolphin.Display.Fullscreen=False")
	returncode, err := runDolphin(paths, appimage, arguments)
	if err != nil {
		fail(err.Error())
	}
	emit(map[string]any{"ok": returncode == 0, "returncode": returncode}, mode, func() {
		fmt.Println("Dolphin closed.")
	})
}
