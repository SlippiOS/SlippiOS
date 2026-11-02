package dolphin

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SlippiOS/SlippiOS/backend/internal/ini"
)

// Managed describes one Dolphin INI key that maps to a shell setting.
type Managed struct {
	File    string
	Section string
	Key     string
	Setting string
	Kind    string
}

// ManagedValues returns the curated keys for the selected build. The beta
// build stores GFX settings in GFX.ini and Slippi settings in Dolphin.ini;
// Ishiiruka uses Core.SlippiJukeboxEnabled and EFBScale instead.
func ManagedValues(build string) []Managed {
	aspectKind := "aspect"
	resolutionKey := "InternalResolution"
	slippiSection, slippiKey := "Slippi", "EnableJukebox"
	if build == "stable" {
		aspectKind = "stable_aspect"
		resolutionKey = "EFBScale"
		slippiSection, slippiKey = "Core", "SlippiJukeboxEnabled"
	}
	return []Managed{
		{"Dolphin.ini", "Core", "GFXBackend", "backend", "string"},
		{"GFX.ini", "Settings", resolutionKey, "resolution", "int"},
		{"GFX.ini", "Settings", "AspectRatio", "aspect", aspectKind},
		{"GFX.ini", "Hardware", "VSync", "vsync", "bool"},
		{"GFX.ini", "Settings", "MSAA", "msaa", "int"},
		{"Dolphin.ini", "DSP", "Backend", "audio_backend", "string"},
		{"Dolphin.ini", "Core", "AudioLatency", "audio_latency", "int"},
		{"Dolphin.ini", slippiSection, slippiKey, "enable_music", "bool"},
		{"GFX.ini", "Settings", "WaitForShadersBeforeStarting", "precompile_shaders", "bool"},
		{"GFX.ini", "Settings", "ShaderCompilationMode", "shader_mode", "shader_mode"},
		{"GFX.ini", "Settings", "ShowNetPlayPing", "show_netplay_ping", "bool"},
		{"GFX.ini", "Settings", "ShowFPS", "show_fps", "bool"},
	}
}

// ApplyManaged writes the current shell settings into the Dolphin INIs before
// the settings GUI opens, so the GUI starts from the same values.
func ApplyManaged(userDir, build string, values map[string]any) error {
	files := map[string]*ini.File{}
	for _, managed := range ManagedValues(build) {
		path := filepath.Join(userDir, "Config", managed.File)
		file := files[path]
		if file == nil {
			var err error
			file, err = ini.Load(path)
			if err != nil {
				return err
			}
			files[path] = file
		}
		file.Set(managed.Section, managed.Key, managedExpected(managed, values))
	}
	for path, file := range files {
		if err := ini.Save(path, file); err != nil {
			return err
		}
	}
	return nil
}

// SyncManaged updates settings from the Dolphin INIs after the settings GUI
// closes. It only touches curated keys that the GUI wrote.
func SyncManaged(userDir, build string, values map[string]any) (bool, error) {
	dolphin, _ := values["dolphin"].(map[string]any)
	if dolphin == nil {
		dolphin = map[string]any{}
		values["dolphin"] = dolphin
	}
	files := map[string]*ini.File{}
	changed := false
	for _, managed := range ManagedValues(build) {
		path := filepath.Join(userDir, "Config", managed.File)
		file := files[path]
		if file == nil {
			var err error
			file, err = ini.Load(path)
			if err != nil {
				return false, err
			}
			files[path] = file
		}
		raw, ok := file.Value(managed.Section, managed.Key)
		if !ok || raw == "" {
			continue
		}
		next, ok := managedSetting(managed, raw)
		if !ok || sameValue(dolphin[managed.Setting], next) {
			continue
		}
		dolphin[managed.Setting] = next
		changed = true
	}
	return changed, nil
}

func managedExpected(managed Managed, values map[string]any) string {
	dolphin, _ := values["dolphin"].(map[string]any)
	switch managed.Kind {
	case "bool":
		return boolText(flagDefault(dolphin, managed.Setting, managedBoolDefault(managed.Setting)))
	case "int":
		return strconv.FormatInt(number(dolphin, managed.Setting, managedIntDefault(managed.Setting)), 10)
	case "string":
		return text(dolphin, managed.Setting, managedStringDefault(managed.Setting))
	case "shader_mode":
		return strconv.FormatInt(ShaderModeValue(text(dolphin, managed.Setting, "async_uber")), 10)
	case "stable_aspect":
		return strconv.FormatInt(StableAspectValue(number(dolphin, "aspect", 3)), 10)
	default:
		return strconv.FormatInt(number(dolphin, managed.Setting, 3), 10)
	}
}

func managedSetting(managed Managed, raw string) (any, bool) {
	switch managed.Kind {
	case "bool":
		if strings.EqualFold(raw, "True") {
			return true, true
		}
		if strings.EqualFold(raw, "False") {
			return false, true
		}
		return nil, false
	case "int":
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, false
		}
		return value, true
	case "string":
		return raw, true
	case "shader_mode":
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < ShaderModeSynchronous || value > ShaderModeAsynchronousSkip {
			return nil, false
		}
		return ShaderModeName(value), true
	case "stable_aspect":
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, false
		}
		return AspectFromStable(value)
	default:
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 1 || value > 4 {
			return nil, false
		}
		return value, true
	}
}

func sameValue(current, next any) bool {
	switch value := current.(type) {
	case int64:
		switch other := next.(type) {
		case int64:
			return value == other
		case int:
			return value == int64(other)
		}
	case int:
		switch other := next.(type) {
		case int64:
			return int64(value) == other
		case int:
			return value == other
		}
	case bool:
		other, ok := next.(bool)
		return ok && value == other
	case string:
		other, ok := next.(string)
		return ok && value == other
	}
	return false
}

func managedBoolDefault(setting string) bool {
	switch setting {
	case "enable_music", "show_netplay_ping":
		return true
	default:
		return false
	}
}

func managedIntDefault(setting string) int64 {
	switch setting {
	case "resolution":
		return 3
	case "msaa":
		return 1
	case "audio_latency":
		return 20
	default:
		return 0
	}
}

func managedStringDefault(setting string) string {
	switch setting {
	case "backend":
		return "Vulkan"
	case "audio_backend":
		return "Cubeb"
	default:
		return ""
	}
}

func text(values map[string]any, key, fallback string) string {
	value, _ := values[key].(string)
	if value == "" {
		return fallback
	}
	return value
}

func number(values map[string]any, key string, fallback int64) int64 {
	switch value := values[key].(type) {
	case int64:
		return value
	case int:
		return int64(value)
	}
	return fallback
}

func flagDefault(values map[string]any, key string, fallback bool) bool {
	if value, ok := values[key].(bool); ok {
		return value
	}
	return fallback
}

func boolText(value bool) string {
	if value {
		return "True"
	}
	return "False"
}
