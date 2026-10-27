package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

func configureNetwork(values map[string]any) {
	network, _ := values["network"].(map[string]any)
	ssid, _ := network["ssid"].(string)
	ssid = strings.TrimSpace(ssid)
	psk, _ := network["psk"].(string)
	if ssid == "" {
		return
	}
	if _, err := exec.LookPath("nmcli"); err != nil {
		return
	}
	if output, err := exec.Command("nmcli", "-t", "-f", "STATE", "general").Output(); err == nil {
		if strings.HasPrefix(strings.TrimSpace(string(output)), "connected") {
			return
		}
	}

	fmt.Printf("\n                 Connecting to Wi-Fi “%s”…\n", ssid)
	arguments := []string{"--wait", "20", "device", "wifi", "connect", ssid}
	if psk != "" {
		arguments = append(arguments, "password", psk)
	}
	command := exec.Command("nmcli", arguments...)
	_ = command.Run()
}

func onboarding(paths settings.Paths, reason string) (map[string]any, bool) {
	clear()
	fmt.Print("\n                 SLIPPIOS SETUP\n\n")
	fmt.Println("                 Melee 1.02 (NTSC) is required. SlippiOS does not")
	fmt.Print("                 include or download game images.\n\n")
	if reason != "" {
		fmt.Print("                 " + reason + "\n\n")
	}

	iso, ok := chooseISO(paths)
	if !ok {
		return nil, false
	}
	ssid, _ := line("\n                 Wi-Fi SSID (blank to skip): ")
	psk := ""
	if ssid != "" {
		psk, _ = line("                 Wi-Fi password: ")
	}

	values := settings.Defaults()
	game := values["game"].(map[string]any)
	if relative, err := filepath.Rel(paths.ISODir, iso); err == nil &&
		relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		game["iso"] = relative
	} else {
		game["iso"] = iso
	}
	network := values["network"].(map[string]any)
	network["ssid"] = ssid
	network["psk"] = psk
	if err := paths.Save(values); err != nil {
		alert([]string{"Could not save settings.", err.Error()})
		return nil, false
	}
	fmt.Printf("\n                 Wrote %s\n", paths.SettingsFile)
	time.Sleep(time.Second)
	return values, true
}
