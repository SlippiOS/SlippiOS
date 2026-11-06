package host

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var releaseTools = []string{
	"git", "xz", "parted", "mcopy", "mdel", "mmd", "sbsign", "sbverify",
	"rpm2cpio", "cpio", "openssl", "zsyncmake", "curl",
}

var releasePackages = []string{
	"nixpkgs#sbsigntool", "nixpkgs#mtools", "nixpkgs#rpm",
	"nixpkgs#cpio", "nixpkgs#parted", "nixpkgs#openssl",
	"nixpkgs#curl", "nixpkgs#zsync", "nixpkgs#xz",
}

type ReleaseOptions struct {
	NoBuild    bool
	DryRun     bool
	Publish    bool
	Force      bool
	AllowDirty bool
	Tag        string
	AssetsDir  string
	KeepRaw    bool
	Result     string
	InShell    bool
}

func Release(ws *Workspace, opts ReleaseOptions) error {
	tools := append([]string{}, releaseTools...)
	if opts.Publish {
		tools = append(tools, "gh")
	}
	done, err := EnsureTools(ws, tools, releasePackages, opts.InShell)
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
	}

	result := opts.Result
	if result == "" {
		result = "result-local-image"
	}
	if !opts.NoBuild {
		if err := BuildImage(ws, BuildOptions{Config: "slippios-usb", Result: result}); err != nil {
			return err
		}
	} else if !dirExists(ws.Path(result)) {
		return fmt.Errorf("%s is missing; run without --no-build", result)
	}

	resultDir := ws.Path(result)
	imageRaw, rawErr := FindOne(resultDir, "slippios_*.raw", ".nix-store.raw")
	imageXZ := ""
	var version string
	if rawErr != nil {
		imageXZ, err = FindOne(resultDir, "slippios_*.raw.xz", ".nix-store.raw.xz")
		if err != nil {
			return fmt.Errorf("cannot find the full image in %s", resultDir)
		}
		imageXZ = RealPath(imageXZ)
		version = versionFromImage(filepath.Base(imageXZ))
	} else {
		imageRaw = RealPath(imageRaw)
		version = versionFromImage(filepath.Base(imageRaw))
		imageXZ, _ = FindOne(resultDir, "slippios_*.raw.xz", ".nix-store.raw.xz")
		if imageXZ != "" {
			imageXZ = RealPath(imageXZ)
		}
	}
	if version == "" {
		return fmt.Errorf("cannot determine version from the image in %s", resultDir)
	}

	nixStoreRaw, nixStoreErr := FindOne(resultDir, "slippios_"+version+".nix-store.raw")
	if nixStoreErr != nil {
		nixStoreXZ, err := FindOne(resultDir, "slippios_"+version+".nix-store.raw.xz")
		if err != nil {
			return fmt.Errorf("cannot find the nix-store image in %s", resultDir)
		}
		nixStoreRaw = nixStoreXZ
	}
	tag := opts.Tag
	if tag == "" {
		tag = "v" + version
	}
	assets := opts.AssetsDir
	if assets == "" {
		assets = ws.CachePath("release", version)
	}
	assets = RealPath(ws.ResolvePath(assets))
	if err := os.MkdirAll(assets, 0o755); err != nil {
		return err
	}
	if err := CheckFreeSpace(assets, 20_000_000*1024); err != nil {
		return err
	}

	signedName := "slippios-usb-" + version + ".img.xz"
	unsignedName := "slippios-usb-" + version + "-unsigned.img.xz"
	rawName := "slippios_" + version + ".nix-store.raw"
	ukiName := "slippios_" + version + ".efi"
	signedRaw := filepath.Join(assets, "slippios-usb-"+version+"-secureboot.raw")
	unsigned := imageRaw
	decompressedUnsigned := ""
	if imageRaw == "" || strings.HasSuffix(imageRaw, ".xz") {
		unsigned = filepath.Join(assets, "slippios-"+version+"-unsigned.raw")
		decompressedUnsigned = unsigned
	}

	for _, name := range []string{signedName, unsignedName, ukiName, "manifest.json", "SHA256SUMS", rawName + ".zsync"} {
		_ = os.Remove(filepath.Join(assets, name))
	}

	defer func() {
		if !opts.KeepRaw {
			_ = os.Remove(signedRaw)
		}
		if decompressedUnsigned != "" {
			_ = os.Remove(decompressedUnsigned)
		}
		_ = os.RemoveAll(filepath.Join(assets, "uki"))
	}()

	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Tag:     %s\n", tag)
	fmt.Printf("Assets:  %s\n", assets)
	fmt.Printf("Key:     %s\n", key)

	if decompressedUnsigned != "" {
		fmt.Println("Decompressing full image...")
		source := imageRaw
		if source == "" {
			source = imageXZ
		}
		if err := decompressXZ(source, unsigned); err != nil {
			return err
		}
		digHoles(unsigned)
	}

	nixStore := filepath.Join(assets, rawName)
	if strings.HasSuffix(nixStoreRaw, ".xz") {
		fmt.Println("Decompressing nix-store image...")
		if err := decompressXZ(nixStoreRaw, nixStore); err != nil {
			return err
		}
		digHoles(nixStore)
	} else if err := LinkOrCopy(nixStoreRaw, nixStore); err != nil {
		return err
	}

	offset, err := espOffset(unsigned)
	if err != nil {
		return err
	}
	fmt.Println("Signing image...")
	if err := SignImage(ws, SignOptions{Input: unsigned, Output: signedRaw, InShell: opts.InShell}); err != nil {
		return err
	}

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
	uki := filepath.Join(assets, ukiName)
	if err := os.Rename(ukis[0], uki); err != nil {
		return err
	}

	fmt.Println("Compressing signed image...")
	if err := compressXZ(signedRaw, filepath.Join(assets, signedName)); err != nil {
		return err
	}

	if imageXZ != "" {
		if err := LinkOrCopy(RealPath(imageXZ), filepath.Join(assets, unsignedName)); err != nil {
			return err
		}
	} else {
		fmt.Println("Compressing unsigned image...")
		if err := compressXZ(unsigned, filepath.Join(assets, unsignedName)); err != nil {
			return err
		}
	}

	if err := runner.Run("zsyncmake", "-u", rawName, "-o", filepath.Join(assets, rawName+".zsync"), nixStore); err != nil {
		return err
	}

	signedSHA, err := SHA256File(filepath.Join(assets, signedName))
	if err != nil {
		return err
	}
	unsignedSHA, err := SHA256File(filepath.Join(assets, unsignedName))
	if err != nil {
		return err
	}
	rawSHA, err := SHA256File(nixStore)
	if err != nil {
		return err
	}
	ukiSHA, err := SHA256File(uki)
	if err != nil {
		return err
	}
	signedSize, _ := FileSize(filepath.Join(assets, signedName))
	unsignedSize, _ := FileSize(filepath.Join(assets, unsignedName))
	rawSize, _ := FileSize(nixStore)
	ukiSize, _ := FileSize(uki)

	commit, err := runner.Output("git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	manifest := Manifest{
		Version:       version,
		Tag:           tag,
		Commit:        commit,
		Date:          NowUTC(),
		FullImage:     &Asset{Name: signedName, SHA256: signedSHA, Size: signedSize},
		UnsignedImage: &Asset{Name: unsignedName, SHA256: unsignedSHA, Size: unsignedSize},
		NixStore:      &Asset{Name: rawName, SHA256: rawSHA, Size: rawSize, Zsync: rawName + ".zsync"},
		UKI:           &Asset{Name: ukiName, SHA256: ukiSHA, Size: ukiSize},
	}
	if err := WriteManifest(filepath.Join(assets, "manifest.json"), manifest); err != nil {
		return err
	}

	for _, name := range []string{"LICENSE", "NOTICE", "THIRD_PARTY_LICENSES.md", "SOURCE", "README.md"} {
		if err := CopyFile(ws.Path(name), filepath.Join(assets, name)); err != nil {
			return err
		}
	}
	_ = os.RemoveAll(filepath.Join(assets, "licenses"))
	if err := copyDir(ws.Path("licenses"), filepath.Join(assets, "licenses")); err != nil {
		return err
	}
	if err := createTarGz(filepath.Join(assets, "LICENSES.tar.gz"), assets,
		[]string{"LICENSE", "NOTICE", "THIRD_PARTY_LICENSES.md", "SOURCE", "README.md", "licenses"}); err != nil {
		return err
	}

	checksumFiles := []string{
		signedName, unsignedName, rawName, rawName + ".zsync", ukiName,
		"manifest.json", "LICENSE", "NOTICE", "THIRD_PARTY_LICENSES.md",
		"SOURCE", "README.md", "LICENSES.tar.gz",
	}
	if err := writeSHA256Sums(filepath.Join(assets, "SHA256SUMS"), assets, checksumFiles); err != nil {
		return err
	}
	if err := writeSHA256Sums(filepath.Join(assets, "SHA256SUMS.update"), assets,
		[]string{rawName, rawName + ".zsync", ukiName, "manifest.json"}); err != nil {
		return err
	}

	fmt.Println()
	for _, name := range []string{
		signedName, unsignedName, rawName, rawName + ".zsync", ukiName,
		"manifest.json", "SHA256SUMS", "SHA256SUMS.update", "LICENSE",
		"THIRD_PARTY_LICENSES.md", "SOURCE", "LICENSES.tar.gz",
	} {
		if size, err := FileSize(filepath.Join(assets, name)); err == nil {
			fmt.Printf("%12s  %s\n", HumanSize(size), name)
		}
	}

	if opts.DryRun {
		fmt.Printf("\nDry run: assets are in %s\n", assets)
		return nil
	}

	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh is required to upload")
	}
	if err := runner.Run("gh", "auth", "status"); err != nil {
		return fmt.Errorf("gh is not authenticated")
	}
	repo, err := runner.Output("gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	if err != nil {
		return err
	}
	exists := runner.RunQuiet("gh", "api", "repos/"+repo+"/releases/tags/"+tag) == nil
	if exists && !opts.Force {
		return fmt.Errorf("release %s already exists; pass --force to replace its assets", tag)
	}
	if !exists {
		notes := fmt.Sprintf(`SlippiOS %s — new mark, multishine boot animation, smoother splash.

Full image (apps preinstalled): https://sourceforge.net/projects/slippi-os/files/v%s/
Minimal image (apps install on first boot): `+"`slippios-usb-%s-minimal.img.xz`"+` from this release.

Secure Boot: signed images need a one-time MOK enrollment; unsigned images need Secure Boot off.

GPL-3.0-or-later · https://github.com/%s
`, version, version, version, repo)
		if err := runner.Run("gh", "release", "create", tag,
			"--repo", repo, "--draft", "--target", commit,
			"--title", "SlippiOS "+version, "--notes", notes); err != nil {
			return err
		}
	}
	if opts.Publish {
		if err := runner.Run("gh", "release", "edit", tag, "--repo", repo, "--draft=false", "--latest"); err != nil {
			return err
		}
		fmt.Printf("Published: https://github.com/%s/releases/tag/%s\n", repo, tag)
	} else {
		fmt.Printf("Draft: https://github.com/%s/releases/tag/%s\n", repo, tag)
		fmt.Printf("Publish with: gh release edit %s --draft=false --latest\n", tag)
	}
	return nil
}

