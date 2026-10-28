// Package network reports wired and wireless interface state for onboarding.
package network

import (
	"net"
	"os"
	"path/filepath"
	"strings"
)

const sysRoot = "/sys/class/net"

type Interface struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Link    bool   `json:"link"`
	Address string `json:"address,omitempty"`
	Default bool   `json:"default"`
}

type Status struct {
	Address    string      `json:"address"`
	Wired      bool        `json:"wired"`
	WiredLink  bool        `json:"wired_link"`
	Wireless   bool        `json:"wireless"`
	Interfaces []Interface `json:"interfaces"`
}

func StatusNow() Status {
	interfaces, err := net.Interfaces()
	if err != nil {
		return Status{}
	}

	status := Status{}
	defaultAddress := defaultRouteAddress()
	status.Address = defaultAddress
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		kind := interfaceKind(sysRoot, iface.Name)
		address := interfaceAddress(iface)
		entry := Interface{
			Name:    iface.Name,
			Kind:    kind,
			Link:    linkState(sysRoot, iface.Name),
			Address: address,
			Default: address != "" && address == defaultAddress,
		}
		switch kind {
		case "ethernet":
			status.Wired = true
			if entry.Link {
				status.WiredLink = true
			}
		case "wifi":
			status.Wireless = true
		}
		if status.Address == "" && address != "" {
			status.Address = address
		}
		status.Interfaces = append(status.Interfaces, entry)
	}
	return status
}

func interfaceKind(root, name string) string {
	if _, err := os.Stat(filepath.Join(root, name, "wireless")); err == nil {
		return "wifi"
	}
	if strings.HasPrefix(name, "wl") {
		return "wifi"
	}
	if strings.HasPrefix(name, "en") || strings.HasPrefix(name, "eth") {
		return "ethernet"
	}
	if _, err := os.Stat(filepath.Join(root, name, "device")); err == nil {
		return "ethernet"
	}
	return "other"
}

func linkState(root, name string) bool {
	if data, err := os.ReadFile(filepath.Join(root, name, "carrier")); err == nil {
		return strings.TrimSpace(string(data)) == "1"
	}
	if data, err := os.ReadFile(filepath.Join(root, name, "operstate")); err == nil {
		state := strings.TrimSpace(string(data))
		return state == "up" || state == "unknown"
	}
	return false
}

func interfaceAddress(iface net.Interface) string {
	addresses, err := iface.Addrs()
	if err != nil {
		return ""
	}
	for _, address := range addresses {
		var ip net.IP
		switch value := address.(type) {
		case *net.IPNet:
			ip = value.IP
		case *net.IPAddr:
			ip = value.IP
		}
		if ip == nil || ip.To4() == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		return ip.String()
	}
	return ""
}

func defaultRouteAddress() string {
	connection, err := net.Dial("udp4", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer connection.Close()
	if address, ok := connection.LocalAddr().(*net.UDPAddr); ok && address.IP != nil {
		return address.IP.String()
	}
	return ""
}
