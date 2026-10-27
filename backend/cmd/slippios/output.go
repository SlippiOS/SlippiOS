package main

import (
	"encoding/json"
	"fmt"
	"os"
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
		if ok, _ := payload["ok"].(bool); !ok {
			os.Exit(1)
		}
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
