package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	check    bool
	progress string
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
	command.Flags().StringVar(&options.progress, "progress", "", "write download progress as JSON to this file")
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

	report := func(value map[string]any) {}
	if options.progress != "" {
		report = func(value map[string]any) { writeProgress(options.progress, value) }
		report(map[string]any{"phase": "checking"})
	}

	path, err := paths.EnsureWithProgress(func(message string) {
		fmt.Fprintln(mode.stderr(), message)
	}, func(done, total int64) {
		report(map[string]any{"phase": "downloading", "done": done, "total": total})
	}, false)
	if err != nil {
		report(map[string]any{"phase": "error", "error": err.Error()})
		return mode.fail(err.Error())
	}
	report(map[string]any{"phase": "ready"})

	current, _ := paths.CurrentTag()
	return mode.emit(map[string]any{
		"ok": true, "check": false, "current": current, "path": path,
	}, func() {
		mode.printf("Dolphin:   %s\nPath:      %s\n", current, path)
	})
}

func writeProgress(path string, value map[string]any) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(temporary, path)
}
