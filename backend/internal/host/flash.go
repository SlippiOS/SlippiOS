package host

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/pelletier/go-toml/v2"
)

const (
	flashConfigPath  = "/etc/slippios/flash.conf"
	flashHelperPath  = "/usr/local/sbin/slippios-flash"
	flashSudoersPath = "/etc/sudoers.d/slippios-flash"
	flashMarkerPath  = "/etc/slippios/flash-helper.version"
	flashMarkerValue = "slippios-dev-go-2"
)

type FlashConfig struct {
	Device     string `toml:"device"`
	ImageDir   string `toml:"image_dir"`
	AllowFixed bool   `toml:"allow_fixed"`
	SkipVerify bool   `toml:"skip_verify"`
}

type FlashOptions struct {
	Device     string
	ImageDir   string
	Yes        bool
	DryRun     bool
	AllowFixed bool
	SkipVerify bool
}

type USBDevice struct {
	Name      string  `json:"name"`
	Size      string  `json:"size"`
	Model     *string `json:"model"`
	Tran      *string `json:"tran"`
	Removable bool    `json:"rm"`
}

func (d USBDevice) Path() string {
	return "/dev/" + d.Name
}

func (d USBDevice) ModelName() string {
	if d.Model == nil {
		return ""
	}
	return strings.TrimSpace(*d.Model)
}

func (d USBDevice) Fixed() bool {
	return !d.Removable
}

type flashConfigFile struct {
	Device     string `toml:"device"`
	ImageDir   string `toml:"image_dir"`
	AllowFixed bool   `toml:"allow_fixed"`
	SkipVerify bool   `toml:"skip_verify"`
}

func FlashInteractive(ws *Workspace, opts FlashOptions) error {
	if err := ws.Chdir(); err != nil {
		return err
	}
	if opts.ImageDir == "" {
		opts.ImageDir = ws.CachePath("release-dev")
	}
	opts.ImageDir = RealPath(ws.ResolvePath(opts.ImageDir))

	image, err := findFlashImage(opts.ImageDir)
	if err != nil {
		return err
	}

	device := opts.Device
	if device == "" {
		device, err = chooseUSBDevice(opts.Yes)
		if err != nil {
			if err == ErrCancelled {
				fmt.Println("Cancelled.")
			}
			return err
		}
	}
	dev := RealPath(device)
	if _, err := os.Stat(dev); err != nil {
		return fmt.Errorf("target is not present: %s", device)
	}
	if !isWholeDisk(dev) {
		return fmt.Errorf("target is not a whole disk: %s", dev)
	}
	if !isUSBDevice(dev) {
		return fmt.Errorf("target is not a USB disk: %s", dev)
	}
	allowFixed := !isRemovable(dev)

	if !opts.Yes {
		fmt.Printf("Image:  %s\n", image)
		fmt.Printf("Target: %s  %s  %s%s%s\n",
			dev, deviceSize(dev), deviceModel(dev), usbLabel(dev), fixedSuffix(allowFixed))
		ok, err := PromptYesNo(os.Stdin, fmt.Sprintf("This will ERASE %s. Continue? [y/N] ", dev))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelled.")
			return ErrCancelled
		}
	}

	if opts.DryRun {
		fmt.Printf("Would flash %s to %s (allow_fixed=%t, skip_verify=%t).\n", image, dev, allowFixed, opts.SkipVerify)
		return nil
	}

	if err := configureFlash(ws, device, opts.ImageDir, allowFixed, opts.SkipVerify); err != nil {
		return err
	}
	return runFlashHelper()
}

func FlashStatus(ws *Workspace) error {
	helper := installedLabel(flashHelperPath)
	if helper == "installed" && readMarker() == flashMarkerValue {
		helper = "installed (go)"
	}
	fmt.Printf("Helper:  %s\n", helper)
	if NewRunner(ws.Root).RunQuiet("sudo", "-n", "-l", flashHelperPath) == nil {
		fmt.Println("Sudoers: installed")
	} else {
		fmt.Println("Sudoers: missing")
	}
	config, err := readFlashConfig(flashConfigPath)
	if err != nil {
		fmt.Println("Config:  missing")
	} else {
		fmt.Printf("Device:  %s\n", config.Device)
		fmt.Printf("Images:  %s\n", config.ImageDir)
		fmt.Printf("Allow fixed: %s\n", yesNo(config.AllowFixed))
		fmt.Printf("Verify:  %s\n", yesNo(!config.SkipVerify))
		if image, err := findFlashImage(config.ImageDir); err == nil {
			fmt.Printf("Image:   %s\n", image)
		}
	}
	fmt.Println("USB:")
	devices, err := listUSBDevices()
	if err != nil {
		return err
	}
	for _, device := range devices {
		fmt.Printf("  %s  %s  %s%s%s\n",
			device.Path(), device.Size, device.ModelName(), usbLabel(device.Path()),
			fixedSuffix(device.Fixed()))
	}
	return nil
}

