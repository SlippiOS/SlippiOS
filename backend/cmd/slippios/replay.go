package main

import (
	"path/filepath"
	"strings"

	"github.com/SlippiOS/SlippiOS/backend/internal/replay"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/spf13/cobra"
)

func newReplayCommand(mode *modes) *cobra.Command {
	command := &cobra.Command{
		Use:   "replay",
		Short: "List or play Slippi replays",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReplayList(mode)
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List .slp replays",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReplayList(mode)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "play <file>",
		Short: "Open a replay in the Slippi playback build",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReplayPlay(mode, args[0])
		},
	})
	return command
}

func runReplayList(mode *modes) error {
	paths := settings.DefaultPaths()
	replays, err := replay.List(paths.ProfileDir)
	if err != nil {
		return mode.fail(err.Error())
	}
	return mode.emit(map[string]any{"ok": true, "replays": replays}, func() {
		for _, item := range replays {
			mode.println(item.Name)
		}
	})
}

func runReplayPlay(mode *modes, path string) error {
	if !strings.EqualFold(filepath.Ext(path), ".slp") {
		return mode.fail("replay must be a .slp file")
	}
	paths := settings.DefaultPaths()
	values, err := paths.Load()
	if err != nil {
		return mode.fail(err.Error())
	}
	game, _ := values["game"].(map[string]any)
	raw, _ := game["iso"].(string)
	if raw == "" {
		return mode.fail("no game image")
	}
	valid, reason := paths.ISOStatus(raw)
	if !valid {
		return mode.fail(reason)
	}
	returncode, err := replay.Play(paths.DataDir, paths.ProfileDir, paths.ISOPath(raw), path, values)
	if err != nil {
		return mode.fail(err.Error())
	}
	return mode.emit(map[string]any{"ok": returncode == 0, "returncode": returncode}, func() {
		if returncode == 0 {
			mode.println("Playback closed.")
		} else {
			mode.printf("Playback exited with status %d.\n", returncode)
		}
	})
}
