package host

import (
	"fmt"
	"os"
	"path/filepath"
)

type ReleaseDevOptions struct {
	NoBuild    bool
	DryRun     bool
	KeepRaw    bool
	AllowDirty bool
	Result     string
	AssetsDir  string
	InShell    bool
}

func ReleaseDev(ws *Workspace, opts ReleaseDevOptions) error {
	done, err := EnsureTools(ws, releaseTools, releasePackages, opts.InShell)
	if done {
		return err
	}
	key, _, err := signingKey()
	if err != nil {
		return err
	}
	runner := NewRunner(ws.Root)
	if !opts.AllowDirty {
		status, err := runner.Output("git", "status", "--porcelain")
		if err != nil {
			return err
		}
		if status != "" {
			return fmt.Errorf("working tree is dirty; commit first or pass --allow-dirty")
		}
	} else {
		fmt.Fprintln(os.Stderr, "warning: dirty tree; the dev version is based on the last commit")
	}

	result := opts.Result
	if result == "" {
		result = os.Getenv("SLIPPI_RESULT")
	}
	if result == "" {
		result = "result-local-image-dev"
	}
	if !opts.NoBuild {
		if info, err := os.Lstat(ws.Path(result)); err == nil && info.IsDir() {
			return fmt.Errorf("%s is a directory; move the old image aside or set SLIPPI_RESULT", result)
		}
		if err := BuildImage(ws, BuildOptions{Config: "slippios-usb-dev", Result: result}); err != nil {
			return err
		}
	} else if !dirExists(ws.Path(result)) {
		return fmt.Errorf("%s is missing; run without --no-build", result)
	}

	resultDir := ws.Path(result)
	imageRaw, err := FindOne(resultDir, "slippios_*.raw", ".nix-store.raw")
	if err != nil {
		return err
	}
	nixStoreRaw, err := FindOne(resultDir, "slippios_*.nix-store.raw")
	if err != nil {
		return err
	}
	imageRaw = RealPath(imageRaw)
	nixStoreRaw = RealPath(nixStoreRaw)
	version := versionFromImage(filepath.Base(imageRaw))
	if filepath.Base(nixStoreRaw) != "slippios_"+version+".nix-store.raw" {
		return fmt.Errorf("image and store image versions do not match")
	}

	assets := opts.AssetsDir
	if assets == "" {
		assets = os.Getenv("SLIPPIOS_DEV_ASSETS")
	}
	if assets == "" {
		assets = ws.CachePath("release-dev")
	}
	assets = RealPath(ws.ResolvePath(assets))
	if err := os.MkdirAll(assets, 0o755); err != nil {
		return err
	}
	for _, pattern := range []string{
		"slippios_*.nix-store.raw",
		"slippios_*.nix-store.raw.zsync",
		"slippios_*.efi",
		"slippios_*-signed.raw",
		"manifest.json",
		"SHA256SUMS",
		"SHA256SUMS.update",
	} {
		for _, path := range Glob(assets, pattern) {
			_ = os.Remove(path)
		}
	}
	_ = os.RemoveAll(filepath.Join(assets, "uki"))

	if err := CheckFreeSpace(assets, 20_000_000*1024); err != nil {
		return err
	}

	signedRaw := filepath.Join(assets, "slippios_"+version+"-signed.raw")
	if !opts.KeepRaw {
		defer os.Remove(signedRaw)
	}
	defer os.RemoveAll(filepath.Join(assets, "uki"))
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Image:   %s\n", imageRaw)
	fmt.Printf("Assets:  %s\n", assets)
	fmt.Printf("Key:     %s\n", key)

	fmt.Println("Signing image...")
	if err := SignImage(ws, SignOptions{Input: imageRaw, Output: signedRaw, InShell: opts.InShell}); err != nil {
		return err
	}

	offset, err := espOffset(signedRaw)
	if err != nil {
		return err
	}
	fmt.Println("Extracting UKI...")
	ukiDir := filepath.Join(assets, "uki")
	if err := os.MkdirAll(ukiDir, 0o755); err != nil {
		return err
	}
	if err := runner.Run("mcopy", "-i", signedRaw+"@@"+fmt.Sprint(offset), "::/EFI/Linux/*.efi", ukiDir+"/"); err != nil {
		return err
	}
	ukis, err := FindImages(ukiDir, "*.efi")
	if err != nil || len(ukis) != 1 {
		return fmt.Errorf("expected one UKI in the image, found %d", len(ukis))
	}
	uki := filepath.Join(assets, "slippios_"+version+".efi")
	if err := os.Rename(ukis[0], uki); err != nil {
		return err
	}
	_ = os.RemoveAll(ukiDir)

	fmt.Println("Staging update assets...")
	nixStore := filepath.Join(assets, "slippios_"+version+".nix-store.raw")
	if err := LinkOrCopy(nixStoreRaw, nixStore); err != nil {
		return err
	}
	if err := runner.Run("zsyncmake", "-u", filepath.Base(nixStore), "-o", nixStore+".zsync", nixStore); err != nil {
		return err
	}

	rawSHA, err := SHA256File(nixStore)
	if err != nil {
		return err
	}
	rawSize, _ := FileSize(nixStore)
	ukiSHA, err := SHA256File(uki)
	if err != nil {
		return err
	}
	ukiSize, _ := FileSize(uki)
	commit, err := runner.Output("git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	manifest := Manifest{
		Version: version,
		Tag:     "dev-" + version,
		Commit:  commit,
		Date:    NowUTC(),
		NixStore: &Asset{
			Name: filepath.Base(nixStore), SHA256: rawSHA, Size: rawSize,
			Zsync: filepath.Base(nixStore) + ".zsync",
		},
		UKI: &Asset{Name: filepath.Base(uki), SHA256: ukiSHA, Size: ukiSize},
	}
	if err := WriteManifest(filepath.Join(assets, "manifest.json"), manifest); err != nil {
		return err
	}
	if err := writeSHA256Sums(filepath.Join(assets, "SHA256SUMS.update"), assets,
		[]string{filepath.Base(nixStore), filepath.Base(nixStore) + ".zsync", filepath.Base(uki), "manifest.json"}); err != nil {
		return err
	}

	fmt.Println()
	for _, name := range []string{filepath.Base(nixStore), filepath.Base(nixStore) + ".zsync", filepath.Base(uki), "manifest.json", "SHA256SUMS.update"} {
		if size, err := FileSize(filepath.Join(assets, name)); err == nil {
			fmt.Printf("%12s  %s\n", HumanSize(size), name)
		}
	}
	if opts.KeepRaw {
		fmt.Printf("Initial flash image: %s\n", signedRaw)
		fmt.Println("Flash it once with: mise run flash:usb")
	} else {
		_ = os.Remove(signedRaw)
	}

	if opts.DryRun {
		fmt.Printf("\nDry run: assets are in %s\n", assets)
		return nil
	}
	return PublishR2(ws, assets, PublishR2Options{Dev: true})
}
