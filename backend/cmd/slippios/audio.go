package main

import (
	"strconv"

	"github.com/SlippiOS/SlippiOS/backend/internal/desktop"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/spf13/cobra"
)

func newAudioCommand(mode *modes) *cobra.Command {
	command := &cobra.Command{
		Use:   "audio",
		Short: "List or apply audio outputs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mode.fail("usage: slippios audio sinks|apply|volume")
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
	command.AddCommand(&cobra.Command{
		Use:   "apply",
		Short: "Apply the saved audio output and volume",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAudioApply(mode)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "volume [0-100]",
		Short: "Show or set the output volume",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAudioVolume(mode, args)
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

func runAudioApply(mode *modes) error {
	values, err := settings.DefaultPaths().Load()
	if err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	if err := desktop.ApplyAudio(values); err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	return mode.emit(map[string]any{"ok": true}, func() {
		mode.println("Audio settings applied.")
	})
}

func runAudioVolume(mode *modes, args []string) error {
	paths := settings.DefaultPaths()
	values, err := paths.Load()
	if err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	audio, _ := values["audio"].(map[string]any)
	if audio == nil {
		audio = map[string]any{}
		values["audio"] = audio
	}
	if len(args) == 0 {
		volume := audioVolume(audio)
		return mode.emit(map[string]any{"ok": true, "volume": volume}, func() {
			mode.printf("Volume: %d%%\n", volume)
		})
	}
	volume, err := strconv.Atoi(args[0])
	if err != nil || volume < 0 || volume > 100 {
		return mode.fail("volume must be 0-100")
	}
	audio["volume"] = int64(volume)
	if err := paths.Save(values); err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	if err := desktop.ApplyAudio(values); err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	return mode.emit(map[string]any{"ok": true, "volume": volume}, func() {
		mode.printf("Volume: %d%%\n", volume)
	})
}

func audioVolume(audio map[string]any) int {
	switch value := audio["volume"].(type) {
	case int64:
		return int(value)
	case int:
		return value
	case float64:
		return int(value)
	}
	return 100
}
