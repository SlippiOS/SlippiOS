package host

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	shimVersion      = "16.1-7"
	shimRPMURL       = "https://download.fedoraproject.org/pub/fedora/linux/updates/44/Everything/x86_64/Packages/s/shim-x64-16.1-7.x86_64.rpm"
	shimRPMSHA256    = "4badee0c00c4c41a53f41c8a2dd566dca815ca051f1d2fba92490d0a0194f5c3"
	shimSHA256       = "351e131d3c4a636704e9cae7d1296baf8a05beca9922cfebbfa48933c9bf1ad3"
	mokManagerSHA256 = "ed4442fa88cccba3a9e6a32d21a8f4c56bf5aab184f1fb9aadde8d3a9d04dc27"
)

var signImageTools = []string{
	"curl", "sbsign", "sbverify", "mcopy", "mdir", "mdel", "mmd",
	"parted", "rpm2cpio", "cpio", "openssl",
}

var signImagePackages = []string{
	"nixpkgs#sbsigntool", "nixpkgs#mtools", "nixpkgs#rpm",
	"nixpkgs#cpio", "nixpkgs#parted", "nixpkgs#openssl", "nixpkgs#curl",
}

type SignOptions struct {
	Input   string
	Output  string
	InShell bool
}

func SignImage(ws *Workspace, opts SignOptions) error {
	done, err := EnsureTools(ws, signImageTools, signImagePackages, opts.InShell)
	if done {
		return err
	}
	if opts.Input == "" || opts.Output == "" {
		return fmt.Errorf("usage: sign image <unsigned.raw> <signed.raw>")
	}
	opts.Input = ws.ResolvePath(opts.Input)
	opts.Output = ws.ResolvePath(opts.Output)
	key := os.Getenv("SLIPPIOS_SB_KEY")
	cert := os.Getenv("SLIPPIOS_SB_CERT")
	if key == "" || cert == "" {
		return fmt.Errorf("set SLIPPIOS_SB_KEY and SLIPPIOS_SB_CERT")
	}
	if !fileExists(opts.Input) || !fileExists(key) || !fileExists(cert) {
		return fmt.Errorf("image, key, and certificate must exist")
	}
	if RealPath(opts.Input) == RealPath(opts.Output) {
		return fmt.Errorf("input and output must differ")
	}

	shim, mokManager, err := ensureShim(ws)
	if err != nil {
		return err
	}
	offset, err := espOffset(opts.Input)
	if err != nil {
		return err
	}

	work, err := os.MkdirTemp(ws.CachePath("secureboot"), "work.")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	ukiDir := filepath.Join(work, "uki")
	if err := os.MkdirAll(ukiDir, 0o755); err != nil {
		return err
	}

	runner := NewRunner(ws.Root)
	boot := filepath.Join(work, "systemd-boot.efi")
	if err := runner.Run("mcopy", "-i", opts.Input+"@@"+strconv.FormatInt(offset, 10), "::/EFI/BOOT/BOOTX64.EFI", boot); err != nil {
		return err
	}
	if err := runner.Run("mcopy", "-i", opts.Input+"@@"+strconv.FormatInt(offset, 10), "::/EFI/Linux/*.efi", ukiDir+"/"); err != nil {
		return err
	}
	ukis, err := FindImages(ukiDir, "*.efi")
	if err != nil || len(ukis) == 0 {
		return fmt.Errorf("cannot find a UKI in the ESP")
	}

	if output, err := runner.Output("sbverify", "--list", boot); err == nil &&
		strings.Contains(output, "Microsoft Corporation UEFI CA") {
		return fmt.Errorf("input image is already signed; use the unsigned image")
	}

	if err := os.MkdirAll(filepath.Dir(opts.Output), 0o755); err != nil {
		return err
	}
	if err := copySparse(opts.Input, opts.Output); err != nil {
		return err
	}
	if err := os.Chmod(opts.Output, 0o644); err != nil {
		return err
	}

	signedBoot := filepath.Join(work, "systemd-boot.signed.efi")
	if err := runner.Run("sbsign", "--key", key, "--cert", cert, "--output", signedBoot, boot); err != nil {
		return err
	}
	if err := runner.Run("sbverify", "--cert", cert, signedBoot); err != nil {
		return err
	}

	for _, uki := range ukis {
		name := filepath.Base(uki)
		signed := uki + ".signed"
		if err := runner.Run("sbsign", "--key", key, "--cert", cert, "--output", signed, uki); err != nil {
			return err
		}
		if err := runner.Run("sbverify", "--cert", cert, signed); err != nil {
			return err
		}
		if err := runner.Run("mcopy", "-i", opts.Output+"@@"+strconv.FormatInt(offset, 10), "-o", signed, "::/EFI/Linux/"+name); err != nil {
			return err
		}
	}

	cer := filepath.Join(work, "slippios.cer")
	if err := runner.Run("openssl", "x509", "-in", cert, "-outform", "DER", "-out", cer); err != nil {
		return err
	}

	target := opts.Output + "@@" + strconv.FormatInt(offset, 10)
	for _, item := range []struct {
		source string
		target string
	}{
		{shim, "::/EFI/BOOT/BOOTX64.EFI"},
		{signedBoot, "::/grubx64.efi"},
		{signedBoot, "::/EFI/BOOT/grubx64.efi"},
		{mokManager, "::/mmx64.efi"},
		{mokManager, "::/EFI/BOOT/mmx64.efi"},
		{cer, "::/slippios.cer"},
		{cer, "::/EFI/BOOT/slippios.cer"},
		{ws.Path("secure-boot", "ENROLL.txt"), "::/SECUREBOOT.txt"},
		{ws.Path("secure-boot", "LICENSE.shim"), "::/SHIM-LICENSE.txt"},
	} {
		if err := runner.Run("mcopy", "-i", target, "-o", item.source, item.target); err != nil {
			return err
		}
	}
	_ = runner.RunQuiet("mdel", "-i", target, "::/EFI/BOOT/fbx64.efi")
	_ = runner.RunQuiet("mmd", "-i", target, "::/EFI/systemd")
	if err := runner.Run("mcopy", "-i", target, "-o", signedBoot, "::/EFI/systemd/systemd-bootx64.efi"); err != nil {
		return err
	}

	fmt.Printf("Signed image: %s\n", opts.Output)
	fmt.Printf("ESP offset:   %d\n", offset)
	fmt.Println("First boot:   enroll slippios.cer in MOK")
	return nil
}

