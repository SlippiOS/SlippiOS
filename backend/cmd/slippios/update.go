package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
)

func updateCommand(args []string, mode modes) {
	flags := flag.NewFlagSet("update", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	check := flags.Bool("check", false, "only check for a new release")
	if err := flags.Parse(args); err != nil {
		fail(err.Error())
	}
	paths := dolphin.DefaultPaths()
	if *check {
		tag, _, err := dolphin.LatestRelease()
		if err != nil {
			fail(err.Error())
		}
		var current any
		if value, ok := paths.CurrentTag(); ok {
			current = value
		}
		payload := map[string]any{
			"ok": true, "check": true, "current": current, "latest": tag,
		}
		emit(payload, mode, func() {
			shown := "not installed"
			if current != nil {
				shown = fmt.Sprint(current)
			}
			fmt.Printf("Current:   %s\nLatest:    %s\n", shown, tag)
		})
		return
	}

	path, err := paths.Ensure(func(message string) {
		fmt.Fprintln(os.Stderr, message)
	}, false)
	if err != nil {
		fail(err.Error())
	}
	current, _ := paths.CurrentTag()
	emit(map[string]any{
		"ok": true, "check": false, "current": current, "path": path,
	}, mode, func() {
		fmt.Printf("Dolphin:   %s\nPath:      %s\n", current, path)
	})
}
