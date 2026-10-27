package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/SlippiOS/SlippiOS/backend/internal/netpages"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

func netCommand(args []string) {
	flags := flag.NewFlagSet("net", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	token := flags.String("token", "", "URL token")
	port := flags.Int("port", 8080, "listen port")
	mode := flags.String("mode", "setup", "setup, iso, or login")
	if err := flags.Parse(args); err != nil {
		fail(err.Error())
	}
	if *token == "" {
		fail("--token is required")
	}
	switch *mode {
	case "setup", "iso", "login":
	default:
		fail("unknown net mode: " + *mode)
	}
	server := netpages.New(*token, *port, settings.DefaultPaths())
	server.Mode = *mode
	url, err := server.Start()
	if err != nil {
		fail(err.Error())
	}
	line, _ := json.Marshal(map[string]any{"ok": true, "url": url})
	fmt.Println(string(line))
	if err := server.Serve(); err != nil {
		fail(err.Error())
	}
}
