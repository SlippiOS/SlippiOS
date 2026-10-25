// Package settings owns the persistent profile under /data.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const SchemaVersion = 1

const DefaultUserJSON = "slippi-user.json"

var meleeID = regexp.MustCompile(`^GAL[A-Z0-9]01`)

var hardwareIDPaths = []string{
	"/sys/class/dmi/id/product_uuid",
	"/sys/class/dmi/id/board_serial",
}

var placeholderHardwareIDs = map[string]bool{
	"":                                     true,
	"none":                                 true,
	"not specified":                        true,
	"to be filled by o.e.m.":               true,
	"03000200-0400-0500-0006-000700080009": true,
}

// Paths describes the data layout. Tests use a temporary root.
type Paths struct {
	DataDir         string
	UsersDir        string
	ProfileDir      string
	SettingsFile    string
	ISODir          string
	DolphinDir      string
	DolphinUserFile string
	HardwareFile    string
}

func DefaultPaths() Paths {
	root := os.Getenv("SLIPPIOS_DATA")
	if root == "" {
		root = "/data"
	}
	return PathsFor(root)
}

func PathsFor(root string) Paths {
	users := filepath.Join(root, "users")
	profile := filepath.Join(users, "default")
	dolphin := filepath.Join(profile, "dolphin")
	return Paths{
		DataDir:         root,
		UsersDir:        users,
		ProfileDir:      profile,
		SettingsFile:    filepath.Join(profile, "settings.toml"),
		ISODir:          filepath.Join(root, "isos"),
		DolphinDir:      dolphin,
		DolphinUserFile: filepath.Join(dolphin, "Slippi", "user.json"),
		HardwareFile:    filepath.Join(root, "system", "hardware-id"),
	}
}

func Defaults() map[string]any {
	return map[string]any{
		"schema_version": int64(SchemaVersion),
		"game": map[string]any{
			"iso": "melee.iso",
		},
		"display": map[string]any{
			"refresh_hz": int64(120),
			"resolution": "auto",
		},
		"audio": map[string]any{
			"sink": "auto",
		},
		"network": map[string]any{
			"ssid": "",
			"psk":  "",
		},
		"slippi": map[string]any{
			"user_json": DefaultUserJSON,
		},
	}
}

func merge(dst, src map[string]any) {
	for key, value := range src {
		if nested, ok := value.(map[string]any); ok {
			existing, _ := dst[key].(map[string]any)
			if existing == nil {
				existing = map[string]any{}
				dst[key] = existing
			}
			merge(existing, nested)
			continue
		}
		dst[key] = value
	}
}

func (p Paths) Load() (map[string]any, error) {
	raw, err := os.ReadFile(p.SettingsFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("missing %s", p.SettingsFile)
		}
		return nil, fmt.Errorf("cannot read %s: %w", p.SettingsFile, err)
	}
	var decoded map[string]any
	if err := toml.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", p.SettingsFile, err)
	}
	if version, ok := decoded["schema_version"].(int64); !ok || version != SchemaVersion {
		return nil, fmt.Errorf("unsupported schema_version %v", decoded["schema_version"])
	}
	merged := Defaults()
	merge(merged, decoded)
	return merged, nil
}

func (p Paths) Save(values map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(p.SettingsFile), 0o755); err != nil {
		return err
	}
	data, err := toml.Marshal(values)
	if err != nil {
		return err
	}
	return os.WriteFile(p.SettingsFile, data, 0o644)
}

func (p Paths) Resolve(value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(p.ProfileDir, value)
}

func (p Paths) ISOPath(value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(p.ISODir, value)
}

func (p Paths) ISOStatus(value string) (bool, string) {
	iso := p.ISOPath(value)
	file, err := os.Open(iso)
	if err != nil {
		return false, fmt.Sprintf("not found: %s", iso)
	}
	defer file.Close()
	header := make([]byte, 6)
	if _, err := file.Read(header); err != nil {
		return false, fmt.Sprintf("cannot read %s: %v", iso, err)
	}
	if !meleeID.Match(header) {
		return false, fmt.Sprintf("%s is not a Melee image (game id %q)", iso, header)
	}
	return true, "ok"
}

type ISO struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Valid  bool   `json:"valid"`
	Reason string `json:"reason"`
}

