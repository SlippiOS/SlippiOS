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

func TestLoadMigratesSingleController(t *testing.T) {
	paths := PathsFor(t.TempDir())
	if err := os.MkdirAll(filepath.Dir(paths.SettingsFile), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `schema_version = 1

[dolphin]
controller = "pad"
`
	if err := os.WriteFile(paths.SettingsFile, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := paths.Load()
	if err != nil {
		t.Fatal(err)
	}
	dolphin := loaded["dolphin"].(map[string]any)
	if dolphin["controller1"] != "pad" {
		t.Fatalf("controller1 = %v", dolphin["controller1"])
	}
	if dolphin["controller2"] != "adapter" {
		t.Fatalf("controller2 = %v", dolphin["controller2"])
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

func TestInstallLogin(t *testing.T) {
	paths := PathsFor(t.TempDir())
	if err := os.MkdirAll(paths.ProfileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{
		"uid": "u", "playKey": "k", "connectCode": "ABC#123",
	}
	data, _ := json.Marshal(payload)
	source := paths.Resolve(DefaultUserJSON)
	if err := os.WriteFile(source, data, 0o600); err != nil {
		t.Fatal(err)
	}
	ok, detail := paths.InstallLogin(Defaults())
	if !ok || detail != "ABC#123" {
		t.Fatalf("install = %v %q", ok, detail)
	}
	installed, err := os.ReadFile(paths.DolphinUserFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(data) {
		t.Fatalf("installed = %s", installed)
	}
	info, err := os.Stat(paths.DolphinUserFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}
}

func TestCheckHardwareBinding(t *testing.T) {
	paths := PathsFor(t.TempDir())
	device := filepath.Join(t.TempDir(), "product_uuid")
	if err := os.WriteFile(device, []byte("machine-1\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	saved := hardwareIDPaths
	hardwareIDPaths = []string{device}
	t.Cleanup(func() { hardwareIDPaths = saved })

	if err := os.MkdirAll(filepath.Dir(paths.HardwareFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.DolphinUserFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.ProfileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{paths.HardwareFile, paths.DolphinUserFile, paths.Resolve(DefaultUserJSON)} {
		if err := os.WriteFile(path, []byte(`{"uid":"u","playKey":"k","connectCode":"ABC#123"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(paths.HardwareFile, []byte("machine-2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !paths.CheckHardwareBinding() {
		t.Fatal("hardware change not detected")
	}
	for _, path := range []string{paths.HardwareFile, paths.DolphinUserFile, paths.Resolve(DefaultUserJSON)} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still exists", path)
		}
	}
	if paths.CheckHardwareBinding() {
		t.Fatal("first binding reported a change")
	}
	stored, err := os.ReadFile(paths.HardwareFile)
	if err != nil || string(stored) != "machine-1\n" {
		t.Fatalf("hardware file = %q, %v", stored, err)
	}
	if paths.CheckHardwareBinding() {
		t.Fatal("same hardware reported a change")
	}
}
