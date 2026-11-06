package host

import (
	"fmt"
	"os"
	"path/filepath"
)

// Workspace locates the repository and its disposable cache. The developer
// tool is normally started by `go -C backend run`, so the process working
// directory may be backend/ rather than the repository root.
type Workspace struct {
	Root  string
	Cache string
}

func FindWorkspace() (*Workspace, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		if fileExists(filepath.Join(dir, "mise.toml")) &&
			fileExists(filepath.Join(dir, "backend", "go.mod")) {
			cache := os.Getenv("SLIPPIOS_CACHE")
			if cache == "" {
				cache = filepath.Join(dir, ".cache")
			}
			if !filepath.IsAbs(cache) {
				cache = filepath.Join(dir, cache)
			}
			return &Workspace{Root: dir, Cache: filepath.Clean(cache)}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, fmt.Errorf("cannot find the SlippiOS repository root (run from the repository)")
}

func (w *Workspace) Path(parts ...string) string {
	return filepath.Join(append([]string{w.Root}, parts...)...)
}

func (w *Workspace) CachePath(parts ...string) string {
	return filepath.Join(append([]string{w.Cache}, parts...)...)
}

func (w *Workspace) Chdir() error {
	return os.Chdir(w.Root)
}

func (w *Workspace) ResolvePath(path string) string {
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(w.Root, path)
	}
	return filepath.Clean(path)
}

func (w *Workspace) Version() string {
	data, err := os.ReadFile(w.Path("VERSION"))
	if err != nil {
		return ""
	}
	return trimSpace(string(data))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
