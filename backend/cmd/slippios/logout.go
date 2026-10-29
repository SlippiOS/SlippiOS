package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/spf13/cobra"
)

func newLogoutCommand(mode *modes) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Sign out of Slippi",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogout(mode)
		},
	}
}

func runLogout(mode *modes) error {
	paths := settings.DefaultPaths()
	values, err := paths.Load()
	if err != nil {
		values = settings.Defaults()
	}
	configured := settings.DefaultUserJSON
	if slippi, ok := values["slippi"].(map[string]any); ok {
		if value, ok := slippi["user_json"].(string); ok && value != "" {
			configured = value
		}
	}
	// Remove every place a login can be installed: the profile copy, the
	// Dolphin copy shared by stable and mainline, and the legacy locations.
	targets := []string{
		paths.Resolve(configured),
		filepath.Join(paths.ProfileDir, "user.json"),
		filepath.Join(paths.DataDir, "user.json"),
		paths.DolphinUserFile,
		filepath.Join(paths.DolphinStableDir, "Slippi", "user.json"),
	}
	removed := 0
	for _, target := range targets {
		if err := os.Remove(target); err == nil {
			removed++
		} else if !errors.Is(err, os.ErrNotExist) {
			return mode.fail(fmt.Sprintf("cannot remove %s: %v", target, err))
		}
	}
	return mode.emit(map[string]any{
		"ok": true, "removed": removed, "signed_in": false,
	}, func() {
		mode.println("Signed out.")
	})
}
