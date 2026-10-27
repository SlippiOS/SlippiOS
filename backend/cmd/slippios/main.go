package main

import "os"

func main() {
	args, mode := parseModes(os.Args[1:])
	if len(args) == 0 {
		fail("command required")
	}
	switch args[0] {
	case "status":
		statusCommand(mode)
	case "settings":
		settingsCommand(args[1:], mode)
	case "isos":
		isosCommand(mode)
	case "login":
		loginCommand(args[1:], mode)
	case "net":
		netCommand(args[1:])
	case "update":
		updateCommand(args[1:], mode)
	case "os":
		osCommand(args[1:], mode)
	case "play":
		playCommand(args[1:], mode)
	case "dolphin":
		dolphinCommand(args[1:], mode)
	case "display":
		displayCommand(args[1:], mode)
	case "audio":
		audioCommand(args[1:], mode)
	case "poweroff":
		poweroffCommand(mode)
	case "wifi":
		wifiCommand(args[1:], mode)
	default:
		fail("unknown command: " + args[0])
	}
}
