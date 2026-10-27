package main

import (
	"fmt"

	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

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
