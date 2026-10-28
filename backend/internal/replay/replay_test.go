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