func (p Paths) SearchDirs() []string {
	dirs := []string{p.ISODir}
	for _, base := range []string{"/run/media", "/media", "/mnt"} {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				dirs = append(dirs, filepath.Join(base, entry.Name()))
			}
		}
		// /run/media/<user>/<volume> is also common.
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			nested, err := os.ReadDir(filepath.Join(base, entry.Name()))
			if err != nil {
				continue
			}
			for _, child := range nested {
				if child.IsDir() {
					dirs = append(dirs, filepath.Join(base, entry.Name(), child.Name()))
				}
			}
		}
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if !seen[dir] {
			seen[dir] = true
			result = append(result, dir)
		}
	}
	return result
}

func (p Paths) FindISOs() []ISO {
	return p.FindISOsIn(p.SearchDirs())
}

func (p Paths) FindISOsIn(dirs []string) []ISO {
	seen := map[string]bool{}
	result := []ISO{}
	for _, root := range dirs {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".iso") {
				return nil
			}
			if seen[path] {
				return nil
			}
			seen[path] = true
			valid, reason := p.ISOStatus(path)
			result = append(result, ISO{Path: path, Name: filepath.Base(path), Valid: valid, Reason: reason})
			return nil
		})
		if err != nil {
			continue
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func (p Paths) LoginStatus(path string) (bool, string) {
	candidate := path
	if !filepath.IsAbs(candidate) {
		candidate = p.Resolve(candidate)
	}
	data, err := os.ReadFile(candidate)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, fmt.Sprintf("not found: %s", candidate)
		}
		return false, fmt.Sprintf("cannot read %s: %v", candidate, err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return false, fmt.Sprintf("cannot read %s: %v", candidate, err)
	}
	for _, key := range []string{"uid", "playKey", "connectCode"} {
		if value[key] == nil || value[key] == "" {
			return false, fmt.Sprintf("%s is missing %s", candidate, key)
		}
	}
	return true, fmt.Sprint(value["connectCode"])
}

// HardwareID returns the best-effort stable machine identity (DMI product UUID).
func HardwareID() (string, bool) {
	for _, path := range hardwareIDPaths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		value := strings.TrimSpace(string(raw))
		if !placeholderHardwareIDs[strings.ToLower(value)] {
			return value, true
		}
	}
	return "", false
}

// CheckHardwareBinding drops a persisted login when the cartridge moved to
// different hardware. It records the current machine on first use.
func (p Paths) CheckHardwareBinding() bool {
	current, ok := HardwareID()
	if !ok {
		return false
	}
	stored, err := os.ReadFile(p.HardwareFile)
	previous := strings.TrimSpace(string(stored))
	if err != nil || previous == "" {
		if err := os.MkdirAll(filepath.Dir(p.HardwareFile), 0o755); err == nil {
			_ = os.WriteFile(p.HardwareFile, []byte(current+"\n"), 0o644)
		}
		return false
	}
	if previous == current {
		return false
	}
	_ = os.Remove(filepath.Join(p.ProfileDir, DefaultUserJSON))
	_ = os.Remove(p.DolphinUserFile)
	_ = os.Remove(p.HardwareFile)
	return true
}

// InstallLogin copies an imported user.json into Dolphin's user directory.
func (p Paths) InstallLogin(values map[string]any) (bool, string) {
	configured := DefaultUserJSON
	if slippi, ok := values["slippi"].(map[string]any); ok {
		if value, ok := slippi["user_json"].(string); ok && value != "" {
			configured = value
		}
	}
	candidates := []string{
		p.Resolve(configured),
		filepath.Join(p.ProfileDir, "user.json"),
		filepath.Join(p.DataDir, "user.json"),
	}
	for _, candidate := range candidates {
		ok, detail := p.LoginStatus(candidate)
		if !ok {
			continue
		}
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p.DolphinUserFile), 0o755); err != nil {
			return false, fmt.Sprintf("cannot create %s: %v", filepath.Dir(p.DolphinUserFile), err)
		}
		if err := os.WriteFile(p.DolphinUserFile, data, 0o600); err != nil {
			return false, fmt.Sprintf("cannot write %s: %v", p.DolphinUserFile, err)
		}
		_ = os.Chmod(p.DolphinUserFile, 0o600)
		if current, ok := HardwareID(); ok {
			if err := os.MkdirAll(filepath.Dir(p.HardwareFile), 0o755); err == nil {
				_ = os.WriteFile(p.HardwareFile, []byte(current+"\n"), 0o644)
			}
		}
		return true, detail
	}
	return false, fmt.Sprintf("no Slippi login file at %s", p.Resolve(configured))
}
