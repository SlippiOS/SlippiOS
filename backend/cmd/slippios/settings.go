package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

func settingsCommand(args []string, mode modes) {
	paths := settings.DefaultPaths()
	if len(args) == 0 || args[0] == "show" {
		values, err := paths.Load()
		if err != nil {
			if _, statErr := os.Stat(paths.SettingsFile); statErr != nil {
				values = settings.Defaults()
				err = nil
			}
		}
		if err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		emit(map[string]any{"ok": true, "settings": values}, mode, func() {
			fmt.Printf("Game:      %v\n", values["game"].(map[string]any)["iso"])
			display := values["display"].(map[string]any)
			fmt.Printf("Display:   %v Hz, %v\n", display["refresh_hz"], display["resolution"])
			fmt.Printf("Audio:     %v\n", values["audio"].(map[string]any)["sink"])
			fmt.Printf("Network:   %v\n", values["network"].(map[string]any)["ssid"])
		})
		return
	}
	if args[0] != "set" || len(args) < 3 {
		fail("usage: slippios settings set section.key value")
	}
	key, value := args[1], args[2]
	section, name, ok := strings.Cut(key, ".")
	if !ok {
		fail("setting must be section.key")
	}
	values, err := paths.Load()
	if err != nil {
		if _, statErr := os.Stat(paths.SettingsFile); statErr == nil {
			fail(err.Error())
		}
		values = settings.Defaults()
	}
	valuesSection, ok := values[section].(map[string]any)
	if !ok {
		fail("unknown setting " + key)
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
				fail(key + " must be true or false")
			}
			valuesSection[name] = parsed
		case int64:
			number, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				fail(key + " must be a number")
			}
			valuesSection[name] = number
		default:
			valuesSection[name] = value
		}
	}
	if err := paths.Save(values); err != nil {
		fail(err.Error())
	}
	emit(map[string]any{"ok": true, "settings": values}, mode, func() {
		fmt.Println("Saved", key)
	})
}
