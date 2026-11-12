package host

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type RunOptions struct {
	Image      string
	DiskSize   string
	DiskMode   string
	Display    string
	VGA        string
	SecureBoot bool
	SSHPort    string
	Serial     string
	Monitor    string
	InShell    bool
}

func RunImage(ws *Workspace, opts RunOptions) error {
	image := ws.ResolvePath(opts.Image)
	if image == "" {
		found, err := findLocalImage(ws)
		if err != nil {
			return err
		}
		image = found
	}
	if _, err := os.Stat(image); err != nil {
		return fmt.Errorf("cannot read image: %s", image)
	}
	tools := []string{"qemu-img", "qemu-system-x86_64"}
	packages := []string{"nixpkgs#qemu"}
	if strings.HasSuffix(image, ".xz") {
		tools = append(tools, "xz")
		packages = append(packages, "nixpkgs#xz")
	}
	if ran, err := EnsureTools(ws, tools, packages, opts.InShell); ran {
		return err
	}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s is required", tool)
		}
	}

	if opts.DiskSize == "" {
		opts.DiskSize = envDefault("SLIPPI_DISK_SIZE", "16G")
	}
	if opts.DiskMode == "" {
		opts.DiskMode = envDefault("SLIPPI_DISK", "virtio")
	}
	if opts.Display == "" {
		opts.Display = envDefault("SLIPPI_DISPLAY", "gtk")
	}
	if opts.VGA == "" {
		opts.VGA = envDefault("SLIPPI_VGA", "std")
	}
	if opts.SSHPort == "" {
		opts.SSHPort = envDefault("SLIPPI_SSH_PORT", "2222")
	}
	if opts.Serial == "" {
		opts.Serial = os.Getenv("SLIPPI_SERIAL")
	}
	if !opts.SecureBoot {
		opts.SecureBoot = os.Getenv("SLIPPI_SECURE_BOOT") == "1"
	}

	if err := os.MkdirAll(ws.Cache, 0o755); err != nil {
		return err
	}
	imageID, err := ImageID(image)
	if err != nil {
		return err
	}
	raw := image
	if strings.HasSuffix(image, ".xz") {
		raw = ws.CachePath(strings.TrimSuffix(filepath.Base(image), ".xz") + "-" + imageID + ".raw")
		if _, err := os.Stat(raw); err != nil {
			fmt.Printf("Decompressing %s\n", image)
			if err := decompressXZ(image, raw); err != nil {
				return err
			}
		}
	} else {
		raw = RealPath(image)
	}

	overlay := ws.CachePath("slippios-" + imageID + ".qcow2")
	if _, err := os.Stat(overlay); err != nil {
		if err := NewRunner(ws.Root).Run("qemu-img", "create", "-f", "qcow2", "-F", "raw", "-b", raw, overlay, opts.DiskSize); err != nil {
			return err
		}
	}

	code, vars, err := findOVMF(opts.SecureBoot)
	if err != nil {
		return err
	}

	varsCopy := ws.CachePath(fmt.Sprintf("OVMF_VARS-%s-%t.fd", filepath.Base(raw), opts.SecureBoot))
	if _, err := os.Stat(varsCopy); err != nil {
		if err := CopyFile(vars, varsCopy); err != nil {
			return err
		}
	}
	// Nix store firmware is read-only; QEMU needs a writable vars image.
	if err := os.Chmod(varsCopy, 0o644); err != nil {
		return err
	}

	accel := "tcg"
	cpu := "max"
	if kvmAvailable() {
		accel = "kvm"
		cpu = "host"
	}

	args := []string{
		"-machine", "q35,accel=" + accel,
		"-cpu", cpu,
		"-m", "4096", "-smp", "4",
		"-drive", "if=pflash,format=raw,readonly=on,file=" + code,
		"-drive", "if=pflash,format=raw,file=" + varsCopy,
	}
	if opts.DiskMode == "usb" {
		args = append(args,
			"-device", "qemu-xhci",
			"-drive", "if=none,id=usbstick,file="+overlay+",format=qcow2",
			"-device", "usb-storage,drive=usbstick,bootindex=0",
		)
	} else {
		args = append(args,
			"-drive", "if=none,id=disk,file="+overlay+",format=qcow2",
			"-device", "virtio-blk-pci,drive=disk,bootindex=0",
		)
	}
	args = append(args,
		"-nic", "user,model=virtio-net-pci,hostfwd=tcp::8080-:8080,hostfwd=tcp::"+opts.SSHPort+"-:22",
	)
	if opts.VGA == "virtio" {
		args = append(args, "-vga", "none", "-device", "virtio-gpu-pci")
	} else {
		args = append(args, "-vga", "std")
	}
	if opts.Serial != "" {
		args = append(args, "-serial", "file:"+opts.Serial)
	}
	monitor := opts.Monitor
	if monitor == "" {
		monitor = os.Getenv("SLIPPI_MONITOR")
	}
	if monitor == "" {
		monitor = ws.CachePath("slippios-" + imageID + ".sock")
	}
	_ = os.Remove(monitor)
	args = append(args, "-monitor", "unix:"+monitor+",server,nowait", "-display", opts.Display)

	fmt.Printf("Image:   %s\n", raw)
	fmt.Printf("Overlay: %s (%s)\n", overlay, opts.DiskSize)
	fmt.Printf("Disk:    %s\n", opts.DiskMode)
	fmt.Printf("Display: %s\n", opts.Display)
	fmt.Printf("Monitor: %s\n", monitor)
	fmt.Printf("Network: host port 8080 is forwarded; from this PC use http://localhost:8080/<token>/\n")
	fmt.Printf("SSH:     ssh -p %s player@localhost (dev image, password: slippios)\n", opts.SSHPort)

	return Exec(ws.Root, "qemu-system-x86_64", args, os.Environ())
}

