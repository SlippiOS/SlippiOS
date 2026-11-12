package host

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ReleaseMinimalOptions struct {
	AssetsDir string
	InShell   bool
}

func ReleaseMinimal(ws *Workspace, opts ReleaseMinimalOptions) error {
	done, err := EnsureTools(ws, releaseTools, releasePackages, opts.InShell)
	if done {
		return err
	}
	if _, _, err := signingKey(); err != nil {
		return err
	}
	version := ws.Version()
	if version == "" {
		return fmt.Errorf("VERSION is missing")
	}
	assets := opts.AssetsDir
	if assets == "" {
		assets = ws.CachePath("release", version)
	}
	assets = RealPath(ws.ResolvePath(assets))
	if !fileExists(filepath.Join(assets, "manifest.json")) {
		return fmt.Errorf("missing %s; run mise run release first", filepath.Join(assets, "manifest.json"))
	}
	runner := NewRunner(ws.Root)
	status, err := runner.Output("git", "status", "--porcelain")
	if err != nil {
		return err
	}
	if status != "" {
		return fmt.Errorf("working tree is dirty; commit first")
	}

	signedName := "slippios-usb-" + version + "-minimal.img.xz"
	unsignedName := "slippios-usb-" + version + "-minimal-unsigned.img.xz"
	signedRaw := filepath.Join(assets, "slippios-usb-"+version+"-minimal-secureboot.raw")
	unsignedRaw := filepath.Join(assets, "slippios-usb-"+version+"-minimal-unsigned.raw")

	if err := BuildImage(ws, BuildOptions{Config: "slippios-usb-minimal", Result: "result-minimal-image"}); err != nil {
		return err
	}
	imageXZ, err := FindOne(ws.Path("result-minimal-image"), "slippios_*.raw.xz", ".nix-store.raw.xz")
	if err != nil {
		return fmt.Errorf("cannot find the minimal image in result-minimal-image")
	}
	imageXZ = RealPath(imageXZ)

	defer func() {
		_ = os.Remove(signedRaw)
		_ = os.Remove(unsignedRaw)
	}()

	fmt.Println("Decompressing minimal image...")
	if err := decompressXZ(imageXZ, unsignedRaw); err != nil {
		return err
	}
	digHoles(unsignedRaw)
	fmt.Println("Signing minimal image...")
	if err := SignImage(ws, SignOptions{Input: unsignedRaw, Output: signedRaw, InShell: opts.InShell}); err != nil {
		return err
	}
	fmt.Println("Compressing minimal image...")
	if err := compressXZ(signedRaw, filepath.Join(assets, signedName)); err != nil {
		return err
	}
	if err := CopyFile(imageXZ, filepath.Join(assets, unsignedName)); err != nil {
		return err
	}

	sumsPath := filepath.Join(assets, "SHA256SUMS")
	data, err := os.ReadFile(sumsPath)
	if err != nil {
		return err
	}
	var kept []string
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if line == "" {
			continue
		}
		if strings.Contains(line, "slippios-usb-"+version+"-minimal.img.xz") ||
			strings.Contains(line, "slippios-usb-"+version+"-minimal-unsigned.img.xz") {
			continue
		}
		kept = append(kept, line)
	}
	for _, name := range []string{signedName, unsignedName} {
		sum, err := SHA256File(filepath.Join(assets, name))
		if err != nil {
			return err
		}
		kept = append(kept, sum+"  "+name)
	}
	if err := os.WriteFile(sumsPath, []byte(strings.Join(kept, "\n")+"\n"), 0o644); err != nil {
		return err
	}

	fmt.Println()
	for _, name := range []string{signedName, unsignedName, "SHA256SUMS"} {
		if size, err := FileSize(filepath.Join(assets, name)); err == nil {
			fmt.Printf("%12s  %s\n", HumanSize(size), name)
		}
	}
	fmt.Println()
	fmt.Println("Minimal image ready. Publish it with:")
	fmt.Printf("  mise run publish:sourceforge -- %s <sourceforge-user>\n", assets)
	return nil
}
