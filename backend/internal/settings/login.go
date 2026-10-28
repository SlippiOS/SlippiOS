package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func (p Paths) LoginStatus(path string) (bool, string) {
	candidate := path
	if !filepath.IsAbs(candidate) {
		candidate = p.Resolve(candidate)
	}
	data, err := os.ReadFile(candidate)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, fmt.Sprintf("not found: %s", candidate)
		}
		return false, fmt.Sprintf("cannot read %s: %v", candidate, err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return false, fmt.Sprintf("cannot read %s: %v", candidate, err)
	}
	for _, key := range []string{"uid", "playKey", "connectCode"} {
		if value[key] == nil || value[key] == "" {
			return false, fmt.Sprintf("%s is missing %s", candidate, key)
		}
	}
	return true, fmt.Sprint(value["connectCode"])
}

func (p Paths) InstallLogin(values map[string]any) (bool, string) {
	configured := DefaultUserJSON
	if slippi, ok := values["slippi"].(map[string]any); ok {
		if value, ok := slippi["user_json"].(string); ok && value != "" {
			configured = value
		}
	}
	candidates := []string{
		p.Resolve(configured),
		filepath.Join(p.ProfileDir, "user.json"),
		filepath.Join(p.DataDir, "user.json"),
	}
	for _, candidate := range candidates {
		ok, detail := p.LoginStatus(candidate)
		if !ok {
			continue
		}
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		// Slippi Dolphin's Rust user parser requires the latestVersion field
		// (the struct field is latest_version with a camelCase alias). The
		// login flow does not know the latest release yet, so seed it empty.
		var value map[string]any
		if err := json.Unmarshal(data, &value); err != nil {
			continue
		}
		if _, ok := value["latestVersion"]; !ok {
			value["latestVersion"] = ""
		}
		data, err = json.MarshalIndent(value, "", "  ")
		if err != nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p.DolphinUserFile), 0o755); err != nil {
			return false, fmt.Sprintf("cannot create %s: %v", filepath.Dir(p.DolphinUserFile), err)
		}
		if err := os.WriteFile(p.DolphinUserFile, data, 0o600); err != nil {
			return false, fmt.Sprintf("cannot write %s: %v", p.DolphinUserFile, err)
		}
		_ = os.Chmod(p.DolphinUserFile, 0o600)
		if current, ok := HardwareID(); ok {
			if err := os.MkdirAll(filepath.Dir(p.HardwareFile), 0o755); err == nil {
				_ = os.WriteFile(p.HardwareFile, []byte(current+"\n"), 0o644)
			}
		}
		return true, detail
	}
	return false, fmt.Sprintf("no Slippi login file at %s", p.Resolve(configured))
}