func FlashEnable(ws *Workspace, opts FlashOptions) error {
	if os.Geteuid() != 0 {
		self, err := os.Executable()
		if err != nil {
			return err
		}
		args := []string{self, "flash", "enable"}
		if opts.Device != "" {
			args = append(args, "--device", opts.Device)
		}
		if opts.ImageDir != "" {
			args = append(args, "--image-dir", opts.ImageDir)
		}
		if opts.AllowFixed || (opts.Device != "" && !isRemovable(RealPath(opts.Device))) {
			args = append(args, "--allow-fixed")
		}
		if opts.SkipVerify {
			args = append(args, "--skip-verify")
		}
		return NewRunner(ws.Root).Run("sudo", args...)
	}

	user := os.Getenv("FLASH_USER")
	if user == "" {
		user = os.Getenv("SUDO_USER")
	}
	if user == "" {
		return fmt.Errorf("run this through sudo from your user account")
	}
	device := opts.Device
	if device == "" {
		var err error
		device, err = chooseUSBDevice(true)
		if err != nil {
			return err
		}
	}
	imageDir := opts.ImageDir
	if imageDir == "" {
		imageDir = ws.CachePath("release-dev")
	}
	imageDir = RealPath(ws.ResolvePath(imageDir))
	if _, err := findFlashImage(imageDir); err != nil {
		return err
	}
	allowFixed := opts.AllowFixed || !isRemovable(RealPath(device))

	config := FlashConfig{Device: device, ImageDir: imageDir, AllowFixed: allowFixed, SkipVerify: opts.SkipVerify}
	if err := writeFlashConfig(flashConfigPath, config); err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(flashHelperPath), 0o755); err != nil {
		return err
	}
	if err := CopyFile(self, flashHelperPath); err != nil {
		return err
	}
	if err := os.Chmod(flashHelperPath, 0o755); err != nil {
		return err
	}
	sudoers := fmt.Sprintf("%s ALL=(root) NOPASSWD: %s\n", user, flashHelperPath)
	tmpSudoers := flashSudoersPath + ".tmp"
	if err := os.WriteFile(tmpSudoers, []byte(sudoers), 0o440); err != nil {
		return err
	}
	if err := os.Chmod(tmpSudoers, 0o440); err != nil {
		_ = os.Remove(tmpSudoers)
		return err
	}
	if err := NewRunner(ws.Root).Run("visudo", "-cf", tmpSudoers); err != nil {
		_ = os.Remove(tmpSudoers)
		return err
	}
	if err := os.Rename(tmpSudoers, flashSudoersPath); err != nil {
		_ = os.Remove(tmpSudoers)
		return err
	}
	if err := os.WriteFile(flashMarkerPath, []byte(flashMarkerValue+"\n"), 0o644); err != nil {
		return err
	}
	if err := NewRunner(ws.Root).Run("visudo", "-c"); err != nil {
		return err
	}
	fmt.Printf("Enabled for %s\n", user)
	fmt.Printf("Device: %s\n", device)
	fmt.Printf("Images: %s\n", imageDir)
	fmt.Println("Flash with: mise run flash:usb")
	return nil
}

func FlashDisable(ws *Workspace) error {
	if os.Geteuid() != 0 {
		self, err := os.Executable()
		if err != nil {
			return err
		}
		return NewRunner(ws.Root).Run("sudo", self, "flash", "disable")
	}
	for _, path := range []string{flashSudoersPath, flashSudoersPath + ".tmp", flashHelperPath, flashConfigPath, flashMarkerPath} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := NewRunner(ws.Root).Run("visudo", "-c"); err != nil {
		return err
	}
	fmt.Println("Disabled the passwordless flash helper.")
	return nil
}

