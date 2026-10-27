package main

import (
	"fmt"
	"os/exec"
)

func poweroffCommand(mode modes) {
	if err := exec.Command("systemctl", "poweroff", "--no-block").Run(); err != nil {
		fail(err.Error())
	}
	emit(map[string]any{"ok": true}, mode, func() {
		fmt.Println("Powering off.")
	})
}
