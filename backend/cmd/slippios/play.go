package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SlippiOS/SlippiOS/backend/internal/desktop"
	"github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/spf13/cobra"
)

type playOptions struct {
	requireLogin bool
}

func newPlayCommand(mode *modes) *cobra.Command {
	options := playOptions{}
	command := &cobra.Command{
		Use:   "play",
		Short: "Launch Slippi Dolphin with the selected image",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlay(mode, options)
		},
	}
	command.Flags().BoolVar(&options.requireLogin, "require-login", false, "refuse to start without a saved login")
	return command
}

func runPlay(mode *modes, options playOptions) error {
	paths := settings.DefaultPaths()
	if installed, _ := paths.InstallLogin(settings.Defaults()); !installed && options.requireLogin {
		return mode.fail("not signed in")
	}
	appimage, err := dolphin.DefaultPaths().Ensure(func(message string) {
		fmt.Fprintln(mode.stderr(), message)
	}, false)
	if err != nil {
		return mode.fail(err.Error())
	}

	values, err := paths.Load()
	if err != nil {
		return mode.fail("no game image")
	}
	game, _ := values["game"].(map[string]any)
	raw, _ := game["iso"].(string)
	if raw == "" {
		return mode.fail("no game image")
	}
	valid, reason := paths.ISOStatus(raw)
	if !valid {
		return mode.fail(reason)
	}
	iso := paths.ISOPath(raw)

	width, height := 640, 480
	if w, h, err := desktop.CurrentResolution(); err == nil {
		width, height = w, h
	}
	arguments := dolphinArgs(values)
	arguments = append(arguments,
		"-C", "Dolphin.Display.Fullscreen=True",
		"-C", "Dolphin.Display.BorderlessFullscreen=True",
		"-C", "Dolphin.Display.RenderWindowAutoSize=False",
		"-C", fmt.Sprintf("Dolphin.Display.RenderWindowWidth=%d", width),
		"-C", fmt.Sprintf("Dolphin.Display.RenderWindowHeight=%d", height),
		"-C", "Dolphin.Display.RenderWindowXPos=0",
		"-C", "Dolphin.Display.RenderWindowYPos=0",
		"-b", "-e", iso)
	returncode, err := runDolphin(paths, appimage, arguments)
	if err != nil {
		return mode.fail(err.Error())
	}
	logPath := filepath.Join(paths.ProfileDir, "logs", "dolphin.log")
	return mode.emit(map[string]any{
		"ok": returncode == 0, "iso": iso, "returncode": returncode,
	}, func() {
		if returncode == 0 {
			mode.println("Dolphin exited cleanly.")
		} else {
			mode.printf("Dolphin exited with status %d.\n", returncode)
			mode.println("Log: " + logPath)
		}
	})
}

// Enum values are the integers Dolphin stores in its INIs.
func dolphinArgs(values map[string]any) []string {
	dolphin, _ := values["dolphin"].(map[string]any)
	text := func(key, fallback string) string {
		value, _ := dolphin[key].(string)
		if value == "" {
			return fallback
		}
		return value
	}
	number := func(key string, fallback int64) int64 {
		switch value := dolphin[key].(type) {
		case int64:
			return value
		case int:
			return int64(value)
		}
		return fallback
	}
	flag := func(key string) bool {
		value, _ := dolphin[key].(bool)
		return value
	}
	boolText := func(value bool) string {
		if value {
			return "True"
		}
		return "False"
	}

	arguments := []string{
		"-C", "Dolphin.Core.GFXBackend=" + text("backend", "Vulkan"),
		"-C", "Graphics.Settings.InternalResolution=" + strconv.FormatInt(number("resolution", 3), 10),
		"-C", "Graphics.Settings.AspectRatio=" + strconv.FormatInt(number("aspect", 3), 10),
		"-C", "Graphics.Settings.MSAA=" + strconv.FormatInt(number("msaa", 1), 10),
		"-C", "Graphics.Hardware.VSync=" + boolText(flag("vsync")),
		"-C", "Dolphin.DSP.Backend=" + text("audio_backend", "Cubeb"),
		"-C", "Dolphin.Core.AudioLatency=" + strconv.FormatInt(number("audio_latency", 20), 10),
	}
	switch text("controller", "adapter") {
	case "adapter":
		for port := 0; port < 4; port++ {
			arguments = append(arguments,
				"-C", fmt.Sprintf("Dolphin.Core.SIDevice%d=12", port))
		}
	case "keyboard":
		arguments = append(arguments, "-C", "Dolphin.Core.SIDevice0=7")
	default:
		arguments = append(arguments, "-C", "Dolphin.Core.SIDevice0=6")
	}
	return arguments
}

func runDolphin(paths settings.Paths, appimage string, arguments []string) (int, error) {
	environment := os.Environ()
	runtime := fmt.Sprintf("/run/user/%d", os.Getuid())
	if info, err := os.Stat(runtime); err == nil && info.IsDir() {
		hasRuntime := false
		for _, value := range environment {
			if strings.HasPrefix(value, "XDG_RUNTIME_DIR=") {
				hasRuntime = true
				break
			}
		}
		if !hasRuntime {
			environment = append(environment, "XDG_RUNTIME_DIR="+runtime)
		}
	}

	args := append([]string{appimage, "-u", paths.DolphinDir}, arguments...)
	command := exec.Command("slippi-appimage-run", args...)
	command.Env = environment

	logDir := filepath.Join(paths.ProfileDir, "logs")
	logPath := filepath.Join(logDir, "dolphin.log")
	var logFile *os.File
	if err := os.MkdirAll(logDir, 0o755); err == nil {
		if file, err := os.Create(logPath); err == nil {
			logFile = file
			defer logFile.Close()
		}
	}
	if logFile != nil {
		command.Stdout = logFile
		command.Stderr = logFile
	} else {
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
	}

	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return exitError.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}
