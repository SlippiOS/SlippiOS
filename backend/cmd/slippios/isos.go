package main

import (
	"os"
	"path/filepath"
	"strings"

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
			mountError := ""
			if data, err := os.ReadFile(filepath.Join(paths.DataDir, "system", "isos-mount")); err == nil {
				text := strings.TrimSpace(string(data))
				if text != "" && text != "mounted" {
					mountError = text
				}
			}
			return mode.emit(map[string]any{
				"ok":          true,
				"isos":        payload,
				"mounted":     isMountpoint(paths.ISODir),
				"mount_error": mountError,
			}, func() {
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

func isMountpoint(path string) bool {
	data, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 4 && fields[4] == path {
			return true
		}
	}
	return false
}
