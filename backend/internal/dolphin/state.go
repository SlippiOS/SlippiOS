package dolphin

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func (p Paths) CurrentPath() (string, bool) {
	target, err := os.Readlink(p.Current)
	if err != nil {
		return "", false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(p.Current), target)
	}
	info, err := os.Stat(target)
	if err != nil || info.IsDir() {
		return "", false
	}
	return target, true
}

func (p Paths) CurrentTag() (string, bool) {
	if path, ok := p.CurrentPath(); ok {
		return filepath.Base(filepath.Dir(path)), true
	}
	if info, err := os.Stat(BundledPath()); err == nil && !info.IsDir() {
		return bundledTag, true
	}
	return "", false
}

func readJSON(path string) map[string]any {
	value := map[string]any{}
	data, err := os.ReadFile(path)
	if err != nil {
		return value
	}
	_ = json.Unmarshal(data, &value)
	return value
}

func writeJSON(path string, value map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func checkedAt(state map[string]any) int64 {
	value, ok := state["checked_at"].(float64)
	if !ok {
		return 0
	}
	return int64(value)
}