func RunFlashHelper() error {
	if len(os.Args) != 1 {
		return fmt.Errorf("usage: slippios-flash")
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("slippios-flash must run as root")
	}
	config, err := readFlashConfig(flashConfigPath)
	if err != nil {
		return fmt.Errorf("missing %s; run mise run flash:usb", flashConfigPath)
	}
	if config.Device == "" || config.ImageDir == "" {
		return fmt.Errorf("device or image_dir is missing from %s", flashConfigPath)
	}
	dev := RealPath(config.Device)
	if _, err := os.Stat(dev); err != nil {
		return fmt.Errorf("target is not present: %s", config.Device)
	}
	if !isBlockDevice(dev) {
		return fmt.Errorf("target is not a block device: %s", dev)
	}
	if !isWholeDisk(dev) {
		return fmt.Errorf("target is not a whole disk: %s", dev)
	}
	if !config.AllowFixed && !isRemovable(dev) {
		return fmt.Errorf("refusing to write a non-removable device: %s\nuse --allow-fixed when enabling the helper for a USB NVMe enclosure", dev)
	}
	transport, err := NewRunner("").Output("lsblk", "-dno", "TRAN", dev)
	if err != nil || strings.TrimSpace(transport) != "usb" {
		return fmt.Errorf("refusing to write a non-USB device: %s", dev)
	}
	image, err := findFlashImage(config.ImageDir)
	if err != nil {
		return err
	}
	imageSize, err := FileSize(image)
	if err != nil {
		return err
	}
	targetSize, err := blockDeviceSize(dev)
	if err != nil {
		return err
	}
	if targetSize < imageSize {
		return fmt.Errorf("target is too small: %d < %d", targetSize, imageSize)
	}
	if err := unmountAll(dev); err != nil {
		return err
	}

	fmt.Printf("Flashing %s\n", image)
	fmt.Printf("Target: %s (%s)\n", dev, HumanSize(targetSize))
	if err := writeImage(image, dev, imageSize); err != nil {
		return err
	}
	syscall.Sync()
	fmt.Println()
	if config.SkipVerify {
		fmt.Println("Skipping verification (skip_verify=true).")
		fmt.Printf("Flashed %s (verification skipped)\n", dev)
		return nil
	}
	fmt.Println("Verifying (reading the image back)…")
	if err := verifyImage(image, dev, imageSize); err != nil {
		return err
	}
	fmt.Printf("Flashed and verified %s\n", dev)
	return nil
}

func findFlashImage(dir string) (string, error) {
	images, err := FindImages(dir, "slippios_*.raw", ".nix-store.raw")
	if err != nil {
		return "", err
	}
	if len(images) != 1 {
		return "", fmt.Errorf("expected exactly one dev image in %s, found %d\nrun: mise run release:dev --keep-raw", dir, len(images))
	}
	return images[0], nil
}

func listUSBDevices() ([]USBDevice, error) {
	out, err := NewRunner("").Output("lsblk", "-J", "-d", "-o", "NAME,SIZE,MODEL,TRAN,RM")
	if err != nil {
		return nil, err
	}
	var value struct {
		Blockdevices []USBDevice `json:"blockdevices"`
	}
	if err := json.Unmarshal([]byte(out), &value); err != nil {
		return nil, err
	}
	var devices []USBDevice
	for _, device := range value.Blockdevices {
		if device.Tran == nil || *device.Tran != "usb" {
			continue
		}
		devices = append(devices, device)
	}
	return devices, nil
}

func chooseUSBDevice(yes bool) (string, error) {
	devices, err := listUSBDevices()
	if err != nil {
		return "", err
	}
	if len(devices) == 0 {
		return "", fmt.Errorf("No USB flash media detected. Connect the drive and retry")
	}
	if len(devices) == 1 {
		device := devices[0]
		fmt.Fprintf(os.Stderr, "Detected USB media: %s  %s  %s%s%s\n",
			device.Path(), device.Size, device.ModelName(), usbLabel(device.Path()),
			fixedSuffix(device.Fixed()))
		return byID(device.Path()), nil
	}
	if yes {
		return "", fmt.Errorf("multiple USB disks found; pass --device")
	}
	fmt.Fprintln(os.Stderr, "USB media:")
	for index, device := range devices {
		fmt.Fprintf(os.Stderr, "  %d) %s  %s  %s%s%s\n",
			index+1, device.Path(), device.Size, device.ModelName(),
			usbLabel(device.Path()), fixedSuffix(device.Fixed()))
	}
	fmt.Fprintf(os.Stderr, "Select device [1-%d] (q to cancel): ", len(devices))
	choice, err := ReadLine(os.Stdin)
	if err != nil {
		return "", err
	}
	if choice == "" || choice == "q" {
		return "", ErrCancelled
	}
	index, err := strconv.Atoi(choice)
	if err != nil || index < 1 || index > len(devices) {
		return "", fmt.Errorf("invalid selection")
	}
	return byID(devices[index-1].Path()), nil
}

