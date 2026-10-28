// Package replay lists Slippi replays and launches the Ishiiruka playback
// build. The netplay Dolphin AppImage cannot play .slp files; Slippi ships a
// separate playback build for that.
package replay

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/SlippiOS/SlippiOS/backend/internal/desktop"
	"github.com/SlippiOS/SlippiOS/backend/internal/stable"
)

const (
	graphqlURL      = "https://internal.slippi.gg/graphql"
	fallbackURL     = "https://github.com/project-slippi/Ishiiruka-Playback/releases/download/v3.5.2/playback-3.5.2-Linux.zip"
	fallbackVersion = "3.5.2"
	appimageName    = "Slippi_Playback-x86_64.AppImage"
	userAgent       = "SlippiOS/0.2"
)

type Replay struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
}

func List(profileDir string) ([]Replay, error) {
	directories := []string{
		filepath.Join(profileDir, "replays"),
		filepath.Join(profileDir, "dolphin", "Slippi"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		directories = append(directories, filepath.Join(home, "Slippi"))
	}

	seen := map[string]bool{}
	var replays []Replay
	for _, directory := range directories {
		entries, err := os.ReadDir(directory)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".slp") {
				continue
			}
			path := filepath.Join(directory, entry.Name())
			if seen[path] {
				continue
			}
			seen[path] = true
			info, err := entry.Info()
			if err != nil {
				continue
			}
			replays = append(replays, Replay{
				Name:     entry.Name(),
				Path:     path,
				Size:     info.Size(),
				Modified: info.ModTime().Unix(),
			})
		}
	}
	sort.Slice(replays, func(i, j int) bool {
		return replays[i].Modified > replays[j].Modified
	})
	return replays, nil
}

func PlaybackPath(dataDir string) string {
	return filepath.Join(dataDir, "system", "playback")
}

