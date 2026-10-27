package settings

import (
	"os"
	"path/filepath"
	"strings"
)

var hardwareIDPaths = []string{
	"/sys/class/dmi/id/product_uuid",
	"/sys/class/dmi/id/board_serial",
}

var placeholderHardwareIDs = map[string]bool{
	"":                                     true,
	"none":                                 true,
	"not specified":                        true,
	"to be filled by o.e.m.":               true,
	"03000200-0400-0500-0006-000700080009": true,
}

func HardwareID() (string, bool) {
	for _, path := range hardwareIDPaths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		value := strings.TrimSpace(string(raw))
		if !placeholderHardwareIDs[strings.ToLower(value)] {
			return value, true
		}
	}
	return "", false
}

func (p Paths) CheckHardwareBinding() bool {
	current, ok := HardwareID()
	if !ok {
		return false
	}
	stored, err := os.ReadFile(p.HardwareFile)
	previous := strings.TrimSpace(string(stored))
	if err != nil || previous == "" {
		if err := os.MkdirAll(filepath.Dir(p.HardwareFile), 0o755); err == nil {
			_ = os.WriteFile(p.HardwareFile, []byte(current+"\n"), 0o644)
		}
		return false
	}
	if previous == current {
		return false
	}
	_ = os.Remove(filepath.Join(p.ProfileDir, DefaultUserJSON))
	_ = os.Remove(p.DolphinUserFile)
	_ = os.Remove(p.HardwareFile)
	return true
}
