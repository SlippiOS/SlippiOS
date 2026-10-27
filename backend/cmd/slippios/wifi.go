package main

import (
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/SlippiOS/SlippiOS/backend/internal/wifi"
	"github.com/spf13/cobra"
)

type wifiConnectOptions struct {
	ssid     string
	password string
}

func newWiFiCommand(mode *modes) *cobra.Command {
	command := &cobra.Command{
		Use:   "wifi",
		Short: "Manage Wi-Fi",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mode.fail("usage: slippios wifi scan|status|connect")
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "scan",
		Short: "List nearby networks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWiFiScan(mode)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show the current connection",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWiFiStatus(mode)
		},
	})
	options := wifiConnectOptions{}
	connect := &cobra.Command{
		Use:   "connect",
		Short: "Join a network",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWiFiConnect(mode, options)
		},
	}
	connect.Flags().StringVar(&options.ssid, "ssid", "", "network name")
	connect.Flags().StringVar(&options.password, "password", "", "network password")
	command.AddCommand(connect)
	return command
}

func runWiFiScan(mode *modes) error {
	networks, err := wifi.Scan()
	if err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	payload := make([]map[string]any, 0, len(networks))
	for _, network := range networks {
		payload = append(payload, map[string]any{
			"ssid": network.SSID, "signal": network.Signal,
			"security": network.Security, "active": network.Active,
		})
	}
	return mode.emit(map[string]any{"ok": true, "networks": payload}, func() {
		for _, network := range networks {
			active := " "
			if network.Active {
				active = "*"
			}
			security := network.Security
			if security == "" {
				security = "open"
			}
			mode.printf("%s %s  %d%%  %s\n", active, network.SSID, network.Signal, security)
		}
	})
}

func runWiFiStatus(mode *modes) error {
	status := wifi.StatusNow()
	return mode.emit(map[string]any{"ok": true, "connected": status.Connected, "ssid": status.SSID}, func() {
		if status.Connected {
			mode.printf("Wi-Fi: connected to %s\n", status.SSID)
		} else {
			mode.println("Wi-Fi: not connected")
		}
	})
}

func runWiFiConnect(mode *modes, options wifiConnectOptions) error {
	status, err := wifi.Connect(settings.DefaultPaths(), options.ssid, options.password)
	if err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	return mode.emit(map[string]any{"ok": true, "ssid": status.SSID}, func() {
		mode.printf("Wi-Fi: connected to %s\n", status.SSID)
	})
}
