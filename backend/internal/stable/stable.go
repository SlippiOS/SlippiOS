// Package stable writes the Ishiiruka/FM-Slippi configuration. The stable
// build predates Mainline's -C config layer, so its INI files are written
// directly.
package stable

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/SlippiOS/SlippiOS/backend/internal/controllers"
	dolphinpkg "github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
	"github.com/SlippiOS/SlippiOS/backend/internal/ini"
)

func Configure(profileDir, stableDir, isoDir string, values map[string]any, width, height int, fullscreen bool) error {
	dolphin, _ := values["dolphin"].(map[string]any)
	backend := text(dolphin, "backend", "Vulkan")
	resolution := number(dolphin, "resolution", 3)
	aspect := number(dolphin, "aspect", 3)
	msaa := number(dolphin, "msaa", 1)
	vsync := flag(dolphin, "vsync")
	enableMusic := flagDefault(dolphin, "enable_music", true)
	showFPS := flag(dolphin, "show_fps")
	showPing := flagDefault(dolphin, "show_netplay_ping", true)
	precompile := flag(dolphin, "precompile_shaders")
	shaderMode := dolphinpkg.ShaderModeValue(text(dolphin, "shader_mode", "async_uber"))

	dolphinINI, err := ini.Load(filepath.Join(stableDir, "Config", "Dolphin.ini"))
	if err != nil {
		return err
	}
	dolphinINI.Set("General", "ISOPaths", "1")
	dolphinINI.Set("General", "ISOPath0", isoDir)
	dolphinINI.Set("Interface", "HideCursor", "True")
	dolphinINI.Set("Interface", "ConfirmStop", "False")
	dolphinINI.Set("Display", "Fullscreen", boolText(fullscreen))
	dolphinINI.Set("Display", "FullscreenResolution", "Auto")
	dolphinINI.Set("Display", "RenderWindowAutoSize", "False")
	dolphinINI.Set("Display", "RenderWindowWidth", fmt.Sprintf("%d", width))
	dolphinINI.Set("Display", "RenderWindowHeight", fmt.Sprintf("%d", height))
	// Keep the render window off-screen while the settings launcher is open.
	// Otherwise it maps a white frame at the top-left and leaves it behind.
	if fullscreen {
		dolphinINI.Set("Display", "RenderWindowXPos", "0")
		dolphinINI.Set("Display", "RenderWindowYPos", "0")
	} else {
		dolphinINI.Set("Display", "RenderWindowXPos", "-10000")
		dolphinINI.Set("Display", "RenderWindowYPos", "-10000")
	}
	dolphinINI.Set("Core", "CPUThread", "True")
	dolphinINI.Set("Core", "GFXBackend", backend)
	dolphinINI.Set("Core", "SlippiReplayDir", filepath.Join(profileDir, "replays"))
	dolphinINI.Set("Core", "SlippiSaveReplays", "True")
	dolphinINI.Set("Core", "SlippiReplayMonthFolders", "False")
	dolphinINI.Set("Core", "SlippiJukeboxEnabled", boolText(enableMusic))
	dolphinINI.Set("Core", "SlippiOnlineDelay", "2")
	for port := 0; port < 4; port++ {
		dolphinINI.Set("Core", fmt.Sprintf("SIDevice%d", port), fmt.Sprintf("%d", siValue(values, port+1)))
	}
	dolphinINI.Set("DSP", "Backend", "ALSA")
	dolphinINI.Set("DSP", "Volume", "100")
	if err := ini.Save(filepath.Join(stableDir, "Config", "Dolphin.ini"), dolphinINI); err != nil {
		return err
	}

	gfx, err := ini.Load(filepath.Join(stableDir, "Config", "GFX.ini"))
	if err != nil {
		return err
	}
	gfx.Set("Hardware", "VSync", boolText(vsync))
	gfx.Set("Settings", "AspectRatio", strconv.FormatInt(dolphinpkg.StableAspectValue(aspect), 10))
	gfx.Set("Settings", "BorderlessFullscreen", boolText(fullscreen))
	gfx.Set("Settings", "EFBScale", strconv.FormatInt(resolution, 10))
	gfx.Set("Settings", "MSAA", strconv.FormatInt(msaa, 10))
	gfx.Set("Settings", "ShowFPS", boolText(showFPS))
	gfx.Set("Settings", "ShowNetPlayPing", boolText(showPing))
	gfx.Set("Settings", "WaitForShadersBeforeStarting", boolText(precompile))
	gfx.Set("Settings", "ShaderCompilationMode", strconv.FormatInt(shaderMode, 10))
	if err := ini.Save(filepath.Join(stableDir, "Config", "GFX.ini"), gfx); err != nil {
		return err
	}

	if err := controllers.EnsureDefaults(stableDir); err != nil {
		return err
	}
	for _, port := range controllers.Ports(values) {
		if port.Device != controllers.DevicePad {
			continue
		}
		if err := controllers.ApplyProfile(stableDir, port.Port, port.Profile); err != nil {
			return err
		}
	}
	return controllers.ConfigureHotkeys(stableDir, values)
}

func siValue(values map[string]any, port int) int {
	dolphin, _ := values["dolphin"].(map[string]any)
	switch text(dolphin, fmt.Sprintf("controller%d", port), "adapter") {
	case "pad":
		return 6
	case "keyboard":
		return 7
	case "none":
		return 0
	default:
		return 12
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

func flag(values map[string]any, key string) bool {
	value, _ := values[key].(bool)
	return value
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
