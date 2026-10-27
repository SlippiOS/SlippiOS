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
			"iso": "",
		},
		"display": map[string]any{
			"refresh_hz": int64(120),
			"resolution": "auto",
			"rate":       "",
			"custom":     "",
		},
		"audio": map[string]any{
			"sink": "auto",
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
			"controller":    "adapter",
			"profile":       "",
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
