// Package gecko reads and writes Dolphin's Gecko code configuration for
// Melee. It mirrors the Slippi Launcher: global codes come from the Slippi
// Dolphin Sys directory, while user overrides and custom codes live in the
// profile's GameSettings/GALE01.ini.
package gecko

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SlippiOS/SlippiOS/backend/internal/ini"
)

type Code struct {
	Name           string   `json:"name"`
	Creator        string   `json:"creator"`
	Notes          []string `json:"notes"`
	Lines          []string `json:"-"`
	Enabled        bool     `json:"enabled"`
	DefaultEnabled bool     `json:"default_enabled"`
	UserDefined    bool     `json:"user_defined"`
	Required       bool     `json:"required"`
}

func LocalPath(profileDir string) string {
	return filepath.Join(profileDir, "dolphin", "GameSettings", "GALE01.ini")
}

// GlobalPath returns the GALE01 code list shipped inside the extracted Slippi
// Dolphin AppImage.
func GlobalPath(profileDir string) (string, error) {
	pattern := filepath.Join(profileDir, "cache", "appimage-run", "*", "usr", "bin", "Sys",
		"GameSettings", "GALE01r2.ini")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("Slippi Dolphin is not prepared yet; launch Melee once")
	}
	sort.Slice(matches, func(i, j int) bool {
		left, leftErr := os.Stat(matches[i])
		right, rightErr := os.Stat(matches[j])
		if leftErr != nil || rightErr != nil {
			return matches[i] > matches[j]
		}
		return left.ModTime().After(right.ModTime())
	})
	return matches[0], nil
}

func Load(profileDir string) ([]Code, error) {
	globalPath, err := GlobalPath(profileDir)
	if err != nil {
		return nil, err
	}
	global, err := ini.Load(globalPath)
	if err != nil {
		return nil, err
	}
	local, err := ini.Load(LocalPath(profileDir))
	if err != nil {
		return nil, err
	}

	codes := parseCodes(global.Lines("Gecko"), false)
	applyEnabled(global.Lines("Gecko_Enabled"), true, codes)
	applyEnabled(global.Lines("Gecko_Disabled"), false, codes)
	for i := range codes {
		codes[i].DefaultEnabled = codes[i].Enabled
	}

	codes = append(codes, parseCodes(local.Lines("Gecko"), true)...)
	applyEnabled(local.Lines("Gecko_Enabled"), true, codes)
	applyEnabled(local.Lines("Gecko_Disabled"), false, codes)

	for i := range codes {
		codes[i].Required = strings.HasPrefix(codes[i].Name, "Required:")
	}
	return codes, nil
}

func SetEnabled(profileDir, name string, enabled bool) error {
	codes, err := Load(profileDir)
	if err != nil {
		return err
	}
	index := find(codes, name)
	if index < 0 {
		return fmt.Errorf("unknown Gecko code %q", name)
	}
	if codes[index].Required && !enabled {
		return fmt.Errorf("%q is required and cannot be disabled", name)
	}
	codes[index].Enabled = enabled
	return save(profileDir, codes)
}

