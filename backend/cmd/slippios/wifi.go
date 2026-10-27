package main

import (
	"flag"
	"fmt"

	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/SlippiOS/SlippiOS/backend/internal/wifi"
)

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
