package main

import (
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/spf13/cobra"
)

func newISOsCommand(mode *modes) *cobra.Command {
	return &cobra.Command{
		Use:   "isos",
		Short: "List Melee images",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			paths := settings.DefaultPaths()
			entries := paths.FindISOs()
			payload := make([]map[string]any, 0, len(entries))
			for _, entry := range entries {
				payload = append(payload, map[string]any{
					"path": entry.Path, "name": entry.Name,
					"valid": entry.Valid, "reason": entry.Reason,
				})
			}
			return mode.emit(map[string]any{"ok": true, "isos": payload}, func() {
				if len(entries) == 0 {
					mode.println("No game images found in /data/isos.")
					return
				}
				for _, entry := range entries {
					state := "ok"
					if !entry.Valid {
						state = entry.Reason
					}
					mode.printf("%s: %s\n", entry.Name, state)
				}
			})
		},
	}
}