func Add(profileDir, text string) ([]Code, error) {
	added := parseCodes(strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n"), true)
	if len(added) == 0 {
		return nil, fmt.Errorf("no Gecko codes found")
	}
	codes, err := Load(profileDir)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, code := range codes {
		names[code.Name] = true
	}
	for i := range added {
		if added[i].Name == "" {
			return nil, fmt.Errorf("every Gecko code needs a $name line")
		}
		if names[added[i].Name] {
			return nil, fmt.Errorf("a Gecko code named %q already exists", added[i].Name)
		}
		names[added[i].Name] = true
		added[i].Enabled = true
		added[i].DefaultEnabled = false
		added[i].UserDefined = true
	}
	codes = append(codes, added...)
	if err := save(profileDir, codes); err != nil {
		return nil, err
	}
	return added, nil
}

func Remove(profileDir, name string) error {
	codes, err := Load(profileDir)
	if err != nil {
		return err
	}
	index := find(codes, name)
	if index < 0 {
		return fmt.Errorf("unknown Gecko code %q", name)
	}
	if !codes[index].UserDefined {
		return fmt.Errorf("%q is built in and cannot be removed", name)
	}
	if codes[index].Required {
		return fmt.Errorf("%q is required and cannot be removed", name)
	}
	codes = append(codes[:index], codes[index+1:]...)
	return save(profileDir, codes)
}

const QuitCodeName = "Required: SlippiOS Quit Button"

// The main menu's Data slot is repurposed as Quit. The code runs from the
// main menu think function, checks the top-level Data selection, sends a
// SLIPPIOS_QUIT message over the Slippi EXI device, and clears the button
// mask so Melee does not open the Data menu.
const quitCode = `C222DB40 00000019
9421FFE0 7C0802A6
9001001C 90610018
90810014 90A10010
3BC00000 70600010
4182008C 881C0000
2C000000 40820080
A01C0002 2C000004
40820074 806DAFC4
380000D0 98030000
38000000 98030001
38000004 98030002
38000053 98030003
3C004C49 60005050
90030004 3C00494F
6000535F 90030008
3C005155 60004954
9003000C 38000000
98030010 38800080
38A00001 3D808000
618C55F0 7D8903A6
4E800421 38000000
901C000C 80610018
80810014 80A10010
8001001C 7C0803A6
38210020 4E800020`

// EnsureQuitCode adds the built-in Quit code to the profile's local GALE01
// settings the first time it is needed. A code that already exists is left
// alone.
func EnsureQuitCode(profileDir string) error {
	path := LocalPath(profileDir)
	local, err := ini.Load(path)
	if err != nil {
		return err
	}
	nameLine := "$" + QuitCodeName
	lines := local.Lines("Gecko")
	found := false
	for _, line := range lines {
		if strings.TrimSpace(line) == nameLine {
			found = true
			break
		}
	}
	if !found {
		lines = append(append([]string(nil), lines...), nameLine)
		for _, line := range strings.Split(quitCode, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				lines = append(lines, line)
			}
		}
		local.SetLines("Gecko", lines)
	}
	if !hasLine(local.Lines("Gecko_Enabled"), nameLine) {
		local.SetLines("Gecko_Enabled", append(local.Lines("Gecko_Enabled"), nameLine))
	}
	if hasLine(local.Lines("Gecko_Disabled"), nameLine) {
		disabled := local.Lines("Gecko_Disabled")
		filtered := make([]string, 0, len(disabled))
		for _, line := range disabled {
			if strings.TrimSpace(line) != nameLine {
				filtered = append(filtered, line)
			}
		}
		local.SetLines("Gecko_Disabled", filtered)
	}
	return ini.Save(path, local)
}

func hasLine(lines []string, target string) bool {
	for _, line := range lines {
		if strings.TrimSpace(line) == target {
			return true
		}
	}
	return false
}

func save(profileDir string, codes []Code) error {
	path := LocalPath(profileDir)
	local, err := ini.Load(path)
	if err != nil {
		return err
	}

	var geckoLines []string
	for _, code := range codes {
		if !code.UserDefined {
			continue
		}
		geckoLines = append(geckoLines, title(code))
		for _, note := range code.Notes {
			geckoLines = append(geckoLines, "*"+note)
		}
		geckoLines = append(geckoLines, code.Lines...)
	}
	local.SetLines("Gecko", geckoLines)

	var enabled []string
	var disabled []string
	for _, code := range codes {
		if code.Enabled == code.DefaultEnabled {
			continue
		}
		line := "$" + code.Name
		if code.Enabled {
			enabled = append(enabled, line)
		} else {
			disabled = append(disabled, line)
		}
	}
	local.SetLines("Gecko_Enabled", enabled)
	local.SetLines("Gecko_Disabled", disabled)
	return ini.Save(path, local)
}

func parseCodes(lines []string, userDefined bool) []Code {
	var codes []Code
	var current *Code
	flush := func() {
		if current != nil {
			codes = append(codes, *current)
			current = nil
		}
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "$") {
			flush()
			name, creator := parseTitle(line[1:])
			current = &Code{Name: name, Creator: creator, UserDefined: userDefined}
			continue
		}
		if current == nil {
			continue
		}
		if strings.HasPrefix(line, "*") {
			current.Notes = append(current.Notes, strings.TrimSpace(line[1:]))
			continue
		}
		current.Lines = append(current.Lines, line)
	}
	flush()
	return codes
}

func parseTitle(text string) (string, string) {
	text = strings.TrimSpace(text)
	start := strings.Index(text, "[")
	end := strings.Index(text, "]")
	if start < 0 || end < start {
		return text, ""
	}
	return strings.TrimSpace(text[:start]), strings.TrimSpace(text[start+1 : end])
}

func title(code Code) string {
	if code.Creator == "" {
		return "$" + code.Name
	}
	return "$" + code.Name + " [" + code.Creator + "]"
}

func applyEnabled(lines []string, enabled bool, codes []Code) {
	for _, line := range lines {
		if !strings.HasPrefix(line, "$") {
			continue
		}
		name := strings.TrimSpace(line[1:])
		for i := range codes {
			if codes[i].Name == name {
				codes[i].Enabled = enabled
			}
		}
	}
}

func find(codes []Code, name string) int {
	for i := range codes {
		if codes[i].Name == name {
			return i
		}
	}
	return -1
}