func SignUKI(ws *Workspace, opts SignOptions) error {
	done, err := EnsureTools(ws, []string{"sbsign", "sbverify"}, []string{"nixpkgs#sbsigntool"}, opts.InShell)
	if done {
		return err
	}
	if opts.Input == "" || opts.Output == "" {
		return fmt.Errorf("usage: sign uki <unsigned.efi> <signed.efi>")
	}
	opts.Input = ws.ResolvePath(opts.Input)
	opts.Output = ws.ResolvePath(opts.Output)
	key := os.Getenv("SLIPPIOS_SB_KEY")
	cert := os.Getenv("SLIPPIOS_SB_CERT")
	if key == "" || cert == "" {
		return fmt.Errorf("set SLIPPIOS_SB_KEY and SLIPPIOS_SB_CERT")
	}
	if !fileExists(opts.Input) || !fileExists(key) || !fileExists(cert) {
		return fmt.Errorf("input, key, and certificate must exist")
	}
	if RealPath(opts.Input) == RealPath(opts.Output) {
		return fmt.Errorf("input and output must differ")
	}
	if err := os.MkdirAll(filepath.Dir(opts.Output), 0o755); err != nil {
		return err
	}
	runner := NewRunner(ws.Root)
	if err := runner.Run("sbsign", "--key", key, "--cert", cert, "--output", opts.Output, opts.Input); err != nil {
		return err
	}
	if err := runner.Run("sbverify", "--cert", cert, opts.Output); err != nil {
		return err
	}
	fmt.Printf("Signed UKI: %s\n", opts.Output)
	return nil
}

