package main

import (
	"fmt"

	"github.com/SlippiOS/SlippiOS/backend/internal/desktop"
)

func audioCommand(args []string, mode modes) {
	if len(args) == 0 || args[0] != "sinks" {
		fail("usage: slippios audio sinks")
	}
	sinks, err := desktop.Sinks()
	if err != nil {
		emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
		return
	}
	list := make([]map[string]any, 0, len(sinks))
	for _, sink := range sinks {
		list = append(list, map[string]any{
			"name":        sink.Name,
			"description": sink.Description,
		})
	}
	emit(map[string]any{"ok": true, "sinks": list}, mode, func() {
		for _, sink := range sinks {
			fmt.Println(sink.Name)
		}
	})
}
