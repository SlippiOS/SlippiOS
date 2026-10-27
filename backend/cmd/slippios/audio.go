package main

import (
	"github.com/SlippiOS/SlippiOS/backend/internal/desktop"
	"github.com/spf13/cobra"
)

func newAudioCommand(mode *modes) *cobra.Command {
	command := &cobra.Command{
		Use:   "audio",
		Short: "List audio outputs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mode.fail("usage: slippios audio sinks")
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "sinks",
		Short: "List PulseAudio outputs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAudioSinks(mode)
		},
	})
	return command
}

func runAudioSinks(mode *modes) error {
	sinks, err := desktop.Sinks()
	if err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	list := make([]map[string]any, 0, len(sinks))
	for _, sink := range sinks {
		list = append(list, map[string]any{
			"name":        sink.Name,
			"description": sink.Description,
		})
	}
	return mode.emit(map[string]any{"ok": true, "sinks": list}, func() {
		for _, sink := range sinks {
			mode.println(sink.Name)
		}
	})
}
