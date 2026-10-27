package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/spf13/cobra"
)

func newSettingsCommand(mode *modes) *cobra.Command {
	command := &cobra.Command{
		Use:   "settings",
		Short: "Show or change settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSettingsShow(mode)
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSettingsShow(mode)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "set <section.key> <value>",
		Short: "Set one setting",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSettingsSet(mode, args[0], args[1])
		},
	})
	return command
}

func runSettingsShow(mode *modes) error {
	paths := settings.DefaultPaths()
	values, err := paths.Load()
	if err != nil {
		if _, statErr := os.Stat(paths.SettingsFile); statErr != nil {
			values = settings.Defaults()
			err = nil
		}
	}
	if err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	return mode.emit(map[string]any{"ok": true, "settings": values}, func() {
		mode.printf("Game:      %v\n", values["game"].(map[string]any)["iso"])
		display := values["display"].(map[string]any)
		mode.printf("Display:   %v Hz, %v\n", display["refresh_hz"], display["resolution"])
		mode.printf("Audio:     %v\n", values["audio"].(map[string]any)["sink"])
		mode.printf("Network:   %v\n", values["network"].(map[string]any)["ssid"])
	})
}

func runSettingsSet(mode *modes, key, value string) error {
	paths := settings.DefaultPaths()
	section, name, ok := strings.Cut(key, ".")
	if !ok {
		return mode.fail("setting must be section.key")
	}
	values, err := paths.Load()
	if err != nil {
		if _, statErr := os.Stat(paths.SettingsFile); statErr == nil {
			return mode.fail(err.Error())
		}
		values = settings.Defaults()
	}
	valuesSection, ok := values[section].(map[string]any)
	if !ok {
		return mode.fail("unknown setting " + key)
	}
	if name == "iso" {
		path := paths.ISOPath(value)
		if relative, err := filepath.Rel(paths.ISODir, path); err == nil &&
			relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			value = relative
		} else {
			value = path
		}
	}
	if section == "display" && name == "resolution" {
		valuesSection[name] = value
		valuesSection["custom"] = ""
	} else if section == "display" && name == "refresh_hz" && value == "auto" {
		valuesSection[name] = int64(0)
	} else {
		// Use the schema type, not the current value: older files may have
		// stored a number as a string, and should still heal.
		schema := settings.Defaults()
		schemaSection, _ := schema[section].(map[string]any)
		kind := valuesSection[name]
		if schemaSection != nil {
			if expected, ok := schemaSection[name]; ok {
				kind = expected
			}
		}
		switch kind.(type) {
		case bool:
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return mode.fail(key + " must be true or false")
			}
			valuesSection[name] = parsed
		case int64:
			number, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return mode.fail(key + " must be a number")
			}
			valuesSection[name] = number
		default:
			valuesSection[name] = value
		}
	}
	if err := paths.Save(values); err != nil {
		return mode.fail(err.Error())
	}
	return mode.emit(map[string]any{"ok": true, "settings": values}, func() {
		mode.println("Saved", key)
	})
}
