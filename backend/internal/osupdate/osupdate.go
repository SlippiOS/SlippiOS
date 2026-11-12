// Package osupdate stages and installs SlippiOS system updates.
package osupdate

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

const (
	defaultBase     = "https://pub-ac47e58c86e44a728518e30a6eb756da.r2.dev/"
	defaultManifest = defaultBase + "manifest.json"
	devBase         = defaultBase + "dev/"
	devManifest     = devBase + "manifest.json"
	downloadBase    = "https://github.com/SlippiOS/SlippiOS/releases/download"
	userAgent       = "SlippiOS/0.2"
)

var (
	sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	procMounts    = "/proc/mounts"
	bootIDFile    = "/proc/sys/kernel/random/boot_id"
)

type Paths struct {
	DataDir     string
	StageDir    string
	StatusFile  string
	Progress    string
	Latest      string
	Request     string
	Reboot      string
	Source      string
	VersionFile string
}

func DefaultPaths() Paths {
	root := os.Getenv("SLIPPIOS_DATA")
	if root == "" {
		root = "/data"
	}
	paths := ForData(root)
	// The progress file is updated for every download chunk. Keeping it on
	// the USB drive makes the update write-bound; /run is a tmpfs.
	runtime := filepath.Join("/run/user", strconv.Itoa(os.Getuid()))
	if info, err := os.Stat(runtime); err == nil && info.IsDir() {
		paths.Progress = filepath.Join(runtime, "slippios-update-progress.json")
	}
	return paths
}

func ForData(root string) Paths {
	stage := filepath.Join(root, "system", "update")
	return Paths{
		DataDir:     root,
		StageDir:    stage,
		StatusFile:  filepath.Join(stage, "status.json"),
		Progress:    filepath.Join(stage, "progress.json"),
		Latest:      filepath.Join(stage, "latest.json"),
		Request:     filepath.Join(stage, "request"),
		Reboot:      filepath.Join(stage, "reboot"),
		Source:      filepath.Join(root, "system", "update-source.json"),
		VersionFile: "/etc/slippios/version",
	}
}