func byID(dev string) string {
	target := RealPath(dev)
	links, _ := filepath.Glob("/dev/disk/by-id/usb-*")
	for _, link := range links {
		if RealPath(link) == target {
			return link
		}
	}
	return dev
}

func usbLabel(dev string) string {
	out, err := NewRunner("").Output("udevadm", "info", "--query=property", "--name="+dev)
	if err != nil {
		return ""
	}
	values := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok {
			values[key] = value
		}
	}
	vid := values["ID_VENDOR_ID"]
	mid := values["ID_MODEL_ID"]
	vendor := strings.ReplaceAll(values["ID_VENDOR"], "_", " ")
	model := strings.ReplaceAll(values["ID_MODEL"], "_", " ")
	switch {
	case vid != "" && mid != "" && vendor != "" && model != "":
		return fmt.Sprintf(" [%s %s (%s:%s)]", vendor, model, vid, mid)
	case vid != "" && mid != "":
		return fmt.Sprintf(" [(%s:%s)]", vid, mid)
	default:
		return ""
	}
}

func fixedSuffix(fixed bool) string {
	if fixed {
		return ", fixed"
	}
	return ""
}

func isRemovable(dev string) bool {
	name := filepath.Base(RealPath(dev))
	data, err := os.ReadFile(filepath.Join("/sys/block", name, "removable"))
	return err == nil && strings.TrimSpace(string(data)) == "1"
}

func isBlockDevice(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode()&os.ModeDevice != 0 && info.Mode()&os.ModeCharDevice == 0
}

func isWholeDisk(dev string) bool {
	out, err := NewRunner("").Output("lsblk", "-dno", "TYPE", dev)
	return err == nil && strings.TrimSpace(out) == "disk"
}

func isUSBDevice(dev string) bool {
	out, err := NewRunner("").Output("lsblk", "-dno", "TRAN", dev)
	return err == nil && strings.TrimSpace(out) == "usb"
}

func blockDeviceSize(dev string) (int64, error) {
	out, err := NewRunner("").Output("lsblk", "-b", "-dno", "SIZE", dev)
	if err != nil {
		return 0, err
	}
	return ParseInt(out)
}

func deviceSize(dev string) string {
	out, err := NewRunner("").Output("lsblk", "-dno", "SIZE", dev)
	if err != nil {
		return "?"
	}
	return strings.TrimSpace(out)
}

