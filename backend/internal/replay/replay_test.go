package replay

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListFindsSLP(t *testing.T) {
	profile := t.TempDir()
	dir := filepath.Join(profile, "replays")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"one.slp", "two.SLP", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	replays, err := List(profile)
	if err != nil {
		t.Fatal(err)
	}
	if len(replays) != 2 {
		t.Fatalf("replays = %#v", replays)
	}
	for _, item := range replays {
		if item.Path == "" || item.Name == "" {
			t.Fatalf("replay = %#v", item)
		}
	}
}

func TestLastPlaybackFrame(t *testing.T) {
	path := filepath.Join(t.TempDir(), "replay.log")
	if err := os.WriteFile(path, []byte("noise\n[CURRENT_FRAME] 12\n[CURRENT_FRAME] 13\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	frame, ok := lastPlaybackFrame(path)
	if !ok || frame != 13 {
		t.Fatalf("frame = %d, ok = %v", frame, ok)
	}
}

func TestListEmptyReturnsArray(t *testing.T) {
	profile := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(profile, "replays"), 0o755); err != nil {
		t.Fatal(err)
	}
	replays, err := List(profile)
	if err != nil {
		t.Fatal(err)
	}
	if replays == nil {
		t.Fatal("empty replay list is nil")
	}
}
