// Package dolphin fetches and verifies the newest Slippi Mainline Dolphin
// release. A failed check or download leaves the current build usable.
package dolphin

import (
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const (
	assetName   = "Slippi_Netplay_Mainline-x86_64.AppImage"
	userAgent   = "SlippiOS/0.1"
	checkPeriod = 15 * time.Minute
)

var (
	// API lists recent upstream releases. Tests replace it with a local server.
	API = "https://api.github.com/repos/project-slippi/dolphin/releases?per_page=10"

	tagPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

// Paths describes the on-disk Dolphin installation.
type Paths struct {
	Base     string
	Versions string
	Current  string
	State    string
	Lock     string
}

func DefaultPaths() Paths {
	root := os.Getenv("SLIPPIOS_DATA")
	if root == "" {
		root = "/data"
	}
	return ForData(root)
}

func ForData(root string) Paths {
	base := filepath.Join(root, "system", "dolphin")
	return Paths{
		Base:     base,
		Versions: filepath.Join(base, "versions"),
		Current:  filepath.Join(base, "current"),
		State:    filepath.Join(base, "state.json"),
		Lock:     filepath.Join(base, "lock"),
	}
}
