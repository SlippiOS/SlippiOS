package controllers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SlippiOS/SlippiOS/backend/internal/ini"
)

func TestDefaultsAndProfiles(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureDefaults(dir); err != nil {
		t.Fatal(err)
	}
	profiles, err := ListProfiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, profile := range profiles {
		names[profile.Name] = true
	}
	for _, want := range []string{"Keyboard", "Xbox 360 (Microsoft)"} {
		if !names[want] {
			t.Fatalf("missing profile %q in %#v", want, names)
		}
	}
}

func TestApplyProfileKeepsOtherPorts(t *testing.T) {
	dir := t.TempDir()
	path := PadConfigPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte("[GCPad1]\nDevice = evdev/0/Other\nButtons/A = `Button 9`\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ApplyProfile(dir, 2, "Xbox 360 (Microsoft)"); err != nil {
		t.Fatal(err)
	}
	file, err := ini.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := file.Lines("GCPad1"); len(got) == 0 || !strings.Contains(got[0], "Other") {
		t.Fatalf("port 1 changed: %#v", got)
	}
	port2 := file.Lines("GCPad2")
	if len(port2) == 0 || port2[0] != "Device = evdev/0/Microsoft X-Box 360 pad" {
		t.Fatalf("port 2 = %#v", port2)
	}
	if !strings.Contains(strings.Join(port2, "\n"), "Buttons/A = `Button 0`") {
		t.Fatalf("port 2 buttons = %#v", port2)
	}
}

func TestPortsAndConfigured(t *testing.T) {
	dir := t.TempDir()
	values := map[string]any{"dolphin": map[string]any{}}
	if err := SetPort(values, 1, DevicePad, "Keyboard"); err != nil {
		t.Fatal(err)
	}
	if err := SetPort(values, 2, DeviceAdapter, ""); err != nil {
		t.Fatal(err)
	}
	ports := Ports(values)
	if ports[0].Device != DevicePad || ports[0].Profile != "Keyboard" {
		t.Fatalf("port 1 = %#v", ports[0])
	}
	if ports[1].Device != DeviceAdapter || ports[1].Profile != "" {
		t.Fatalf("port 2 = %#v", ports[1])
	}
	if err := ApplyConfigured(dir, values); err != nil {
		t.Fatal(err)
	}
	file, err := ini.Load(PadConfigPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if got := file.Lines("GCPad1"); len(got) == 0 || !strings.Contains(got[0], "Keyboard Mouse") {
		t.Fatalf("keyboard profile not applied: %#v", got)
	}
	if got := file.Lines("GCPad2"); len(got) != 0 {
		t.Fatalf("adapter port wrote a profile: %#v", got)
	}
}

func TestConfigureHotkeys(t *testing.T) {
	dir := t.TempDir()
	values := map[string]any{"dolphin": map[string]any{}}
	if err := SetPort(values, 1, DevicePad, "Xbox 360 (Microsoft)"); err != nil {
		t.Fatal(err)
	}
	if err := ApplyProfile(dir, 1, "Xbox 360 (Microsoft)"); err != nil {
		t.Fatal(err)
	}
	if err := ConfigureHotkeys(dir, values); err != nil {
		t.Fatal(err)
	}
	file, err := ini.Load(filepath.Join(dir, "Config", "Hotkeys.ini"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Join(file.Lines("Hotkeys"), "\n")
	if !strings.Contains(lines, "Device = evdev/0/Microsoft X-Box 360 pad") {
		t.Fatalf("hotkeys = %s", lines)
	}
	if !strings.Contains(lines, "General/Stop = `Button MODE`") {
		t.Fatalf("hotkeys = %s", lines)
	}
	if !strings.Contains(lines, "General/Exit = `XInput2/0/Keyboard Mouse:ESCAPE`") {
		t.Fatalf("hotkeys missing Escape exit: %s", lines)
	}
}

func TestResolveProfileDropsLegacySuffix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(ProfilesDir(dir), "Glyph.ini")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[Profile]\nDevice = evdev/0/Pad\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ResolveProfile(dir, "Glyph_Linux"); got != "Glyph" {
		t.Fatalf("ResolveProfile = %q", got)
	}
	if got := ResolveProfile(dir, "Glyph"); got != "Glyph" {
		t.Fatalf("ResolveProfile = %q", got)
	}
}

func TestNormalizeProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(ProfilesDir(dir), "Glyph.ini")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[Profile]\nDevice = evdev/0/Pad\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	values := map[string]any{"dolphin": map[string]any{}}
	if err := SetPort(values, 1, DevicePad, "Glyph_Linux"); err != nil {
		t.Fatal(err)
	}
	changed, err := NormalizeProfiles(values, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || Ports(values)[0].Profile != "Glyph" {
		t.Fatalf("changed = %v, ports = %#v", changed, Ports(values))
	}
}

func TestMatchProfileForPad(t *testing.T) {
	profiles := []Profile{
		{Name: "Other", Device: "evdev/0/Other"},
		{Name: "Glyph", Device: "Mad Catz Xbox 360 Controller"},
	}
	if got := MatchProfileForPad(profiles, "Mad Catz Xbox 360 Controller"); got != "Glyph" {
		t.Fatalf("MatchProfileForPad = %q", got)
	}
}

func TestProfileModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Glyph.ini")
	data := "# Model = Mad Catz Xbox 360 Controller\n[Profile]\nDevice = evdev/0/Mad Catz Xbox 360 Controller\nButtons/A = `Button 0`\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := profileDevice(path); got != "Mad Catz Xbox 360 Controller" {
		t.Fatalf("profileDevice = %q", got)
	}
}
