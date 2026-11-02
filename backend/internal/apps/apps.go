// Package apps manages the bundled stable and playback Slippi applications.
// The release image seeds them under /data/system/apps; the download fallback
// exists for dev images and for installing them on an existing console.
package apps

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	graphqlURL = "https://internal.slippi.gg/graphql"
	userAgent  = "SlippiOS/0.2"
)

var (
	stableFallbackVersion = "3.6.4"
	stableFallbackURL     = "https://github.com/project-slippi/Ishiiruka/releases/download/v3.6.4/FM-Slippi-3.6.4-Linux.zip"
	playbackFallbackURL   = "https://github.com/project-slippi/Ishiiruka-Playback/releases/download/v3.5.2/playback-3.5.2-Linux.zip"
	playbackFallbackVer   = "3.5.2"
)

func StableDir(dataDir string) string {
	return filepath.Join(dataDir, "system", "apps", "stable")
}

func PlaybackDir(dataDir string) string {
	return filepath.Join(dataDir, "system", "apps", "playback")
}

func StablePath(dataDir string) string {
	return filepath.Join(StableDir(dataDir), "Slippi_Online-x86_64.AppImage")
}

// StableVersion is the pinned stable build shipped with the image.
func StableVersion() string {
	return stableFallbackVersion
}

func PlaybackPath(dataDir string) string {
	return filepath.Join(PlaybackDir(dataDir), "Slippi_Playback-x86_64.AppImage")
}

func EnsureStable(dataDir string, log func(string)) (string, error) {
	// A release image seeds the pinned build here. Use it without contacting
	// the network; the download fallback is only for dev images and repairs.
	if info, err := os.Stat(StablePath(dataDir)); err == nil && !info.IsDir() {
		return StablePath(dataDir), nil
	}
	version, url := latest("NETPLAY", false, stableFallbackVersion, stableFallbackURL)
	return ensure(dataDir, StableDir(dataDir), "Slippi_Online-x86_64.AppImage", version, url, log)
}

func EnsurePlayback(dataDir string, log func(string)) (string, error) {
	if info, err := os.Stat(PlaybackPath(dataDir)); err == nil && !info.IsDir() {
		return PlaybackPath(dataDir), nil
	}
	version, url := latest("PLAYBACK", true, playbackFallbackVer, playbackFallbackURL)
	return ensure(dataDir, PlaybackDir(dataDir), "Slippi_Playback-x86_64.AppImage", version, url, log)
}

func ensure(dataDir, directory, appName, version, url string, log func(string)) (string, error) {
	if log == nil {
		log = func(string) {}
	}
	app := filepath.Join(directory, appName)
	if info, err := os.Stat(app); err == nil && !info.IsDir() {
		return app, nil
	}
	if err := os.MkdirAll(filepath.Dir(directory), 0o755); err != nil {
		return "", err
	}
	log("Downloading Slippi app " + version + "…")
	archive := filepath.Join(filepath.Dir(directory), filepath.Base(directory)+"-"+version+".zip")
	if err := download(url, archive); err != nil {
		return "", err
	}
	staging := directory + ".tmp"
	_ = os.RemoveAll(staging)
	if err := unzip(archive, staging); err != nil {
		_ = os.RemoveAll(staging)
		return "", err
	}
	_ = os.RemoveAll(directory)
	if err := os.Rename(staging, directory); err != nil {
		return "", err
	}
	_ = os.Remove(archive)
	if err := os.Chmod(filepath.Join(directory, appName), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(directory, ".version"), []byte(version+"\n"), 0o644); err != nil {
		return "", err
	}
	return filepath.Join(directory, appName), nil
}

func latest(purpose string, includeBeta bool, fallbackVersion, fallbackURL string) (string, string) {
	query := fmt.Sprintf(`{"query":"query { getLatestDolphin(purpose: %s, includeBeta: %t) { linuxDownloadUrl version } }"}`, purpose, includeBeta)
	request, err := http.NewRequest(http.MethodPost, graphqlURL, strings.NewReader(query))
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
		mode := os.FileMode(0o644)
		if strings.HasSuffix(file.Name, ".AppImage") {
			mode = 0o755
		}
		output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
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
