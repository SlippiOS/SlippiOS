package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadDefaults(t *testing.T) {
	paths := PathsFor(t.TempDir())
	values := Defaults()
	values["network"].(map[string]any)["ssid"] = "Studio"
	if err := paths.Save(values); err != nil {
		t.Fatal(err)
	}
	loaded, err := paths.Load()
	if err != nil {
		t.Fatal(err)
	}
	network := loaded["network"].(map[string]any)
	if network["ssid"] != "Studio" {
		t.Fatalf("ssid = %v", network["ssid"])
	}
	if loaded["schema_version"].(int64) != SchemaVersion {
		t.Fatalf("schema = %v", loaded["schema_version"])
	}
}

func TestISOStatus(t *testing.T) {
	paths := PathsFor(t.TempDir())
	if err := os.MkdirAll(paths.ISODir, 0o755); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(paths.ISODir, "good.iso")
	if err := os.WriteFile(good, append([]byte("GALE01"), make([]byte, 32)...), 0o444); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(paths.ISODir, "bad.iso")
	if err := os.WriteFile(bad, []byte("not-a-melee-image"), 0o444); err != nil {
		t.Fatal(err)
	}
	if ok, reason := paths.ISOStatus("good.iso"); !ok {
		t.Fatalf("good rejected: %s", reason)
	}
	if ok, _ := paths.ISOStatus("bad.iso"); ok {
		t.Fatal("bad accepted")
	}
	if ok, _ := paths.ISOStatus("missing.iso"); ok {
		t.Fatal("missing accepted")
	}
}

func TestFindISOs(t *testing.T) {
	paths := PathsFor(t.TempDir())
	if err := os.MkdirAll(paths.ISODir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"one.iso", "two.iso"} {
		if err := os.WriteFile(filepath.Join(paths.ISODir, name),
			append([]byte("GALE01"), make([]byte, 8)...), 0o444); err != nil {
			t.Fatal(err)
		}
	}
	found := paths.FindISOsIn([]string{paths.ISODir})
	if len(found) != 2 || !found[0].Valid || !found[1].Valid {
		t.Fatalf("found = %#v", found)
	}
}

func TestLoginStatus(t *testing.T) {
	paths := PathsFor(t.TempDir())
	if err := os.MkdirAll(filepath.Dir(paths.DolphinUserFile), 0o755); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{
		"uid": "u", "playKey": "k", "connectCode": "ABC#123",
	}
	data, _ := json.Marshal(payload)
	if err := os.WriteFile(paths.DolphinUserFile, data, 0o600); err != nil {
		t.Fatal(err)
	}
	ok, detail := paths.LoginStatus(paths.DolphinUserFile)
	if !ok || detail != "ABC#123" {
		t.Fatalf("login = %v %q", ok, detail)
	}
}
