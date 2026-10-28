package gecko

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	profile := t.TempDir()
	globalDir := filepath.Join(profile, "cache", "appimage-run", "hash", "usr", "bin", "Sys",
		"GameSettings")
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	global := `[Gecko_Enabled]
$Required: General Codes
$Recommended: Normal Lag Reduction

[Gecko]
$Required: General Codes [Achilles]
0415EE98 38600001
$Recommended: Normal Lag Reduction [Hannes]
04123456 00000000
$Optional: Widescreen 16:9 [Dan]
04123456 00000001
`
	if err := os.WriteFile(filepath.Join(globalDir, "GALE01r2.ini"), []byte(global), 0o644); err != nil {
		t.Fatal(err)
	}
	return profile
}

func TestLoadDefaults(t *testing.T) {
	profile := fixture(t)
	codes, err := Load(profile)
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 3 {
		t.Fatalf("codes = %#v", codes)
	}
	if !codes[0].Enabled || !codes[0].Required || codes[0].DefaultEnabled != true {
		t.Fatalf("required = %#v", codes[0])
	}
	if codes[2].Enabled || codes[2].Required {
		t.Fatalf("optional = %#v", codes[2])
	}
}

func TestSetEnabledWritesOverride(t *testing.T) {
	profile := fixture(t)
	if err := SetEnabled(profile, "Optional: Widescreen 16:9", true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(LocalPath(profile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "[Gecko_Enabled]\n$Optional: Widescreen 16:9\n") {
		t.Fatalf("local = %s", data)
	}
	codes, err := Load(profile)
	if err != nil {
		t.Fatal(err)
	}
	if !codes[2].Enabled {
		t.Fatalf("widescreen not enabled: %#v", codes[2])
	}
}

func TestRequiredCannotBeDisabled(t *testing.T) {
	profile := fixture(t)
	if err := SetEnabled(profile, "Required: General Codes", false); err == nil {
		t.Fatal("required code disabled")
	}
}

func TestAddAndRemoveCustomCode(t *testing.T) {
	profile := fixture(t)
	added, err := Add(profile, "$Custom: Test [Me]\n*note\n04123456 00000000\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 1 || added[0].Name != "Custom: Test" || added[0].Creator != "Me" {
		t.Fatalf("added = %#v", added)
	}
	codes, err := Load(profile)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, code := range codes {
		if code.Name == "Custom: Test" {
			found = true
			if !code.UserDefined || !code.Enabled {
				t.Fatalf("custom = %#v", code)
			}
		}
	}
	if !found {
		t.Fatal("custom code missing")
	}
	if err := Remove(profile, "Custom: Test"); err != nil {
		t.Fatal(err)
	}
	codes, err = Load(profile)
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range codes {
		if code.Name == "Custom: Test" {
			t.Fatal("custom code still present")
		}
	}
}