// EnsurePlayback downloads and extracts the latest playback build. The zip
// contains the AppImage and the Sys/ directory it expects next to it.
func EnsurePlayback(dataDir string, log func(string)) (string, error) {
	if log == nil {
		log = func(string) {}
	}
	base := PlaybackPath(dataDir)
	versionFile := filepath.Join(base, ".version")
	version, url := latestPlayback()
	if data, err := os.ReadFile(versionFile); err == nil && strings.TrimSpace(string(data)) == version {
		candidate := filepath.Join(base, version, appimageName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	log("Downloading Slippi playback " + version + "…")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	archive := filepath.Join(base, "playback-"+version+".zip")
	if err := download(url, archive); err != nil {
		return "", err
	}
	staging := filepath.Join(base, version+".tmp")
	_ = os.RemoveAll(staging)
	if err := unzip(archive, staging); err != nil {
		_ = os.RemoveAll(staging)
		return "", err
	}
	target := filepath.Join(base, version)
	_ = os.RemoveAll(target)
	if err := os.Rename(staging, target); err != nil {
		return "", err
	}
	_ = os.Remove(archive)
	if err := os.WriteFile(versionFile, []byte(version+"\n"), 0o644); err != nil {
		return "", err
	}
	return filepath.Join(target, appimageName), nil
}

func Play(dataDir, profileDir, isoPath, replayPath string, values map[string]any) (int, error) {
	if _, err := os.Stat(replayPath); err != nil {
		return 0, fmt.Errorf("replay not found: %s", replayPath)
	}
	if _, err := os.Stat(isoPath); err != nil {
		return 0, fmt.Errorf("Melee image not found: %s", isoPath)
	}
	appimage, err := EnsurePlayback(dataDir, func(message string) {
		fmt.Fprintln(os.Stderr, message)
	})
	if err != nil {
		return 0, err
	}

	commsDir := PlaybackPath(dataDir)
	if err := os.MkdirAll(commsDir, 0o755); err != nil {
		return 0, err
	}
	commsPath := filepath.Join(commsDir, "comms.json")
	comms := map[string]any{
		"mode":      "normal",
		"replay":    replayPath,
		"commandId": fmt.Sprintf("%d", time.Now().UnixNano()),
	}
	data, err := json.Marshal(comms)
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(commsPath, data, 0o644); err != nil {
		return 0, err
	}

	userDir := filepath.Join(profileDir, "dolphin-playback")
	cacheDir := filepath.Join(profileDir, "cache")
	logDir := filepath.Join(profileDir, "logs")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return 0, err
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return 0, err
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return 0, err
	}

	width, height := 640, 480
	if w, h, err := desktop.CurrentResolution(); err == nil {
		width, height = w, h
	}
	if err := stable.Configure(profileDir, userDir, filepath.Dir(isoPath), playbackSettings(values), width, height, true); err != nil {
		return 0, err
	}

	args := []string{
		appimage,
		"-u", userDir,
		"-b", "-e", isoPath,
		"-i", commsPath,
		"--hide-seekbar",
	}
	if os.Getenv("SLIPPIOS_REPLAY_COUT") == "1" {
		args = append(args, "--cout")
	}
	command := exec.Command("slippi-appimage-run", args...)
	command.Dir = filepath.Dir(appimage)
	environment := os.Environ()
	filtered := environment[:0]
	for _, value := range environment {
		if strings.HasPrefix(value, "XDG_CACHE_HOME=") {
			continue
		}
		filtered = append(filtered, value)
	}
	command.Env = append(filtered, "XDG_CACHE_HOME="+cacheDir)
	logFile, err := os.Create(filepath.Join(logDir, "replay.log"))
	if err != nil {
		return 0, err
	}
	defer logFile.Close()
	command.Stdout = logFile
	command.Stderr = logFile
	if err := command.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}

func playbackSettings(values map[string]any) map[string]any {
	configured, _ := values["dolphin"].(map[string]any)
	backend := os.Getenv("SLIPPIOS_PLAYBACK_BACKEND")
	if backend == "" {
		backend, _ = configured["backend"].(string)
	}
	if backend == "" {
		// The 3.5.2 playback build's Vulkan path opens the Dolphin GUI on the
		// EM680 instead of starting the replay. OpenGL starts reliably.
		backend = "OGL"
	}
	resolution := int64(3)
	aspect := int64(3)
	msaa := int64(1)
	vsync := false
	if value, ok := configured["resolution"].(int64); ok {
		resolution = value
	}
	if value, ok := configured["aspect"].(int64); ok {
		aspect = value
	}
	if value, ok := configured["msaa"].(int64); ok {
		msaa = value
	}
	if value, ok := configured["vsync"].(bool); ok {
		vsync = value
	}
	dolphin := map[string]any{
		"backend":    backend,
		"resolution": resolution,
		"aspect":     aspect,
		"msaa":       msaa,
		"vsync":      vsync,
	}
	for port := 1; port <= 4; port++ {
		dolphin[fmt.Sprintf("controller%d", port)] = "none"
		dolphin[fmt.Sprintf("profile%d", port)] = ""
	}
	return map[string]any{"dolphin": dolphin}
}

func latestPlayback() (string, string) {
	body := `{"query":"query { getLatestDolphin(purpose: PLAYBACK, includeBeta: true) { linuxDownloadUrl version } }"}`
	request, err := http.NewRequest(http.MethodPost, graphqlURL, strings.NewReader(body))
	if err != nil {
		return fallbackVersion, fallbackURL
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", userAgent)
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return fallbackVersion, fallbackURL
	}
	defer response.Body.Close()
	var payload struct {
		Data struct {
			Latest struct {
				URL     string `json:"linuxDownloadUrl"`
				Version string `json:"version"`
			} `json:"getLatestDolphin"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return fallbackVersion, fallbackURL
	}
	if payload.Data.Latest.URL == "" || payload.Data.Latest.Version == "" {
		return fallbackVersion, fallbackURL
	}
	return payload.Data.Latest.Version, payload.Data.Latest.URL
}

func download(url, destination string) error {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", userAgent)
	client := &http.Client{Timeout: 10 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return fmt.Errorf("download returned %s", response.Status)
	}
	partial := destination + ".part"
	file, err := os.Create(partial)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, response.Body); err != nil {
		file.Close()
		_ = os.Remove(partial)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(partial)
		return err
	}
	return os.Rename(partial, destination)
}

func unzip(archive, destination string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	cleanDestination := filepath.Clean(destination) + string(os.PathSeparator)
	for _, file := range reader.File {
		target := filepath.Join(destination, file.Name)
		if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), cleanDestination) {
			return fmt.Errorf("zip entry escapes destination: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		source, err := file.Open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
		if err != nil {
			source.Close()
			return err
		}
		_, copyErr := io.Copy(output, source)
		source.Close()
		output.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}
