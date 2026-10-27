package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

func statusPayload() map[string]any {
	paths := settings.DefaultPaths()
	summary := map[string]any{"exists": false, "path": paths.SettingsFile}
	values, loadErr := paths.Load()
	if loadErr == nil {
		summary["exists"] = true
		raw, _ := values["game"].(map[string]any)["iso"].(string)
		valid, reason := paths.ISOStatus(raw)
		summary["iso"] = raw
		summary["iso_path"] = paths.ISOPath(raw)
		summary["iso_valid"] = valid
		summary["iso_reason"] = reason
		if display, ok := values["display"].(map[string]any); ok {
			summary["refresh_hz"] = display["refresh_hz"]
			summary["resolution"] = display["resolution"]
		}
	} else if _, statErr := os.Stat(paths.SettingsFile); statErr == nil {
		summary["error"] = loadErr.Error()
	}

	login, loginDetail := paths.LoginStatus(paths.DolphinUserFile)
	adapter := adapterPresent()
	currentDolphin := dolphinCurrent()
	return map[string]any{
		"ok":       true,
		"settings": summary,
		"login":    map[string]any{"signed_in": login, "detail": loginDetail},
		"dolphin":  map[string]any{"current": currentDolphin},
		"adapter":  map[string]any{"present": adapter},
	}
}

func adapterPresent() bool {
	entries, err := filepath.Glob("/sys/bus/usb/devices/*")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		vendor, err := os.ReadFile(filepath.Join(entry, "idVendor"))
		if err != nil {
			continue
		}
		product, err := os.ReadFile(filepath.Join(entry, "idProduct"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(vendor)) == "057e" &&
			strings.TrimSpace(string(product)) == "0337" {
			return true
		}
	}
	return false
}

func dolphinCurrent() string {
	if tag, ok := dolphin.DefaultPaths().CurrentTag(); ok {
		return tag
	}
	return ""
}

func printStatus(payload map[string]any) {
	summary := payload["settings"].(map[string]any)
	state := "missing"
	if summary["exists"] == true {
		state = "ok"
	}
	fmt.Printf("Settings:  %s (%s)\n", summary["path"], state)
	if summary["iso"] != nil {
		gameState := "ok"
		if summary["iso_valid"] != true {
			gameState, _ = summary["iso_reason"].(string)
		}
		fmt.Printf("Game:      %s (%s)\n", summary["iso_path"], gameState)
		if summary["refresh_hz"] != nil {
			fmt.Printf("Display:   %v Hz\n", summary["refresh_hz"])
		}
	} else {
		fmt.Println("Game:      not configured")
	}
	login := payload["login"].(map[string]any)
	if login["signed_in"] == true {
		fmt.Printf("Login:     signed in (%s)\n", login["detail"])
	} else {
		fmt.Println("Login:     not signed in")
	}
	dolphin := payload["dolphin"].(map[string]any)
	current, _ := dolphin["current"].(string)
	if current == "" {
		current = "not installed"
	}
	fmt.Printf("Dolphin:   %s\n", current)
	adapter := payload["adapter"].(map[string]any)
	if adapter["present"] == true {
		fmt.Println("Adapter:   connected")
	} else {
		fmt.Println("Adapter:   not connected")
	}
}

func statusCommand(mode modes) {
	payload := statusPayload()
	emit(payload, mode, func() { printStatus(payload) })
}
