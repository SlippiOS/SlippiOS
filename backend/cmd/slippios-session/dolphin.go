package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/SlippiOS/SlippiOS/backend/internal/desktop"
	"github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
	"github.com/SlippiOS/SlippiOS/backend/internal/gecko"
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
	if err := gecko.EnsureQuitCode(paths.ProfileDir); err != nil {
		alert([]string{"Could not install the Quit button.", err.Error()})
		return
	}
	if err := dolphin.EnableQuitSupport(paths.DolphinDir); err != nil {
		alert([]string{"Could not install the Quit button.", err.Error()})
		return
	}
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
		"-C", "Logger.Logs.SLIPPI=True",
		"-C", "Logger.Options.WriteToConsole=True",
		"-C", "Logger.Options.WriteToFile=True",
		"-C", "Logger.Options.Verbosity=4",
		"-C", "Graphics.Settings.HiresTextures=True",
		"-C", "Graphics.Settings.CacheHiresTextures=True",
		"-b", "-e", iso,
	)
	environment := environmentWithRuntime()
	cacheDir := filepath.Join(paths.ProfileDir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		filtered := environment[:0]
		for _, value := range environment {
			if strings.HasPrefix(value, "XDG_CACHE_HOME=") {
				continue
			}
			filtered = append(filtered, value)
		}
		environment = append(filtered, "XDG_CACHE_HOME="+cacheDir)
	}
	command.Env = environment
	command.Stdout = logFile
	command.Stderr = logFile
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			alert([]string{"slippi-appimage-run is not installed in this image."})
			return
		}
		alert([]string{"Could not start Slippi Dolphin.", err.Error()})
		return
	}
	done := make(chan struct{})
	go dolphin.WatchQuit(logPath, command.Process.Pid, done)
	runErr := command.Wait()
	close(done)
	if dolphin.QuitRequested(logPath) {
		return
	}
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
