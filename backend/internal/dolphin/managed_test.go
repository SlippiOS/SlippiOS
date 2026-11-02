package dolphin

import (
	"path/filepath"
	"testing"

	"github.com/SlippiOS/SlippiOS/backend/internal/ini"
)

func TestApplyAndSyncManagedBeta(t *testing.T) {
	dir := t.TempDir()
	values := map[string]any{"dolphin": map[string]any{
		"backend":            "Vulkan",
		"resolution":         int64(3),
		"aspect":             int64(3),
		"vsync":              false,
		"msaa":               int64(1),
		"audio_backend":      "Cubeb",
		"audio_latency":      int64(20),
		"enable_music":       true,
		"precompile_shaders": false,
		"shader_mode":        "async_uber",
		"show_netplay_ping":  true,
		"show_fps":           false,
	}}
	if err := ApplyManaged(dir, "beta", values); err != nil {
		t.Fatal(err)
	}

	dolphinPath := filepath.Join(dir, "Config", "Dolphin.ini")
	dolphinINI, err := ini.Load(dolphinPath)
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := dolphinINI.Value("Slippi", "EnableJukebox"); !ok || value != "True" {
		t.Fatalf("EnableJukebox = %q, %v", value, ok)
	}

	gfxPath := filepath.Join(dir, "Config", "GFX.ini")
	gfx, err := ini.Load(gfxPath)
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := gfx.Value("Settings", "ShowFPS"); !ok || value != "False" {
		t.Fatalf("ShowFPS = %q, %v", value, ok)
	}
	if value, ok := gfx.Value("Settings", "ShaderCompilationMode"); !ok || value != "2" {
		t.Fatalf("ShaderCompilationMode = %q, %v", value, ok)
	}

	dolphinINI.Set("Slippi", "EnableJukebox", "False")
	if err := ini.Save(dolphinPath, dolphinINI); err != nil {
		t.Fatal(err)
	}
	gfx.Set("Settings", "ShowFPS", "True")
	gfx.Set("Settings", "ShaderCompilationMode", "0")
	gfx.Set("Settings", "WaitForShadersBeforeStarting", "True")
	gfx.Set("Settings", "ShowNetPlayPing", "False")
	if err := ini.Save(gfxPath, gfx); err != nil {
		t.Fatal(err)
	}

	changed, err := SyncManaged(dir, "beta", values)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected settings to change")
	}
	dolphinSettings := values["dolphin"].(map[string]any)
	if dolphinSettings["enable_music"] != false {
		t.Fatalf("enable_music = %#v", dolphinSettings["enable_music"])
	}
	if dolphinSettings["show_fps"] != true {
		t.Fatalf("show_fps = %#v", dolphinSettings["show_fps"])
	}
	if dolphinSettings["shader_mode"] != "sync" {
		t.Fatalf("shader_mode = %#v", dolphinSettings["shader_mode"])
	}
	if dolphinSettings["precompile_shaders"] != true {
		t.Fatalf("precompile_shaders = %#v", dolphinSettings["precompile_shaders"])
	}
	if dolphinSettings["show_netplay_ping"] != false {
		t.Fatalf("show_netplay_ping = %#v", dolphinSettings["show_netplay_ping"])
	}
}

func TestSyncManagedStableAspect(t *testing.T) {
	dir := t.TempDir()
	values := map[string]any{"dolphin": map[string]any{"aspect": int64(3)}}
	if err := ApplyManaged(dir, "stable", values); err != nil {
		t.Fatal(err)
	}
	gfxPath := filepath.Join(dir, "Config", "GFX.ini")
	gfx, err := ini.Load(gfxPath)
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := gfx.Value("Settings", "AspectRatio"); !ok || value != "5" {
		t.Fatalf("AspectRatio = %q, %v", value, ok)
	}
	gfx.Set("Settings", "AspectRatio", "3")
	if err := ini.Save(gfxPath, gfx); err != nil {
		t.Fatal(err)
	}
	changed, err := SyncManaged(dir, "stable", values)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected aspect to change")
	}
	if got := values["dolphin"].(map[string]any)["aspect"]; got != int64(4) {
		t.Fatalf("aspect = %#v", got)
	}
}
