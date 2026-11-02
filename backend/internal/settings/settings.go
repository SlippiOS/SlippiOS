// Package settings owns the persistent profile under /data.
package settings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const SchemaVersion = 1

const DefaultUserJSON = "slippi-user.json"

type Paths struct {
	DataDir          string
	UsersDir         string
	ProfileDir       string
	SettingsFile     string
	ISODir           string
	DolphinDir       string
	DolphinStableDir string
	DolphinUserFile  string
	HardwareFile     string
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
	dolphinStable := filepath.Join(profile, "dolphin-stable")
	return Paths{
		DataDir:          root,
		UsersDir:         users,
		ProfileDir:       profile,
		SettingsFile:     filepath.Join(profile, "settings.toml"),
		ISODir:           filepath.Join(root, "isos"),
		DolphinDir:       dolphin,
		DolphinStableDir: dolphinStable,
		DolphinUserFile:  filepath.Join(dolphin, "Slippi", "user.json"),
		HardwareFile:     filepath.Join(root, "system", "hardware-id"),
	}
}
func (p Paths) BuildDir(build string) string {
	if build == "beta" {
		return p.DolphinDir
	}
	return p.DolphinStableDir
}

// Build returns the selected Dolphin build. Anything unknown falls back to
// the bundled stable build.
func Build(values map[string]any) string {
	dolphin, _ := values["dolphin"].(map[string]any)
	if build, _ := dolphin["build"].(string); build == "beta" {
		return "beta"
	}
	return "stable"
}

func Defaults() map[string]any {
	return map[string]any{
		"schema_version": int64(SchemaVersion),
		"game": map[string]any{
			"iso": "",
		},
		"display": map[string]any{
			"refresh_hz": int64(120),
			"resolution": "auto",
			"rate":       "",
			"custom":     "",
		},
		"audio": map[string]any{
			"sink":   "auto",
			"volume": int64(100),
		},
		"network": map[string]any{
			"ssid": "",
			"psk":  "",
		},
		"dolphin": map[string]any{
			"backend":       "Vulkan",
			"resolution":    int64(3),
			"aspect":        int64(3),
			"vsync":         false,
			"msaa":          int64(1),
			"audio_backend": "Cubeb",
			"audio_latency": int64(20),
			"build":         "stable",
			"controller":    "adapter",
			"profile":       "",
			"controller1":   "adapter",
			"controller2":   "adapter",
			"controller3":   "adapter",
			"controller4":   "adapter",
			"profile1":      "",
			"profile2":      "",
			"profile3":      "",
			"profile4":      "",
		},
		"expert": map[string]any{
			"enabled": false,
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

// Ensure loads settings, creating a default file when it is missing.
func (p Paths) Ensure() (map[string]any, error) {
	values, err := p.Load()
	if err == nil {
		return values, nil
	}
	if _, statErr := os.Stat(p.SettingsFile); statErr == nil {
		return nil, err
	}
	values = Defaults()
	if err := p.Save(values); err != nil {
		return nil, err
	}
	return values, nil
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
	migrateControllers(decoded, merged)
	return merged, nil
}

// Older settings had one controller setting for all four ports. Keep the
// original behaviour for port 1 and leave the other ports on the adapter.
func migrateControllers(decoded, merged map[string]any) {
	rawDolphin, _ := decoded["dolphin"].(map[string]any)
	if rawDolphin == nil {
		return
	}
	mergedDolphin, _ := merged["dolphin"].(map[string]any)
	if mergedDolphin == nil {
		return
	}
	if _, exists := rawDolphin["controller1"]; !exists {
		if legacy, _ := rawDolphin["controller"].(string); legacy != "" {
			mergedDolphin["controller1"] = legacy
		}
	}
	if _, exists := rawDolphin["profile1"]; !exists {
		if legacy, _ := rawDolphin["profile"].(string); legacy != "" {
			mergedDolphin["profile1"] = legacy
		}
	}
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
