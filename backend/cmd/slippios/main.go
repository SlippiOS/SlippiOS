// Command slippios is the Go control-plane spike.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/SlippiOS/SlippiOS/backend/internal/wifi"
)

type modes struct {
	json  bool
	human bool
}

func parseModes(args []string) ([]string, modes) {
	result := make([]string, 0, len(args))
	mode := modes{}
	for _, argument := range args {
		switch argument {
		case "--json":
			mode.json = true
		case "--human":
			mode.human = true
		default:
			result = append(result, argument)
		}
	}
	return result, mode
}

func isTTY() bool {
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func emit(payload map[string]any, mode modes, human func()) {
	jsonMode := mode.json || (!mode.human && !isTTY())
	if jsonMode {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "")
		_ = encoder.Encode(payload)
		return
	}
	if ok, _ := payload["ok"].(bool); !ok {
		message, _ := payload["error"].(string)
		fmt.Fprintln(os.Stderr, "error:", message)
		os.Exit(1)
	}
	human()
}

func fail(message string) {
	emit(map[string]any{"ok": false, "error": message}, modes{}, func() {})
	os.Exit(1)
}

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
	dolphin := dolphinCurrent(paths)
	return map[string]any{
		"ok":       true,
		"settings": summary,
		"login":    map[string]any{"signed_in": login, "detail": loginDetail},
		"dolphin":  map[string]any{"current": dolphin},
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

func dolphinCurrent(paths settings.Paths) string {
	target, err := os.Readlink(filepath.Join(paths.DolphinDir, "current"))
	if err != nil {
		return ""
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(paths.DolphinDir, target)
	}
	if info, err := os.Stat(target); err != nil || info.IsDir() {
		return ""
	}
	return filepath.Base(filepath.Dir(target))
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
		if relative, err := filepath.Rel(paths.ISODir, path); err == nil {
			value = relative
		}
	}
	if name == "refresh_hz" {
		number, err := strconv.Atoi(value)
		if err != nil {
			fail("refresh_hz must be a number")
		}
		valuesSection[name] = int64(number)
	} else {
		valuesSection[name] = value
	}
	if err := paths.Save(values); err != nil {
		fail(err.Error())
	}
	emit(map[string]any{"ok": true, "settings": values}, mode, func() {
		fmt.Println("Saved", key)
	})
}

func isosCommand(mode modes) {
	paths := settings.DefaultPaths()
	entries := paths.FindISOs()
	payload := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		payload = append(payload, map[string]any{
			"path": entry.Path, "name": entry.Name,
			"valid": entry.Valid, "reason": entry.Reason,
		})
	}
	emit(map[string]any{"ok": true, "isos": payload}, mode, func() {
		if len(entries) == 0 {
			fmt.Println("No game images found in /data/isos.")
			return
		}
		for _, entry := range entries {
			state := "ok"
			if !entry.Valid {
				state = entry.Reason
			}
			fmt.Printf("%s: %s\n", entry.Name, state)
		}
	})
}

func wifiCommand(args []string, mode modes) {
	if len(args) == 0 {
		fail("usage: slippios wifi scan|status|connect")
	}
	switch args[0] {
	case "scan":
		networks, err := wifi.Scan()
		if err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		payload := make([]map[string]any, 0, len(networks))
		for _, network := range networks {
			payload = append(payload, map[string]any{
				"ssid": network.SSID, "signal": network.Signal,
				"security": network.Security, "active": network.Active,
			})
		}
		emit(map[string]any{"ok": true, "networks": payload}, mode, func() {
			for _, network := range networks {
				active := " "
				if network.Active {
					active = "*"
				}
				security := network.Security
				if security == "" {
					security = "open"
				}
				fmt.Printf("%s %s  %d%%  %s\n", active, network.SSID, network.Signal, security)
			}
		})
	case "status":
		status := wifi.StatusNow()
		emit(map[string]any{"ok": true, "connected": status.Connected, "ssid": status.SSID}, mode, func() {
			if status.Connected {
				fmt.Printf("Wi-Fi: connected to %s\n", status.SSID)
			} else {
				fmt.Println("Wi-Fi: not connected")
			}
		})
	case "connect":
		flags := flag.NewFlagSet("wifi connect", flag.ExitOnError)
		ssid := flags.String("ssid", "", "network name")
		password := flags.String("password", "", "network password")
		_ = flags.Parse(args[1:])
		status, err := wifi.Connect(settings.DefaultPaths(), *ssid, *password)
		if err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		emit(map[string]any{"ok": true, "ssid": status.SSID}, mode, func() {
			fmt.Printf("Wi-Fi: connected to %s\n", status.SSID)
		})
	default:
		fail("unknown wifi action")
	}
}

func main() {
	args, mode := parseModes(os.Args[1:])
	if len(args) == 0 {
		fail("command required")
	}
	switch args[0] {
	case "status":
		payload := statusPayload()
		emit(payload, mode, func() { printStatus(payload) })
	case "settings":
		settingsCommand(args[1:], mode)
	case "isos":
		isosCommand(mode)
	case "wifi":
		wifiCommand(args[1:], mode)
	default:
		fail("unknown command: " + args[0])
	}
}