func ensureShim(ws *Workspace) (string, string, error) {
	base := ws.CachePath("secureboot")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", "", err
	}
	rpmPath := filepath.Join(base, "shim-x64-"+shimVersion+".x86_64.rpm")
	if !fileExists(rpmPath) {
		tmp := rpmPath + ".tmp"
		if err := downloadFile(shimRPMURL, tmp); err != nil {
			return "", "", err
		}
		sum, err := SHA256File(tmp)
		if err != nil {
			return "", "", err
		}
		if sum != shimRPMSHA256 {
			os.Remove(tmp)
			return "", "", fmt.Errorf("sha256 mismatch for shim rpm")
		}
		if err := os.Rename(tmp, rpmPath); err != nil {
			return "", "", err
		}
	}
	sum, err := SHA256File(rpmPath)
	if err != nil {
		return "", "", err
	}
	if sum != shimRPMSHA256 {
		return "", "", fmt.Errorf("sha256 mismatch for %s", rpmPath)
	}

	root := filepath.Join(base, "shim-"+shimVersion)
	shim := filepath.Join(root, "usr/lib/efi/shim", shimVersion, "EFI/fedora/shimx64.efi")
	mokManager := filepath.Join(root, "usr/lib/efi/shim", shimVersion, "EFI/fedora/mmx64.efi")
	if !fileExists(shim) || !fileExists(mokManager) {
		if err := os.RemoveAll(root); err != nil {
			return "", "", err
		}
		if err := os.MkdirAll(root, 0o755); err != nil {
			return "", "", err
		}
		if err := extractRPM(rpmPath, root); err != nil {
			return "", "", err
		}
	}
	for _, item := range []struct {
		path string
		sum  string
	}{
		{shim, shimSHA256},
		{mokManager, mokManagerSHA256},
	} {
		sum, err := SHA256File(item.path)
		if err != nil {
			return "", "", err
		}
		if sum != item.sum {
			return "", "", fmt.Errorf("sha256 mismatch for %s", item.path)
		}
	}

	out, err := NewRunner(ws.Root).Output("sbverify", "--list", shim)
	if err != nil {
		return "", "", err
	}
	if !strings.Contains(out, "Microsoft Corporation UEFI CA 2011") ||
		!strings.Contains(out, "Microsoft UEFI CA 2023") {
		return "", "", fmt.Errorf("shim is missing the Microsoft UEFI CA signatures")
	}
	return shim, mokManager, nil
}

func extractRPM(rpmPath, dir string) error {
	rpm := exec.Command("rpm2cpio", rpmPath)
	cpio := exec.Command("cpio", "-idm", "--quiet")
	cpio.Dir = dir
	pipe, err := rpm.StdoutPipe()
	if err != nil {
		return err
	}
	cpio.Stdin = pipe
	cpio.Stdout = os.Stdout
	cpio.Stderr = os.Stderr
	if err := cpio.Start(); err != nil {
		return err
	}
	if err := rpm.Start(); err != nil {
		return err
	}
	if err := rpm.Wait(); err != nil {
		return err
	}
	return cpio.Wait()
}

func espOffset(image string) (int64, error) {
	out, err := NewRunner("").Output("parted", "-s", "-m", image, "unit", "B", "print")
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 6 || fields[5] != "boot" {
			continue
		}
		value := strings.TrimSuffix(fields[1], "B")
		offset, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return offset, nil
		}
	}
	return 0, fmt.Errorf("cannot find the ESP partition")
}

func copySparse(src, dst string) error {
	runner := NewRunner("")
	if err := runner.Run("cp", "--reflink=auto", "--sparse=always", src, dst); err == nil {
		return nil
	}
	return runner.Run("cp", "--sparse=always", src, dst)
}

func streamToFile(command *exec.Cmd, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	command.Stdout = file
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		file.Close()
		os.Remove(path)
		return err
	}
	return file.Close()
}
