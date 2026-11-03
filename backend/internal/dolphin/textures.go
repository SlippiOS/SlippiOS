package dolphin

import (
	"bytes"
	_ "embed"
	"os"
	"path/filepath"

	"github.com/SlippiOS/SlippiOS/backend/internal/ini"
)

// The main menu's Data label, redrawn as Quit. Dolphin matches custom
// textures by the dumped texture name, so the filename must not change.
//
//go:embed assets/tex1_176x30_d1fa83d25e26b8a9_2.png
var quitTexture []byte

const quitTextureName = "tex1_176x30_d1fa83d25e26b8a9_2.png"

// InstallQuitTexture writes the Quit label into the build's custom texture
// directory. The caller also has to enable Dolphin's HiresTextures setting.
func InstallQuitTexture(userDir string) error {
	dir := filepath.Join(userDir, "Load", "Textures", "GALE01", "SlippiOS")
	path := filepath.Join(dir, quitTextureName)
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, quitTexture) {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, quitTexture, 0o644)
}

// EnableQuitSupport installs the Quit label and turns on the settings the
// launcher needs: custom textures for the label and the Slippi log for the
// quit marker.
func EnableQuitSupport(userDir string) error {
	if err := InstallQuitTexture(userDir); err != nil {
		return err
	}
	logger, err := ini.Load(filepath.Join(userDir, "Config", "Logger.ini"))
	if err != nil {
		return err
	}
	logger.Set("Options", "WriteToConsole", "True")
	logger.Set("Options", "WriteToFile", "True")
	logger.Set("Options", "Verbosity", "4")
	logger.Set("Logs", "SLIPPI", "True")
	if err := ini.Save(filepath.Join(userDir, "Config", "Logger.ini"), logger); err != nil {
		return err
	}
	gfx, err := ini.Load(filepath.Join(userDir, "Config", "GFX.ini"))
	if err != nil {
		return err
	}
	gfx.Set("Settings", "HiresTextures", "True")
	gfx.Set("Settings", "CacheHiresTextures", "True")
	return ini.Save(filepath.Join(userDir, "Config", "GFX.ini"), gfx)
}
