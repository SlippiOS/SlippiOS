package network

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInterfaceKind(t *testing.T) {
	root := t.TempDir()
	if got := interfaceKind(root, "enp1s0"); got != "ethernet" {
		t.Fatalf("enp1s0 = %q", got)
	}
	if got := interfaceKind(root, "wlp2s0"); got != "wifi" {
		t.Fatalf("wlp2s0 = %q", got)
	}
	if got := interfaceKind(root, "docker0"); got != "other" {
		t.Fatalf("docker0 = %q", got)
	}
	if err := os.MkdirAll(filepath.Join(root, "usb0", "wireless"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := interfaceKind(root, "usb0"); got != "wifi" {
		t.Fatalf("usb0 with wireless = %q", got)
	}
}

func TestLinkState(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "enp1s0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "enp1s0", "carrier"), []byte("1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !linkState(root, "enp1s0") {
		t.Fatal("carrier 1 should be up")
	}
	if err := os.WriteFile(filepath.Join(root, "enp1s0", "carrier"), []byte("0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if linkState(root, "enp1s0") {
		t.Fatal("carrier 0 should be down")
	}
}