func deviceModel(dev string) string {
	out, err := NewRunner("").Output("lsblk", "-dno", "MODEL", dev)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func unmountAll(dev string) error {
	out, err := NewRunner("").Output("lsblk", "-ln", "-o", "MOUNTPOINTS", dev)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(out, "\n") {
		mountpoint := strings.TrimSpace(line)
		if mountpoint == "" {
			continue
		}
		fmt.Printf("Unmounting %s\n", mountpoint)
		if err := NewRunner("").Run("umount", mountpoint); err != nil {
			return err
		}
	}
	return nil
}

func writeImage(imagePath, devicePath string, size int64) error {
	image, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer image.Close()
	device, err := os.OpenFile(devicePath, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer device.Close()

	progress := NewProgress(size, "Flashing", os.Stderr)
	buffer := make([]byte, 16<<20)
	var offset int64
	for offset < size {
		want := int64(len(buffer))
		if remaining := size - offset; remaining < want {
			want = remaining
		}
		n, readErr := image.ReadAt(buffer[:want], offset)
		if n > 0 {
			written, writeErr := device.WriteAt(buffer[:n], offset)
			progress.Add(int64(written))
			offset += int64(written)
			if writeErr != nil {
				return writeErr
			}
			if written != n {
				return io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
		if n == 0 {
			return io.ErrNoProgress
		}
	}
	progress.Done()
	if offset != size {
		return fmt.Errorf("wrote %d of %d bytes", offset, size)
	}
	return device.Sync()
}

func verifyImage(imagePath, devicePath string, size int64) error {
	image, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer image.Close()
	device, err := os.Open(devicePath)
	if err != nil {
		return err
	}
	defer device.Close()

	progress := NewProgress(size, "Verifying", os.Stderr)
	imageBuffer := make([]byte, 16<<20)
	deviceBuffer := make([]byte, 16<<20)
	var offset int64
	for offset < size {
		want := int64(len(imageBuffer))
		if remaining := size - offset; remaining < want {
			want = remaining
		}
		imageN, imageErr := image.ReadAt(imageBuffer[:want], offset)
		deviceN, deviceErr := device.ReadAt(deviceBuffer[:want], offset)
		if imageN != deviceN {
			return fmt.Errorf("verification failed at offset %d: read %d image bytes and %d device bytes", offset, imageN, deviceN)
		}
		if imageN > 0 && !bytes.Equal(imageBuffer[:imageN], deviceBuffer[:deviceN]) {
			return fmt.Errorf("verification failed at offset %d", offset)
		}
		progress.Add(int64(imageN))
		offset += int64(imageN)
		if imageErr == io.EOF || deviceErr == io.EOF {
			break
		}
		if imageErr != nil {
			return imageErr
		}
		if deviceErr != nil {
			return deviceErr
		}
		if imageN == 0 {
			return io.ErrNoProgress
		}
	}
	progress.Done()
	if offset != size {
		return fmt.Errorf("verified %d of %d bytes", offset, size)
	}
	return nil
}

func configureFlash(ws *Workspace, device, imageDir string, allowFixed, skipVerify bool) error {
	config, err := readFlashConfig(flashConfigPath)
	if err == nil &&
		config.Device == device &&
		config.ImageDir == imageDir &&
		config.AllowFixed == allowFixed &&
		config.SkipVerify == skipVerify &&
		fileExists(flashHelperPath) &&
		readMarker() == flashMarkerValue {
		return nil
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	installer := ws.CachePath("bin", "slippios-dev")
	if err := os.MkdirAll(filepath.Dir(installer), 0o755); err != nil {
		return err
	}
	if err := CopyFile(self, installer); err != nil {
		return err
	}
	if err := os.Chmod(installer, 0o755); err != nil {
		return err
	}
	args := []string{installer, "flash", "enable", "--device", device, "--image-dir", imageDir}
	if allowFixed {
		args = append(args, "--allow-fixed")
	}
	if skipVerify {
		args = append(args, "--skip-verify")
	}
	return NewRunner(ws.Root).Run("sudo", args...)
}

func runFlashHelper() error {
	return NewRunner("").Run("sudo", "-n", flashHelperPath)
}

func readFlashConfig(path string) (FlashConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FlashConfig{}, err
	}
	var file flashConfigFile
	if err := toml.Unmarshal(data, &file); err == nil && (file.Device != "" || file.ImageDir != "") {
		return FlashConfig{Device: file.Device, ImageDir: file.ImageDir, AllowFixed: file.AllowFixed, SkipVerify: file.SkipVerify}, nil
	}
	return parseShellFlashConfig(string(data))
}

func parseShellFlashConfig(data string) (FlashConfig, error) {
	var config FlashConfig
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		switch key {
		case "device":
			config.Device = value
		case "image_dir":
			config.ImageDir = value
		case "allow_fixed":
			config.AllowFixed = value == "1" || strings.EqualFold(value, "true")
		case "skip_verify":
			config.SkipVerify = value == "1" || strings.EqualFold(value, "true")
		}
	}
	if config.Device == "" && config.ImageDir == "" {
		return FlashConfig{}, fmt.Errorf("cannot parse flash config")
	}
	return config, nil
}

func writeFlashConfig(path string, config FlashConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := toml.Marshal(flashConfigFile{
		Device:     config.Device,
		ImageDir:   config.ImageDir,
		AllowFixed: config.AllowFixed,
		SkipVerify: config.SkipVerify,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readMarker() string {
	data, err := os.ReadFile(flashMarkerPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func installedLabel(path string) string {
	if fileExists(path) {
		return "installed"
	}
	return "missing"
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
