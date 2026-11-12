package dolphin

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallQuitTexture(t *testing.T) {
	userDir := t.TempDir()
	if err := InstallQuitTexture(userDir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(userDir, "Load", "Textures", "GALE01", "SlippiOS", quitTextureName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, quitTexture) {
		t.Fatal("installed texture differs from the embedded texture")
	}
	if err := InstallQuitTexture(userDir); err != nil {
		t.Fatal(err)
	}
}
