package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SlippiOS/SlippiOS/backend/internal/apps"
	"github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
	"github.com/SlippiOS/SlippiOS/backend/internal/replay"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/spf13/cobra"
)

type appsInstallOptions struct {
	progress string
}

func newAppsCommand(mode *modes) *cobra.Command {
	command := &cobra.Command{
		Use:   "apps",
		Short: "Inspect or install the bundled Slippi apps",
		Args:  cobra.NoArgs,
	}
	command.AddCommand(newAppsStatusCommand(mode))
	command.AddCommand(newAppsInstallCommand(mode))
	return command
}

func newAppsStatusCommand(mode *modes) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show whether the stable, beta, and playback apps are installed",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			payload := appsStatusPayload()
			return mode.emit(payload, func() { printAppsStatus(mode, payload) })
		},
	}
}

func newAppsInstallCommand(mode *modes) *cobra.Command {
	options := appsInstallOptions{}
	command := &cobra.Command{
		Use:   "install",
		Short: "Download and install the stable, beta, and playback apps",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAppsInstall(mode, options)
		},
	}
	command.Flags().StringVar(&options.progress, "progress", "", "write download progress as JSON to this file")
	return command
}

func appsStatusPayload() map[string]any {
	paths := settings.DefaultPaths()
	stablePath := apps.StablePath(paths.DataDir)
	stable := fileStatus(stablePath)
	stableVersion := readTrimmed(filepath.Join(apps.StableDir(paths.DataDir), ".version"))
	if stableVersion == "" {
		stableVersion = apps.StableVersion()
	}
	stable["version"] = stableVersion

	dolphinPaths := dolphin.DefaultPaths()
	betaPath, betaInstalled := dolphinPaths.CurrentPath()
	if !betaInstalled {
		if info, err := os.Stat(dolphin.BundledPath()); err == nil && !info.IsDir() {
			betaPath = dolphin.BundledPath()
			betaInstalled = true
		}
	}
	betaVersion, _ := dolphinPaths.CurrentTag()
	beta := map[string]any{
		"installed": betaInstalled,
		"path":      betaPath,
		"version":   betaVersion,
	}

	playbackBase := replay.PlaybackPath(paths.DataDir)
	playbackVersion := readTrimmed(filepath.Join(playbackBase, ".version"))
	playbackPath := ""
	playbackInstalled := false
	if playbackVersion != "" {
		candidate := filepath.Join(playbackBase, playbackVersion, "Slippi_Playback-x86_64.AppImage")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			playbackPath = candidate
			playbackInstalled = true
		}
	}
	playback := map[string]any{
		"installed": playbackInstalled,
		"path":      playbackPath,
		"version":   playbackVersion,
	}

	return map[string]any{
		"ok":        true,
		"stable":    stable,
		"beta":      beta,
		"playback":  playback,
		"installed": stable["installed"] == true && betaInstalled && playbackInstalled,
	}
}

func fileStatus(path string) map[string]any {
	info, err := os.Stat(path)
	return map[string]any{
		"installed": err == nil && !info.IsDir(),
		"path":      path,
	}
}

func printAppsStatus(mode *modes, payload map[string]any) {
	labels := map[string]string{"stable": "Stable", "beta": "Beta", "playback": "Playback"}
	for _, name := range []string{"stable", "beta", "playback"} {
		info := payload[name].(map[string]any)
		state := "missing"
		if info["installed"] == true {
			state = "installed"
		}
		version, _ := info["version"].(string)
		if version != "" {
			mode.printf("%-9s %s (%s)\n", labels[name], state, version)
		} else {
			mode.printf("%-9s %s\n", labels[name], state)
		}
	}
}

func runAppsInstall(mode *modes, options appsInstallOptions) error {
	report := func(value map[string]any) {}
	if options.progress != "" {
		report = func(value map[string]any) { writeProgress(options.progress, value) }
	}
	log := func(message string) { fmt.Fprintln(mode.stderr(), message) }
	paths := settings.DefaultPaths()

	report(map[string]any{"phase": "stable"})
	stablePath, err := apps.EnsureStable(paths.DataDir, log)
	if err != nil {
		report(map[string]any{"phase": "error", "error": err.Error()})
		return mode.fail(err.Error())
	}

	report(map[string]any{"phase": "beta"})
	dolphinPaths := dolphin.DefaultPaths()
	betaPath, err := dolphinPaths.EnsureWithProgress(log, func(done, total int64) {
		report(map[string]any{"phase": "beta", "done": done, "total": total})
	}, false)
	if err != nil {
		report(map[string]any{"phase": "error", "error": err.Error()})
		return mode.fail(err.Error())
	}

	report(map[string]any{"phase": "playback"})
	playbackPath, err := replay.EnsurePlayback(paths.DataDir, log)
	if err != nil {
		report(map[string]any{"phase": "error", "error": err.Error()})
		return mode.fail(err.Error())
	}

	report(map[string]any{"phase": "ready"})
	betaVersion, _ := dolphinPaths.CurrentTag()
	playbackVersion := readTrimmed(filepath.Join(replay.PlaybackPath(paths.DataDir), ".version"))
	return mode.emit(map[string]any{
		"ok":     true,
		"stable": map[string]any{"path": stablePath},
		"beta":   map[string]any{"path": betaPath, "version": betaVersion},
		"playback": map[string]any{
			"path":    playbackPath,
			"version": playbackVersion,
		},
	}, func() {
		mode.println("Installed stable, beta, and playback apps.")
	})
}
