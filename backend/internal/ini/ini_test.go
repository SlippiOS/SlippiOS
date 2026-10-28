package ini

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAndRoundTrip(t *testing.T) {
	data := []byte("# comment\n\n[Core]\nFastDiscSpeed = True\n\n[Gecko]\n$Optional: Test\n04123456 00000000\n")
	file := Parse(data)
	if got := file.Lines("Core"); len(got) != 1 || got[0] != "FastDiscSpeed = True" {
		t.Fatalf("core = %#v", got)
	}
	if got := file.Lines("Gecko"); len(got) != 2 || got[0] != "$Optional: Test" {
		t.Fatalf("gecko = %#v", got)
	}
	file.SetLines("Gecko_Enabled", []string{"$Optional: Test"})
	out := string(file.Bytes())
	if !strings.Contains(out, "[Gecko_Enabled]\n$Optional: Test\n") {
		t.Fatalf("output = %s", out)
	}
	if !strings.Contains(out, "FastDiscSpeed = True") {
		t.Fatalf("lost core: %s", out)
	}
}

func TestLoadMissingAndSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Config", "GCPadNew.ini")
	file, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	file.SetLines("GCPad1", []string{"Device = evdev/0/Pad"})
	if err := Save(path, file); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}
	again, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := again.Lines("GCPad1"); len(got) != 1 || got[0] != "Device = evdev/0/Pad" {
		t.Fatalf("reloaded = %#v", got)
	}
}
