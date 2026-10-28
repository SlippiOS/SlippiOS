// Package controllers manages Dolphin's four emulated GameCube controller
// ports and the GCPad profiles assigned to them.
package controllers

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SlippiOS/SlippiOS/backend/internal/ini"
)

const (
	DeviceAdapter  = "adapter"
	DevicePad      = "pad"
	DeviceKeyboard = "keyboard"
	DeviceNone     = "none"
	ProfileCustom  = "custom"
)

//go:embed profiles/*.ini
var builtinProfiles embed.FS

type Port struct {
	Port    int    `json:"port"`
	Device  string `json:"device"`
	Profile string `json:"profile"`
}

type Profile struct {
	Name      string `json:"name"`
	Device    string `json:"device"`
	Connected bool   `json:"connected"`
}

func ProfilesDir(dolphinDir string) string {
	return filepath.Join(dolphinDir, "Config", "Profiles", "GCPad")
}

func PadConfigPath(dolphinDir string) string {
	return filepath.Join(dolphinDir, "Config", "GCPadNew.ini")
}

func ValidDevice(device string) bool {
	switch device {
	case DeviceAdapter, DevicePad, DeviceKeyboard, DeviceNone:
		return true
	}
	return false
}

func Ports(values map[string]any) []Port {
	dolphin, _ := values["dolphin"].(map[string]any)
	ports := make([]Port, 0, 4)
	for port := 1; port <= 4; port++ {
		device := text(dolphin, fmt.Sprintf("controller%d", port), DeviceAdapter)
		profile := text(dolphin, fmt.Sprintf("profile%d", port), "")
		ports = append(ports, Port{Port: port, Device: device, Profile: profile})
	}
	return ports
}

func SetPort(values map[string]any, port int, device, profile string) error {
	if port < 1 || port > 4 {
		return fmt.Errorf("port must be between 1 and 4")
	}
	if !ValidDevice(device) {
		return fmt.Errorf("unknown input %q", device)
	}
	dolphin, _ := values["dolphin"].(map[string]any)
	if dolphin == nil {
		return fmt.Errorf("settings has no dolphin section")
	}
	dolphin[fmt.Sprintf("controller%d", port)] = device
	if device == DevicePad {
		dolphin[fmt.Sprintf("profile%d", port)] = profile
	} else {
		dolphin[fmt.Sprintf("profile%d", port)] = ""
	}
	return nil
}

// EnsureDefaults writes the profiles shipped with SlippiOS when they are not
// already present. Existing files are never overwritten.
func EnsureDefaults(dolphinDir string) error {
	entries, err := builtinProfiles.ReadDir("profiles")
	if err != nil {
		return err
	}
	dir := ProfilesDir(dolphinDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ini") {
			continue
		}
		target := filepath.Join(dir, entry.Name())
		if _, err := os.Stat(target); err == nil {
			continue
		}
		data, err := builtinProfiles.ReadFile("profiles/" + entry.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func ListProfiles(dolphinDir string) ([]Profile, error) {
	if err := EnsureDefaults(dolphinDir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(ProfilesDir(dolphinDir))
	if err != nil {
		return nil, err
	}
	pads := DetectedPads()
	profiles := make([]Profile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ini") {
			continue
		}
		path := filepath.Join(ProfilesDir(dolphinDir), entry.Name())
		device := profileDevice(path)
		profiles = append(profiles, Profile{
			Name:      strings.TrimSuffix(entry.Name(), ".ini"),
			Device:    device,
			Connected: matchesPad(device, pads),
		})
	}
	sort.Slice(profiles, func(i, j int) bool {
		return strings.ToLower(profiles[i].Name) < strings.ToLower(profiles[j].Name)
	})
	return profiles, nil
}

// DetectedPads returns the names of connected joysticks as reported by the
// kernel. Dolphin's evdev profiles use the same names.
func DetectedPads() []string {
	matches, _ := filepath.Glob("/sys/class/input/js*/device/name")
	pads := make([]string, 0, len(matches))
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(data))
		if name != "" {
			pads = append(pads, name)
		}
	}
	return pads
}

func matchesPad(device string, pads []string) bool {
	for _, pad := range pads {
		if strings.Contains(device, pad) {
			return true
		}
	}
	return false
}

// MatchProfile returns the first profile whose evdev device matches a
// connected pad, or ProfileCustom when nothing matches.
func MatchProfile(profiles []Profile, pads []string) string {
	for _, profile := range profiles {
		if matchesPad(profile.Device, pads) {
			return profile.Name
		}
	}
	return ProfileCustom
}

func profileDevice(path string) string {
	file, err := ini.Load(path)
	if err != nil {
		return ""
	}
	for _, line := range file.Lines("Profile") {
		key, value, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(key) == "Device" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// ApplyProfile copies a GCPad profile into the matching [GCPadN] section.
// ProfileCustom leaves the existing section alone.
func ApplyProfile(dolphinDir string, port int, name string) error {
	if port < 1 || port > 4 {
		return fmt.Errorf("port must be between 1 and 4")
	}
	if name == "" || name == ProfileCustom {
		return nil
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return fmt.Errorf("invalid profile name")
	}
	if err := EnsureDefaults(dolphinDir); err != nil {
		return err
	}
	profilePath := filepath.Join(ProfilesDir(dolphinDir), name+".ini")
	profile, err := ini.Load(profilePath)
	if err != nil {
		return err
	}
	lines := profile.Lines("Profile")
	if len(lines) == 0 {
		return fmt.Errorf("profile %q has no [Profile] section", name)
	}
	padPath := PadConfigPath(dolphinDir)
	pad, err := ini.Load(padPath)
	if err != nil {
		return err
	}
	pad.SetLines(fmt.Sprintf("GCPad%d", port), lines)
	return ini.Save(padPath, pad)
}

// ApplyConfigured applies the profile selected for every pad port. Adapter,
// keyboard and disabled ports are left untouched.
func ApplyConfigured(dolphinDir string, values map[string]any) error {
	for _, port := range Ports(values) {
		if port.Device != DevicePad {
			continue
		}
		if err := ApplyProfile(dolphinDir, port.Port, port.Profile); err != nil {
			return err
		}
	}
	return nil
}

func text(values map[string]any, key, fallback string) string {
	value, _ := values[key].(string)
	if value == "" {
		return fallback
	}
	return value
}
