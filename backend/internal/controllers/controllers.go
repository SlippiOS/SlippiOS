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

// MatchProfileForPad returns the first profile matching one detected pad.
func MatchProfileForPad(profiles []Profile, pad string) string {
	for _, profile := range profiles {
		if matchesPad(profile.Device, []string{pad}) {
			return profile.Name
		}
	}
	return ProfileCustom
}

// AdapterPresent reports whether a GameCube adapter is connected.
func AdapterPresent() bool {
	entries, err := filepath.Glob("/sys/bus/usb/devices/*")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		vendor, err := os.ReadFile(filepath.Join(entry, "idVendor"))
		if err != nil {
			continue
		}
		product, err := os.ReadFile(filepath.Join(entry, "idProduct"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(vendor)) == "057e" &&
			strings.TrimSpace(string(product)) == "0337" {
			return true
		}
	}
	return false
}

// AutoAssign gives detected standard pads the empty ports when no GameCube
// adapter is present. A console with any manually configured port is left
// alone. It reports whether the settings changed.
func AutoAssign(values map[string]any, dolphinDir string) ([]Port, bool, error) {
	ports := Ports(values)
	if AdapterPresent() {
		return ports, false, nil
	}
	for _, port := range ports {
		if port.Device != DeviceAdapter || port.Profile != "" {
			return ports, false, nil
		}
	}
	pads := DetectedPads()
	if len(pads) == 0 {
		return ports, false, nil
	}
	profiles, err := ListProfiles(dolphinDir)
	if err != nil {
		return ports, false, err
	}
	changed := false
	for index, pad := range pads {
		if index >= 4 {
			break
		}
		profile := MatchProfileForPad(profiles, pad)
		if profile == "" {
			profile = ProfileCustom
		}
		if profile != ProfileCustom {
			if err := ApplyProfile(dolphinDir, index+1, profile); err != nil {
				return Ports(values), changed, err
			}
		}
		if err := SetPort(values, index+1, DevicePad, profile); err != nil {
			return Ports(values), changed, err
		}
		changed = true
	}
	return Ports(values), changed, nil
}

func profileDevice(path string) string {
	if model := profileModel(path); model != "" {
		return model
	}
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

// profileModel reads the optional "# Model = ..." line used to match a
// profile to the controller name reported by the kernel.
func profileModel(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# Model =") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# Model ="))
		}
	}
	return ""
}

// ResolveProfile returns the installed name for a profile, accepting the old
// *_Linux suffix when the unsuffixed profile exists.
func ResolveProfile(dolphinDir, name string) string {
	if name == "" || name == ProfileCustom {
		return name
	}
	dir := ProfilesDir(dolphinDir)
	if _, err := os.Stat(filepath.Join(dir, name+".ini")); err == nil {
		return name
	}
	if strings.HasSuffix(name, "_Linux") {
		base := strings.TrimSuffix(name, "_Linux")
		if _, err := os.Stat(filepath.Join(dir, base+".ini")); err == nil {
			return base
		}
	}
	return name
}

// NormalizeProfiles rewrites profile names that no longer exist (for example
// Glyph_Linux after the Linux-only profile rename). It reports whether the
// settings changed.
func NormalizeProfiles(values map[string]any, dolphinDir string) (bool, error) {
	if err := EnsureDefaults(dolphinDir); err != nil {
		return false, err
	}
	changed := false
	for _, port := range Ports(values) {
		if port.Device != DevicePad || port.Profile == "" || port.Profile == ProfileCustom {
			continue
		}
		resolved := ResolveProfile(dolphinDir, port.Profile)
		if resolved == port.Profile {
			continue
		}
		if err := SetPort(values, port.Port, DevicePad, resolved); err != nil {
			return changed, err
		}
		changed = true
	}
	return changed, nil
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
	name = ResolveProfile(dolphinDir, name)
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

// ConfigureHotkeys binds Dolphin's Stop hotkey to the Guide button on the
// first standard pad. Dolphin's hotkey system only sees controller-interface
// devices, so a GameCube adapter has no equivalent button.
func ConfigureHotkeys(dolphinDir string, values map[string]any) error {
	device := ""
	profiles, _ := ListProfiles(dolphinDir)
	for _, port := range Ports(values) {
		if port.Device != DevicePad {
			continue
		}
		// The applied GCPad section is authoritative: portable profiles have
		// their Linux device filled in when the profile is applied.
		device = padDevice(dolphinDir, port.Port)
		if device == "" && port.Profile != "" && port.Profile != ProfileCustom {
			for _, profile := range profiles {
				if profile.Name == port.Profile {
					device = profile.Device
					break
				}
			}
		}
		if device != "" {
			break
		}
	}
	path := filepath.Join(dolphinDir, "Config", "Hotkeys.ini")
	file, err := ini.Load(path)
	if err != nil {
		return err
	}
	lines := file.Lines("Hotkeys")
	if device == "" {
		// Escape must work even when no standard pad is configured (for
		// example a fresh image whose ports still use the GameCube adapter).
		lines = setLine(lines, "Device", "XInput2/0/Keyboard Mouse")
	} else {
		lines = setLine(lines, "Device", device)
		lines = setLine(lines, "General/Stop", "`Button MODE`")
	}
	// Escape must be an explicit Exit binding. Creating Hotkeys.ini with only
	// Stop drops Dolphin's default Escape -> Exit binding.
	lines = setLine(lines, "General/Exit", "`XInput2/0/Keyboard Mouse:ESCAPE`")
	file.SetLines("Hotkeys", lines)

	// Stable Ishiiruka reads the Hotkeys1 controller group. The GUI writes
	// physical keyboard bindings there, with the virtual core keyboard as the
	// device. Keep both formats so stable and mainline both exit on Escape.
	hotkeys := file.Lines("Hotkeys1")
	hotkeys = setLine(hotkeys, "Device", "XInput2/0/Virtual core pointer")
	if device != "" {
		hotkeys = setLine(hotkeys, "Keys/Stop", fmt.Sprintf("`%s:Button MODE`", device))
	}
	hotkeys = setLine(hotkeys, "Keys/Exit", "Escape")
	file.SetLines("Hotkeys1", hotkeys)
	return ini.Save(path, file)
}

func padDevice(dolphinDir string, port int) string {
	file, err := ini.Load(PadConfigPath(dolphinDir))
	if err != nil {
		return ""
	}
	for _, line := range file.Lines(fmt.Sprintf("GCPad%d", port)) {
		key, value, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(key) == "Device" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func setLine(lines []string, key, value string) []string {
	prefix := key + " ="
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			lines[i] = key + " = " + value
			return lines
		}
	}
	return append(lines, key+" = "+value)
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
