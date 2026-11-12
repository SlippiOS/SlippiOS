package host

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CleanOptions struct {
	All     bool
	History bool
	Hard    bool
}

func Clean(ws *Workspace, opts CleanOptions) error {
	if opts.Hard {
		return cleanHard(ws)
	}
	if !opts.History {
		cleanVM(ws)
		_ = os.RemoveAll(ws.CachePath("build-tmp"))
		_ = os.MkdirAll(ws.CachePath("build-tmp"), 0o755)
	}
	if opts.All || opts.History {
		if err := CleanHistory(ws); err != nil {
			return err
		}
	}
	if opts.All {
		fmt.Println("Cleaned disposable cache and old images.")
	} else if opts.History {
		fmt.Println("Cleaned build history.")
	}
	PrintDirSize(ws.Cache)
	return nil
}

func CleanHistory(ws *Workspace) error {
	if runningVM(ws) {
		return fmt.Errorf("stop the running VM first: mise run stop:image")
	}

	version := ws.Version()
	releaseDir := ws.CachePath("release")
	if version != "" && dirExists(releaseDir) {
		entries, err := os.ReadDir(releaseDir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.Name() == version {
				continue
			}
			fmt.Printf("Removing old release %s\n", entry.Name())
			if err := os.RemoveAll(filepath.Join(releaseDir, entry.Name())); err != nil {
				return err
			}
		}
	}

	verify := ws.CachePath("verify")
	if dirExists(verify) {
		fmt.Println("Removing verification artifacts")
		if err := os.RemoveAll(verify); err != nil {
			return err
		}
	}

	devResult := ws.Path("result-local-image-dev")
	if info, err := os.Lstat(devResult); err == nil && info.IsDir() {
		fmt.Println("Removing old dev image directory result-local-image-dev")
		if err := os.RemoveAll(devResult); err != nil {
			return err
		}
	}

	keep := map[string]bool{
		"result-local-image":       true,
		"result-minimal-image":     true,
		"result-local-image-dev":   true,
		"result-dev-channel-image": true,
		"result-shell-export":      true,
	}
	keepTargets := map[string]bool{}
	for name := range keep {
		link := ws.Path(name)
		if target, err := os.Readlink(link); err == nil {
			keepTargets[RealPath(target)] = true
		}
	}

	links := Glob(ws.Root, "result-*")
	for _, link := range links {
		info, err := os.Lstat(link)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		name := filepath.Base(link)
		if keep[name] {
			continue
		}
		target := RealPath(link)
		_ = NewNix(ws).StoreDelete(target)
		fmt.Printf("Removing stale result %s\n", name)
		if err := os.Remove(link); err != nil {
			return err
		}
	}

	for _, store := range []string{
		ws.CachePath(".nix-portable", "nix", "store"),
		ws.CachePath("nix", "nix", "store"),
		ws.CachePath("nix", "store"),
	} {
		for _, pattern := range []string{"slippios-0.3.*", "slippios-0.4.*"} {
			for _, path := range Glob(store, pattern) {
				if keepTargets[RealPath(path)] {
					continue
				}
				_ = NewNix(ws).StoreDelete(path)
			}
		}
	}

	return clearDir(ws.CachePath("build-tmp"))
}

func cleanVM(ws *Workspace) {
	for _, pattern := range []string{
		"slippios-*.qcow2",
		"slippios-*.sock",
		"*.iso",
		"ov*.raw",
		"slippios-usb-*.raw*",
		"OVMF_VARS-*.fd",
	} {
		for _, path := range Glob(ws.Cache, pattern) {
			_ = os.RemoveAll(path)
		}
	}
}

func cleanHard(ws *Workspace) error {
	home := homeDir()
	if home == "" {
		return fmt.Errorf("HOME is not set")
	}
	cache := filepath.Clean(ws.Cache)
	if cache == "/" || cache == filepath.Clean(home) {
		return fmt.Errorf("refusing to remove %s", cache)
	}
	chmodTree(cache)
	homePortable := filepath.Join(home, ".nix-portable")
	chmodTree(homePortable)
	if err := os.RemoveAll(cache); err != nil {
		return err
	}
	if err := os.RemoveAll(homePortable); err != nil {
		return err
	}
	fmt.Printf("Wiped %s and %s\n", cache, homePortable)
	return nil
}

func chmodTree(root string) {
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		mode := info.Mode().Perm() | 0o200
		_ = os.Chmod(path, mode)
		return nil
	})
}

func clearDir(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(path, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func runningVM(ws *Workspace) bool {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "qemu-system-x86_64") {
			return true
		}
	}
	return false
}
