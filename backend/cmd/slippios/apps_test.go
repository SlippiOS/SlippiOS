package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SlippiOS/SlippiOS/backend/internal/apps"
	"github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
	"github.com/SlippiOS/SlippiOS/backend/internal/replay"
)

func TestAppsStatusPayload(t *testing.T) {
	data := t.TempDir()
	t.Setenv("SLIPPIOS_DATA", data)

	if payload := appsStatusPayload(); payload["installed"] != false {
		t.Fatalf("empty data dir reports installed: %#v", payload)
	}

	writeTestFile(t, apps.StablePath(data))
	writeTestFile(t, dolphin.BundledPath())
	playbackDir := replay.PlaybackPath(data)
	writeTestFile(t, filepath.Join(playbackDir, "3.5.2", "Slippi_Playback-x86_64.AppImage"))
	if err := os.WriteFile(filepath.Join(playbackDir, ".version"), []byte("3.5.2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	payload := appsStatusPayload()
	if payload["installed"] != true {
		t.Fatalf("installed apps not detected: %#v", payload)
	}
	if payload["beta"].(map[string]any)["version"] != "v4.0.0-mainline-beta.19" {
		t.Fatalf("beta version = %#v", payload["beta"])
	}
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
}