func StopImage(ws *Workspace) error {
	sockets := Glob(ws.Cache, "slippios-*.sock")
	if len(sockets) == 0 {
		return fmt.Errorf("no running image (no monitor socket in %s)", ws.Cache)
	}
	newest := sockets[0]
	newestTime := time.Time{}
	for _, socket := range sockets {
		info, err := os.Stat(socket)
		if err == nil && info.ModTime().After(newestTime) {
			newest = socket
			newestTime = info.ModTime()
		}
	}
	fmt.Printf("Sending power button to %s\n", newest)
	conn, err := net.DialTimeout("unix", newest, 2*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := io.WriteString(conn, "system_powerdown\n"); err != nil {
		return err
	}
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	buffer := make([]byte, 4096)
	for {
		if _, err := conn.Read(buffer); err != nil {
			break
		}
	}
	return nil
}

func findLocalImage(ws *Workspace) (string, error) {
	result := ws.Path("result-local-image")
	entries, err := os.ReadDir(result)
	if err != nil {
		return "", fmt.Errorf("no image found; run 'mise run build:image' first")
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(name, ".nix-store.raw") || strings.HasSuffix(name, ".nix-store.raw.xz") {
			continue
		}
		if strings.HasSuffix(name, ".raw") || strings.HasSuffix(name, ".raw.xz") {
			return filepath.Join(result, name), nil
		}
	}
	return "", fmt.Errorf("no image found; run 'mise run build:image' first")
}

func decompressXZ(src, dst string) error {
	args := []string{"-d", "-c", src}
	if xzSupportsThreads() {
		args = []string{"-d", "-T0", "-c", src}
	}
	cmd := exec.Command("xz", args...)
	out, err := os.Create(dst + ".tmp")
	if err != nil {
		return err
	}
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		out.Close()
		os.Remove(dst + ".tmp")
		return fmt.Errorf("xz: %w", err)
	}
	if err := out.Close(); err != nil {
		os.Remove(dst + ".tmp")
		return err
	}
	return os.Rename(dst+".tmp", dst)
}

// BusyBox xz (for example nix-portable's) has no -T/--threads option.
func xzSupportsThreads() bool {
	out, err := exec.Command("xz", "--help").CombinedOutput()
	return err == nil && strings.Contains(string(out), "-T")
}

func findOVMF(secure bool) (string, string, error) {
	type names struct{ code, vars string }
	candidates := []names{
		{"OVMF_CODE.fd", "OVMF_VARS.fd"},
		{"edk2-x86_64-code.fd", "edk2-i386-vars.fd"},
	}
	if secure {
		candidates = []names{
			{"OVMF_CODE.secboot.fd", "OVMF_VARS.secboot.fd"},
			{"edk2-x86_64-secure-code.fd", "edk2-i386-vars.fd"},
		}
	}

	dirs := []string{}
	if dir := os.Getenv("SLIPPI_OVMF_DIR"); dir != "" {
		dirs = append(dirs, dir)
	}
	if qemu, err := exec.LookPath("qemu-system-x86_64"); err == nil {
		dirs = append(dirs,
			filepath.Join(filepath.Dir(filepath.Dir(qemu)), "share", "qemu"))
	}
	dirs = append(dirs,
		"/usr/share/OVMF",
		"/usr/share/edk2/ovmf",
		"/usr/share/edk2/ovmf/x64",
		"/usr/share/edk2/x64",
		"/usr/share/edk2/x86_64",
		"/usr/share/edk2-ovmf",
		"/usr/share/edk2-ovmf/x64",
		"/usr/share/qemu",
	)
	for _, candidate := range candidates {
		for _, dir := range dirs {
			code := filepath.Join(dir, candidate.code)
			vars := filepath.Join(dir, candidate.vars)
			if fileExists(code) && fileExists(vars) {
				return code, vars, nil
			}
		}
	}
	return "", "", fmt.Errorf("UEFI firmware not found; install edk2-ovmf/ovmf or set SLIPPI_OVMF_DIR")
}

func kvmAvailable() bool {
	file, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	file.Close()
	return true
}

func envDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
