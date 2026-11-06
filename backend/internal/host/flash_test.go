package host

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestParseShellFlashConfig(t *testing.T) {
	config, err := parseShellFlashConfig(`# old helper config
device="/dev/disk/by-id/usb-test-0:0"
image_dir="/home/user/.cache/release-dev"
allow_fixed="1"
`)
	if err != nil {
		t.Fatal(err)
	}
	if config.Device != "/dev/disk/by-id/usb-test-0:0" {
		t.Fatalf("device = %q", config.Device)
	}
	if config.ImageDir != "/home/user/.cache/release-dev" {
		t.Fatalf("image_dir = %q", config.ImageDir)
	}
	if !config.AllowFixed {
		t.Fatal("allow_fixed was not parsed")
	}
}

func TestReadFlashConfigTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flash.conf")
	if err := writeFlashConfig(path, FlashConfig{
		Device:     "/dev/sda",
		ImageDir:   "/tmp/images",
		AllowFixed: true,
	}); err != nil {
		t.Fatal(err)
	}
	config, err := readFlashConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if config != (FlashConfig{Device: "/dev/sda", ImageDir: "/tmp/images", AllowFixed: true}) {
		t.Fatalf("config = %#v", config)
	}
}

func TestFindFlashImage(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"slippios_0.4.2.1-signed.raw",
		"slippios_0.4.2.1.nix-store.raw",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	image, err := findFlashImage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(image) != "slippios_0.4.2.1-signed.raw" {
		t.Fatalf("image = %q", image)
	}
}

func TestWriteAndVerifyImage(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "image.raw")
	devicePath := filepath.Join(dir, "device.raw")
	payload := bytes.Repeat([]byte("slippios"), 200_000)
	if err := os.WriteFile(imagePath, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(devicePath, make([]byte, len(payload)+1024), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeImage(imagePath, devicePath, int64(len(payload))); err != nil {
		t.Fatal(err)
	}
	if err := verifyImage(imagePath, devicePath, int64(len(payload))); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(devicePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got[:len(payload)], payload) {
		t.Fatal("device contents do not match the image")
	}
}

func TestManifestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	manifest := Manifest{
		Version:  "0.4.2.1",
		NixStore: &Asset{Name: "store.raw", SHA256: "abc", Size: 10},
	}
	if err := WriteManifest(path, manifest); err != nil {
		t.Fatal(err)
	}
	got, err := ReadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != manifest.Version || got.NixStore == nil || got.NixStore.Name != "store.raw" {
		t.Fatalf("manifest = %#v", got)
	}
}

func TestReadEnvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r2.env")
	data := "export R2_BUCKET=\"slippios\"\nR2_PUBLIC_URL='https://example.test'\n# comment\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	values, err := readEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if values["R2_BUCKET"] != "slippios" || values["R2_PUBLIC_URL"] != "https://example.test" {
		t.Fatalf("values = %#v", values)
	}
}
