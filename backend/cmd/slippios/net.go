package main

import (
	"encoding/json"

	"github.com/SlippiOS/SlippiOS/backend/internal/netpages"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/spf13/cobra"
)

type netOptions struct {
	token string
	port  int
	mode  string
}

func newNetCommand(mode *modes) *cobra.Command {
	options := netOptions{port: 8080, mode: "setup"}
	command := &cobra.Command{
		Use:   "net",
		Short: "Serve the LAN setup pages",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNet(mode, options)
		},
	}
	command.Flags().StringVar(&options.token, "token", "", "URL token")
	command.Flags().IntVar(&options.port, "port", 8080, "listen port")
	command.Flags().StringVar(&options.mode, "mode", "setup", "setup, iso, or login")
	return command
}

func runNet(mode *modes, options netOptions) error {
	if options.token == "" {
		return mode.fail("--token is required")
	}
	switch options.mode {
	case "setup", "iso", "login":
	default:
		return mode.fail("unknown net mode: " + options.mode)
	}
	server := netpages.New(options.token, options.port, settings.DefaultPaths())
	server.Mode = options.mode
	url, err := server.Start()
	if err != nil {
		return mode.fail(err.Error())
	}
	line, _ := json.Marshal(map[string]any{"ok": true, "url": url})
	mode.println(string(line))
	if err := server.Serve(); err != nil {
		return mode.fail(err.Error())
	}
	return nil
}
