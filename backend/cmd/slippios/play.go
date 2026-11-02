package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/SlippiOS/SlippiOS/backend/internal/apps"
	"github.com/SlippiOS/SlippiOS/backend/internal/controllers"
	"github.com/SlippiOS/SlippiOS/backend/internal/desktop"
	dolphinpkg "github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/SlippiOS/SlippiOS/backend/internal/stable"
	"github.com/spf13/cobra"
)

type playOptions struct {
	requireLogin bool
	readyFile    string
	cancelFile   string
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
	command.Flags().StringVar(&options.readyFile, "ready-file", "", "write this file once the Dolphin window is visible")
	command.Flags().StringVar(&options.cancelFile, "cancel-file", "", "stop Dolphin when this file appears")
	return command
}

func runPlay(mode *modes, options playOptions) error {
	paths := settings.DefaultPaths()
	if installed, _ := paths.InstallLogin(settings.Defaults()); !installed && options.requireLogin {
		return mode.fail("not signed in")
	}
	values, err := paths.Load()
	if err != nil {
		return mode.fail("no game image")
	}
	if changed, err := controllers.NormalizeProfiles(values, paths.DolphinDir); err != nil {
		return mode.fail(err.Error())
	} else if changed {
		if err := paths.Save(values); err != nil {
			return mode.fail(err.Error())
		}
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
	dolphinSettings, _ := values["dolphin"].(map[string]any)
	build := "stable"
	if value, ok := dolphinSettings["build"].(string); ok && value != "" {
		build = value
	}
	var appimage string
	var userDir string
	var arguments []string
	if build == "beta" {
		appimage, err = dolphinpkg.DefaultPaths().Ensure(func(message string) {
			fmt.Fprintln(mode.stderr(), message)
		}, false)
		if err != nil {
			return mode.fail(err.Error())
		}
		if err := controllers.ApplyConfigured(paths.DolphinDir, values); err != nil {
			return mode.fail(err.Error())
		}
		if err := controllers.ConfigureHotkeys(paths.DolphinDir, values); err != nil {
			return mode.fail(err.Error())
		}
		userDir = paths.DolphinDir
		arguments = dolphinArgs(values)
		arguments = append(arguments,
			"-C", "Dolphin.Display.Fullscreen=True",
			"-C", "Dolphin.Display.BorderlessFullscreen=True",
			"-C", "Dolphin.Display.RenderWindowAutoSize=False",
			"-C", fmt.Sprintf("Dolphin.Display.RenderWindowWidth=%d", width),
			"-C", fmt.Sprintf("Dolphin.Display.RenderWindowHeight=%d", height),
			"-C", "Dolphin.Display.RenderWindowXPos=0",
			"-C", "Dolphin.Display.RenderWindowYPos=0",
			"-b", "-e", iso)
	} else {
		appimage, err = apps.EnsureStable(paths.DataDir, func(message string) {
			fmt.Fprintln(mode.stderr(), message)
		})
		if err != nil {
			return mode.fail(err.Error())
		}
		if err := stable.Configure(paths.ProfileDir, paths.DolphinStableDir, paths.ISODir, values, width, height, true); err != nil {
			return mode.fail(err.Error())
		}
		userDir = paths.DolphinStableDir
		arguments = []string{"-b", "-e", iso}
	}
	returncode, err := runDolphin(paths, appimage, userDir, arguments, options.readyFile, options.cancelFile)
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
	flagDefault := func(key string, fallback bool) bool {
		if value, ok := dolphin[key].(bool); ok {
			return value
		}
		return fallback
	}
	boolText := func(value bool) string {
		if value {
			return "True"
		}
		return "False"
	}

	arguments := []string{
		// The Slippi EXI device lives in Slot B. Do not rely on the build's
		// default, because an existing Dolphin.ini can override it.
		"-C", "Dolphin.Core.SlotB=13",
		"-C", "Dolphin.Core.GFXBackend=" + text("backend", "Vulkan"),
		"-C", "Graphics.Settings.InternalResolution=" + strconv.FormatInt(number("resolution", 3), 10),
		"-C", "Graphics.Settings.AspectRatio=" + strconv.FormatInt(number("aspect", 3), 10),
		"-C", "Graphics.Settings.MSAA=" + strconv.FormatInt(number("msaa", 1), 10),
		"-C", "Graphics.Hardware.VSync=" + boolText(flag("vsync")),
		"-C", "Dolphin.DSP.Backend=" + text("audio_backend", "Cubeb"),
		"-C", "Dolphin.Core.AudioLatency=" + strconv.FormatInt(number("audio_latency", 20), 10),
		"-C", "Dolphin.Slippi.ReplayDir=" + filepath.Join(settings.DefaultPaths().ProfileDir, "replays"),
		"-C", "Dolphin.Slippi.SaveReplays=True",
		"-C", "Dolphin.Slippi.EnableJukebox=" + boolText(flagDefault("enable_music", true)),
		"-C", "Graphics.Settings.WaitForShadersBeforeStarting=" + boolText(flag("precompile_shaders")),
		"-C", "Graphics.Settings.ShaderCompilationMode=" + shaderModeArg(values),
		"-C", "Graphics.Settings.ShowNetPlayPing=" + boolText(flagDefault("show_netplay_ping", true)),
		"-C", "Graphics.Settings.ShowFPS=" + boolText(flag("show_fps")),
	}
	for port := 0; port < 4; port++ {
		device := text(fmt.Sprintf("controller%d", port+1), "adapter")
		value := 0
		switch device {
		case "adapter":
			value = 12
		case "pad":
			value = 6
		case "keyboard":
			value = 7
		}
		arguments = append(arguments,
			"-C", fmt.Sprintf("Dolphin.Core.SIDevice%d=%d", port, value))
	}
	return arguments
}

func shaderModeArg(values map[string]any) string {
	dolphin, _ := values["dolphin"].(map[string]any)
	mode, _ := dolphin["shader_mode"].(string)
	return strconv.FormatInt(dolphinpkg.ShaderModeValue(mode), 10)
}

func runDolphin(paths settings.Paths, appimage, userDir string, arguments []string, readyFile, cancelFile string) (int, error) {
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

	args := append([]string{appimage, "-u", userDir}, arguments...)
	command := exec.Command("slippi-appimage-run", args...)
	command.Dir = filepath.Dir(appimage)
	cacheDir := filepath.Join(paths.ProfileDir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		filtered := make([]string, 0, len(environment)+2)
		existingLibraryPath := ""
		for _, value := range environment {
			if strings.HasPrefix(value, "XDG_CACHE_HOME=") {
				continue
			}
			if strings.HasPrefix(value, "LD_LIBRARY_PATH=") {
				existingLibraryPath = strings.TrimPrefix(value, "LD_LIBRARY_PATH=")
				continue
			}
			filtered = append(filtered, value)
		}
		libraryPath := "/usr/lib64"
		if existingLibraryPath != "" {
			libraryPath += ":" + existingLibraryPath
		}
		environment = append(filtered, "XDG_CACHE_HOME="+cacheDir, "LD_LIBRARY_PATH="+libraryPath)
	}
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

	// Keep Dolphin and the AppImage wrapper in their own process group so a
	// cancelled launch can be stopped without touching the shell.
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := command.Start(); err != nil {
		return 0, err
	}
	done := make(chan struct{})
	var windowSeen atomic.Bool
	if readyFile != "" {
		go watchDolphinWindow(readyFile, done, &windowSeen)
	}
	if cancelFile != "" {
		go watchCancel(cancelFile, command.Process.Pid, done)
	}
	err := command.Wait()
	close(done)
	if cancelFile != "" {
		_ = os.Remove(cancelFile)
	}
	// The game can map its window and exit before the shell notices the ready
	// file. Record the successful start so the shell returns to the menu
	// instead of reporting a launch failure.
	if readyFile != "" && windowSeen.Load() {
		_ = os.WriteFile(readyFile, []byte("ready\n"), 0o644)
	}
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return exitError.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}

// watchDolphinWindow writes the ready file once Dolphin has mapped its game
// window. The shell keeps its launching screen visible until then. There is no
// deadline: a cold first start can spend minutes extracting the AppImage
// before the window appears, and the watcher ends when Dolphin exits.
func watchDolphinWindow(path string, done <-chan struct{}, seen *atomic.Bool) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if dolphinWindowVisible() {
				seen.Store(true)
				_ = os.WriteFile(path, []byte("ready\n"), 0o644)
				return
			}
		}
	}
}

// watchCancel stops the launch when the shell creates the cancel file. The
// Dolphin process group gets SIGTERM first and SIGKILL if it does not exit.
func watchCancel(path string, pid int, done <-chan struct{}) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if _, err := os.Stat(path); err != nil {
				continue
			}
			_ = syscall.Kill(-pid, syscall.SIGTERM)
			select {
			case <-done:
				return
			case <-time.After(3 * time.Second):
				_ = syscall.Kill(-pid, syscall.SIGKILL)
			}
			return
		}
	}
}

func dolphinWindowVisible() bool {
	// Stable and playback title themselves "Faster Melee - Slippi (...)";
	// mainline uses a Dolphin title. Exclude the shell's "SlippiOS Shell".
	output, err := exec.Command("xdotool", "search", "--onlyvisible", "--name",
		`Faster Melee|Dolphin|Slippi \(`).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}
