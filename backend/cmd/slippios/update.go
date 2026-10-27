package main

import (
	"fmt"

	"github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	check bool
}

func newUpdateCommand(mode *modes) *cobra.Command {
	options := updateOptions{}
	command := &cobra.Command{
		Use:   "update",
		Short: "Install or check for a Dolphin update",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(mode, options)
		},
	}
	command.Flags().BoolVar(&options.check, "check", false, "only check for a new release")
	return command
}

func runUpdate(mode *modes, options updateOptions) error {
	paths := dolphin.DefaultPaths()
	if options.check {
		tag, _, err := dolphin.LatestRelease()
		if err != nil {
			return mode.fail(err.Error())
		}
		var current any
		if value, ok := paths.CurrentTag(); ok {
			current = value
		}
		payload := map[string]any{
			"ok": true, "check": true, "current": current, "latest": tag,
		}
		return mode.emit(payload, func() {
			shown := "not installed"
			if current != nil {
				shown = fmt.Sprint(current)
			}
			mode.printf("Current:   %s\nLatest:    %s\n", shown, tag)
		})
	}

	path, err := paths.Ensure(func(message string) {
		fmt.Fprintln(mode.stderr(), message)
	}, false)
	if err != nil {
		return mode.fail(err.Error())
	}
	current, _ := paths.CurrentTag()
	return mode.emit(map[string]any{
		"ok": true, "check": false, "current": current, "path": path,
	}, func() {
		mode.printf("Dolphin:   %s\nPath:      %s\n", current, path)
	})
}
