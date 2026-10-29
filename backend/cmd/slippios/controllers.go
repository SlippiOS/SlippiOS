package main

import (
	"strconv"

	"github.com/SlippiOS/SlippiOS/backend/internal/controllers"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/spf13/cobra"
)

func newControllersCommand(mode *modes) *cobra.Command {
	command := &cobra.Command{
		Use:   "controllers",
		Short: "List or configure the four GameCube controller ports",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runControllersList(mode)
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "Show the port assignments and available profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runControllersList(mode)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "profiles",
		Short: "List the installed controller profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runControllersProfiles(mode)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "set <port> <input> [profile]",
		Short: "Set one port to adapter, pad, keyboard or none",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile := ""
			if len(args) == 3 {
				profile = args[2]
			}
			return runControllersSet(mode, args[0], args[1], profile)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "auto",
		Short: "Assign detected standard pads to empty ports",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runControllersAuto(mode)
		},
	})
	return command
}

func runControllersAuto(mode *modes) error {
	paths := settings.DefaultPaths()
	values, err := paths.Ensure()
	if err != nil {
		return mode.fail(err.Error())
	}
	ports, changed, err := controllers.AutoAssign(values, paths.DolphinDir)
	if err != nil {
		return mode.fail(err.Error())
	}
	if changed {
		if err := paths.Save(values); err != nil {
			return mode.fail(err.Error())
		}
	}
	return mode.emit(map[string]any{
		"ok":      true,
		"changed": changed,
		"ports":   ports,
		"pads":    controllers.DetectedPads(),
	}, func() {
		if changed {
			mode.println("Controller assigned.")
		} else {
			mode.println("No controller changes.")
		}
	})
}

func runControllersList(mode *modes) error {
	paths := settings.DefaultPaths()
	values, err := paths.Ensure()
	if err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	profiles, err := controllers.ListProfiles(paths.DolphinDir)
	if err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	if changed, err := controllers.NormalizeProfiles(values, paths.DolphinDir); err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	} else if changed {
		if err := paths.Save(values); err != nil {
			return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
		}
	}
	ports := controllers.Ports(values)
	return mode.emit(map[string]any{
		"ok":       true,
		"ports":    ports,
		"profiles": profiles,
		"pads":     controllers.DetectedPads(),
	}, func() {
		for _, port := range ports {
			if port.Device == controllers.DevicePad {
				mode.printf("Port %d: %s (%s)\n", port.Port, port.Device, port.Profile)
				continue
			}
			mode.printf("Port %d: %s\n", port.Port, port.Device)
		}
	})
}

func runControllersProfiles(mode *modes) error {
	profiles, err := controllers.ListProfiles(settings.DefaultPaths().DolphinDir)
	if err != nil {
		return mode.emit(map[string]any{"ok": false, "error": err.Error()}, nil)
	}
	return mode.emit(map[string]any{"ok": true, "profiles": profiles}, func() {
		for _, profile := range profiles {
			mode.println(profile.Name)
		}
	})
}

func runControllersSet(mode *modes, rawPort, device, profile string) error {
	port, err := strconv.Atoi(rawPort)
	if err != nil || port < 1 || port > 4 {
		return mode.fail("port must be between 1 and 4")
	}
	if !controllers.ValidDevice(device) {
		return mode.fail("input must be adapter, pad, keyboard or none")
	}
	paths := settings.DefaultPaths()
	values, err := paths.Ensure()
	if err != nil {
		return mode.fail(err.Error())
	}
	if device == controllers.DevicePad {
		if profile == "" {
			profile = controllers.Ports(values)[port-1].Profile
		}
		if profile == "" {
			profiles, _ := controllers.ListProfiles(paths.DolphinDir)
			profile = controllers.MatchProfile(profiles, controllers.DetectedPads())
		}
		if profile == "" {
			profile = controllers.ProfileCustom
		}
		if profile != controllers.ProfileCustom {
			if err := controllers.EnsureDefaults(paths.DolphinDir); err != nil {
				return mode.fail(err.Error())
			}
			profile = controllers.ResolveProfile(paths.DolphinDir, profile)
			if err := controllers.ApplyProfile(paths.DolphinDir, port, profile); err != nil {
				return mode.fail(err.Error())
			}
		}
	}
	if err := controllers.SetPort(values, port, device, profile); err != nil {
		return mode.fail(err.Error())
	}
	if err := paths.Save(values); err != nil {
		return mode.fail(err.Error())
	}
	return mode.emit(map[string]any{
		"ok":    true,
		"ports": controllers.Ports(values),
	}, func() {
		mode.printf("Port %d: %s", port, device)
		if device == controllers.DevicePad && profile != "" {
			mode.printf(" (%s)", profile)
		}
		mode.println("")
	})
}
