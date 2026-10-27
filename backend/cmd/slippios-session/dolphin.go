package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/SlippiOS/SlippiOS/backend/internal/desktop"
	"github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

func ensureDolphin() (string, bool) {
	clear()
	fmt.Print("\n\n                 Checking for Slippi Dolphin updates…\n\n")
	appimage, err := dolphin.DefaultPaths().Ensure(func(message string) {
		fmt.Println("                 " + message)
	}, false)
	if err != nil {
		alert([]string{"Could not prepare Slippi Dolphin.", err.Error()})
		return "", false
	}
	return appimage, true
}

func environmentWithRuntime() []string {
	environment := os.Environ()
	runtime := fmt.Sprintf("/run/user/%d", os.Getuid())
	if info, err := os.Stat(runtime); err == nil && info.IsDir() {
		for _, value := range environment {
			if strings.HasPrefix(value, "XDG_RUNTIME_DIR=") {
				return environment
			}
		}
		environment = append(environment, "XDG_RUNTIME_DIR="+runtime)
	}
	return environment
}

func launch(paths settings.Paths, values map[string]any, appimage string) {
	game, _ := values["game"].(map[string]any)
	raw, _ := game["iso"].(string)
	iso := paths.ISOPath(raw)
	clear()
	fmt.Print("\n\n                 Launching Slippi Dolphin…\n\n")
	fmt.Println("                 " + iso)

	logPath := filepath.Join(paths.ProfileDir, "logs", "dolphin.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		alert([]string{"Could not create the Dolphin log.", err.Error()})
		return
	}
	logFile, err := os.Create(logPath)
	if err != nil {
		alert([]string{"Could not create the Dolphin log.", err.Error()})
		return
	}
	defer logFile.Close()

	width, height := 640, 480
	if w, h, err := desktop.CurrentResolution(); err == nil {
		width, height = w, h
	}
	command := exec.Command(
		"slippi-appimage-run", appimage,
		"-u", paths.DolphinDir,
		"-C", "Dolphin.Display.Fullscreen=True",
		"-C", "Dolphin.Display.BorderlessFullscreen=True",
		"-C", "Dolphin.Display.RenderWindowAutoSize=False",
		"-C", fmt.Sprintf("Dolphin.Display.RenderWindowWidth=%d", width),
		"-C", fmt.Sprintf("Dolphin.Display.RenderWindowHeight=%d", height),
		"-C", "Dolphin.Display.RenderWindowXPos=0",
		"-C", "Dolphin.Display.RenderWindowYPos=0",
		"-C", "Graphics.Hardware.VSync=False",
		"-b", "-e", iso,
	)
	command.Env = environmentWithRuntime()
	command.Stdout = logFile
	command.Stderr = logFile
	runErr := command.Run()
	if errors.Is(runErr, exec.ErrNotFound) {
		alert([]string{"slippi-appimage-run is not installed in this image."})
		return
	}
	if runErr == nil {
		return
	}
	var exitError *exec.ExitError
	code := -1
	if errors.As(runErr, &exitError) {
		code = exitError.ExitCode()
	}
	tail := []string{}
	if data, err := os.ReadFile(logPath); err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) > 4 {
			lines = lines[len(lines)-4:]
		}
		tail = lines
	}
	alert(append([]string{
		fmt.Sprintf("Dolphin exited with status %d.", code),
		"Log: " + logPath,
	}, tail...))
}