func signingKey() (string, string, error) {
	key := os.Getenv("SLIPPIOS_SB_KEY")
	cert := os.Getenv("SLIPPIOS_SB_CERT")
	if key == "" {
		key = filepath.Join(os.Getenv("HOME"), ".config/slippios/secureboot/slippios.key")
	}
	if cert == "" {
		cert = filepath.Join(os.Getenv("HOME"), ".config/slippios/secureboot/slippios.crt")
	}
	if !fileExists(key) || !fileExists(cert) {
		return "", "", fmt.Errorf("set SLIPPIOS_SB_KEY and SLIPPIOS_SB_CERT, or install the key at %s/.config/slippios/secureboot", os.Getenv("HOME"))
	}
	info, err := os.Stat(key)
	if err == nil {
		mode := info.Mode().Perm()
		if mode != 0o600 && mode != 0o400 {
			fmt.Fprintf(os.Stderr, "warning: %s has mode %o; use chmod 600\n", key, mode)
		}
	}
	os.Setenv("SLIPPIOS_SB_KEY", key)
	os.Setenv("SLIPPIOS_SB_CERT", cert)
	return key, cert, nil
}

func versionFromImage(name string) string {
	name = strings.TrimPrefix(name, "slippios_")
	name = strings.TrimSuffix(name, ".raw.xz")
	name = strings.TrimSuffix(name, ".raw")
	if strings.HasSuffix(name, "-signed") {
		name = strings.TrimSuffix(name, "-signed")
	}
	return name
}

func compressXZ(src, dst string) error {
	return streamToFile(exec.Command("xz", "-T0", "-1", "-c", src), dst)
}

func writeSHA256Sums(path, dir string, names []string) error {
	var builder strings.Builder
	for _, name := range names {
		sum, err := SHA256File(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		fmt.Fprintf(&builder, "%s  %s\n", sum, name)
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func createTarGz(dst, base string, names []string) error {
	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	for _, name := range names {
		if err := addToTar(tw, base, name); err != nil {
			tw.Close()
			gz.Close()
			file.Close()
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return file.Close()
}

func addToTar(tw *tar.Writer, base, name string) error {
	path := filepath.Join(base, name)
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := addToTar(tw, base, filepath.Join(name, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = name
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(tw, file)
	return err
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return CopyFile(path, target)
	})
}
