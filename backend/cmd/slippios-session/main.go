package main

import (
	"fmt"
	"os"

	"github.com/SlippiOS/SlippiOS/backend/internal/desktop"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

func main() {
	bootAnimation()
	paths := settings.DefaultPaths()

	if paths.CheckHardwareBinding() {
		alert([]string{
			"This cartridge was used on different hardware.",
			"The saved Slippi login has been removed.",
			"Sign in again to play online.",
		})
	}

	_, statErr := os.Stat(paths.SettingsFile)
	haveSettings := statErr == nil
	var values map[string]any
	var loadErr error
	if haveSettings {
		values, loadErr = paths.Load()
	}

	switch {
	case !haveSettings && !anyValidISO(paths):
		values, _ = onboarding(paths, "First-time setup: no settings and no game image.")
	case !haveSettings:
		alert([]string{
			"Settings file is missing.",
			"Expected " + paths.SettingsFile,
			"An existing game image was found, but SlippiOS is not configured.",
		})
		values, _ = onboarding(paths, "Setup: settings file missing.")
	case loadErr != nil:
		alert([]string{"Settings file is invalid.", loadErr.Error()})
		values, _ = onboarding(paths, "Setup: settings file invalid.")
	default:
		game, _ := values["game"].(map[string]any)
		raw, _ := game["iso"].(string)
		if ok, reason := paths.ISOStatus(raw); !ok {
			alert([]string{"Game image problem.", reason})
			values, _ = onboarding(paths, "Setup: game image missing or invalid.")
		}
	}

	if values == nil {
		alert([]string{"Setup was not completed.", "Falling back to a shell."})
		return
	}

	configureNetwork(values)
	_ = desktop.ApplyDisplay(values)
	_ = desktop.ApplyAudio(values)

	if loggedIn, detail := ensureLogin(paths, values); loggedIn {
		fmt.Printf("\n                 Slippi login: %s\n", detail)
	} else {
		fmt.Print("\n                 No Slippi login; online play unavailable.\n")
	}

	appimage, ok := ensureDolphin()
	if !ok {
		return
	}
	launch(paths, values, appimage)
}
