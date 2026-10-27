package main

import (
	"fmt"
	"os"

	"github.com/SlippiOS/SlippiOS/backend/internal/osupdate"
)

func osCommand(args []string, mode modes) {
	if len(args) == 0 {
		fail("os action required")
	}
	paths := osupdate.DefaultPaths()
	switch args[0] {
	case "check":
		checked, err := paths.Check()
		if err != nil {
			fail(err.Error())
		}
		emit(map[string]any{
			"ok": true, "current": checked.Current, "latest": checked.Latest,
			"update_available": checked.Available,
		}, mode, func() {
			fmt.Printf("Current:   %s\nLatest:    %s\n", checked.Current, checked.Latest)
			if checked.Available {
				fmt.Println("Update:    available")
			} else {
				fmt.Println("Update:    up to date")
			}
		})
	case "status":
		status := paths.Status()
		emit(status, mode, func() {
			fmt.Printf("Current:   %v\nLatest:    %v\n", status["current"], status["latest"])
			fmt.Printf("Phase:     %v\n", status["phase"])
			if status["error"] != nil {
				fmt.Printf("Error:     %v\n", status["error"])
			}
		})
	case "update":
		result, err := paths.Stage(func(message string) {
			fmt.Fprintln(os.Stderr, message)
		})
		if err != nil {
			fail(err.Error())
		}
		emit(result, mode, func() {
			if result["updated"] == true {
				fmt.Printf("Staged SlippiOS %v. Restart to install.\n", result["latest"])
			} else {
				fmt.Printf("SlippiOS %v is up to date.\n", result["current"])
			}
		})
	case "reboot":
		if err := paths.RequestReboot(); err != nil {
			fail(err.Error())
		}
		emit(map[string]any{"ok": true}, mode, func() {
			fmt.Println("Restarting…")
		})
	default:
		fail("unknown os action")
	}
}
