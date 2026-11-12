package host

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type BuildOptions struct {
	Config    string
	Dev       bool
	Minimal   bool
	Result    string
	Jobs      int
	SkipClean bool
	TmpDir    string
}

func BuildImage(ws *Workspace, opts BuildOptions) error {
	config := opts.Config
	if config == "" {
		config = os.Getenv("SLIPPI_CONFIG")
	}
	switch {
	case config != "":
	case opts.Dev || os.Getenv("SLIPPI_DEV") == "1":
		config = "slippios-usb-dev"
	case opts.Minimal:
		config = "slippios-usb-minimal"
	default:
		config = "slippios-usb"
	}

	result := opts.Result
	if result == "" {
		result = os.Getenv("SLIPPI_RESULT")
	}
	if result == "" {
		result = "result-local-image"
	}
	resultPath := result
	if !filepath.IsAbs(resultPath) {
		resultPath = ws.Path(resultPath)
	}

	jobs := opts.Jobs
	if jobs == 0 {
		if value := os.Getenv("SLIPPI_JOBS"); value != "" {
			if parsed, err := strconv.Atoi(value); err == nil {
				jobs = parsed
			}
		}
	}
	if jobs == 0 {
		jobs = runtime.NumCPU()
		if jobs > 8 {
			jobs = 8
		}
	}

	tmpDir := opts.TmpDir
	if tmpDir == "" {
		tmpDir = os.Getenv("SLIPPI_TMPDIR")
	}
	if tmpDir == "" {
		tmpDir = ws.CachePath("build-tmp")
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	os.Setenv("TMPDIR", tmpDir)

	if !opts.SkipClean && os.Getenv("SLIPPIOS_SKIP_CLEAN") != "1" {
		if err := CleanHistory(ws); err != nil {
			fmt.Fprintf(os.Stderr, "warning: build history cleanup skipped: %v\n", err)
		}
	}

	oldTarget := ""
	if target, err := os.Readlink(resultPath); err == nil {
		oldTarget = target
	}
	if err := os.Remove(resultPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	nix := NewNix(ws)
	if nix.hasSystemNix() {
		fmt.Printf("Building with system Nix (%d cores)...\n", jobs)
	} else {
		fmt.Printf("Building with nix-portable/bwrap (%d cores)...\n", jobs)
		fmt.Printf("  TMPDIR: %s\n", tmpDir)
	}

	args := []string{
		"build",
		"--option", "extra-substituters", "https://cache.lantian.pub/lantian",
		"--option", "extra-trusted-public-keys", "lantian:WVGO7vsRMSEN+h7Lp7kv6SAmrkEXgUGbbXm7jMDjBuQ=",
		"--print-build-logs",
		"--cores", fmt.Sprint(jobs),
		"--max-jobs", "1",
		".#nixosConfigurations." + config + ".config.system.build.image",
		"-o", resultPath,
	}
	if err := nix.Run(args...); err != nil {
		return err
	}

	if config == "slippios-usb-dev" {
		fmt.Println()
		fmt.Println("Dev build: image is uncompressed.")
	}

	newTarget, err := rewriteResult(ws, resultPath)
	if err != nil {
		return err
	}

	if config == "slippios-usb-dev" && oldTarget != "" && oldTarget != newTarget &&
		isDevImage(oldTarget) {
		fmt.Printf("Pruning previous dev image: %s\n", oldTarget)
		_ = nix.StoreDelete(oldTarget)
	}

	fmt.Println("Artifacts:")
	entries, err := os.ReadDir(resultPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		fmt.Printf("  %s  %d bytes\n", entry.Name(), info.Size())
	}
	return nil
}

func rewriteResult(ws *Workspace, result string) (string, error) {
	target, err := os.Readlink(result)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(target, "/nix/store/") {
		if _, err := os.Stat(target); err != nil {
			for _, candidate := range []string{
				filepath.Join(ws.Cache, ".nix-portable", target),
				filepath.Join(ws.Cache, "nix", target),
				filepath.Join(ws.Cache, "nix", strings.TrimPrefix(target, "/nix")),
			} {
				if _, err := os.Stat(candidate); err == nil {
					target = candidate
					break
				}
			}
		}
	}
	// Keep result links on the physical store, not a compatibility symlink.
	compat := ws.CachePath("nix", "store") + string(filepath.Separator)
	if strings.HasPrefix(target, compat) {
		target = ws.CachePath(".nix-portable", "nix", "store") + string(filepath.Separator) +
			strings.TrimPrefix(target, compat)
	}
	if _, err := os.Stat(target); err == nil {
		if err := os.Remove(result); err != nil {
			return "", err
		}
		if err := os.Symlink(target, result); err != nil {
			return "", err
		}
	}
	return target, nil
}

// A dev image is an uncompressed .raw. Release and minimal images are
// .raw.xz, so only prune a previous result that actually is a dev image.
func isDevImage(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	hasRaw := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".raw.xz") {
			return false
		}
		if strings.HasSuffix(name, ".raw") {
			hasRaw = true
		}
	}
	return hasRaw
}
