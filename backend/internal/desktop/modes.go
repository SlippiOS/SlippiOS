package desktop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type ModeRate struct {
	Resolution string    `json:"resolution"`
	Rates      []float64 `json:"rates"`
	Current    bool      `json:"current"`
	Preferred  bool      `json:"preferred"`
}

type DisplayModes struct {
	Output   string     `json:"output"`
	Current  string     `json:"current"`
	Rate     float64    `json:"rate"`
	EDIDName string     `json:"edid_name,omitempty"`
	Modes    []ModeRate `json:"modes"`
}

func Modes() (DisplayModes, error) {
	var info DisplayModes
	query, err := exec.Command("xrandr", "--query").Output()
	if err != nil {
		return info, err
	}
	lines := strings.Split(string(query), "\n")
	found := false
	for _, line := range lines {
		if !found {
			if !strings.Contains(line, " connected") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			info.Output = fields[0]
			for _, field := range fields {
				if strings.Contains(field, "x") && strings.Contains(field, "+") {
					geometry := strings.SplitN(field, "+", 2)[0]
					if strings.Count(geometry, "x") == 1 {
						info.Current = geometry
					}
				}
			}
			found = true
			continue
		}
		if strings.TrimSpace(line) == "" || !strings.HasPrefix(line, " ") {
			break
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.Count(fields[0], "x") != 1 {
			continue
		}
		mode := ModeRate{Resolution: fields[0]}
		for _, field := range fields[1:] {
			value := strings.TrimRight(field, "*+")
			if value == "" {
				continue
			}
			rate, err := strconv.ParseFloat(value, 64)
			if err != nil {
				continue
			}
			mode.Rates = append(mode.Rates, rate)
			if strings.Contains(field, "*") {
				mode.Current = true
				info.Rate = rate
			}
			if strings.Contains(field, "+") {
				mode.Preferred = true
			}
		}
		info.Modes = append(info.Modes, mode)
	}
	if !found {
		return info, fmt.Errorf("no connected display found")
	}
	if info.Current == "" {
		for _, mode := range info.Modes {
			if mode.Current {
				info.Current = mode.Resolution
				break
			}
		}
	}
	info.EDIDName = edidName(info.Output)
	return info, nil
}

func CurrentResolution() (int, int, error) {
	info, err := Modes()
	if err != nil {
		return 0, 0, err
	}
	var width, height int
	if _, err := fmt.Sscanf(info.Current, "%dx%d", &width, &height); err != nil {
		return 0, 0, err
	}
	if width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("invalid display resolution %q", info.Current)
	}
	return width, height, nil
}

func edidName(output string) string {
	matches, err := filepath.Glob("/sys/class/drm/card*-" + output + "/edid")
	if err != nil || len(matches) == 0 {
		return ""
	}
	data, err := os.ReadFile(matches[0])
	if err != nil || len(data) < 128 {
		return ""
	}
	if data[0] != 0x00 || data[1] != 0xff {
		return ""
	}
	for offset := 54; offset+18 <= 126; offset += 18 {
		if data[offset] != 0 || data[offset+1] != 0 || data[offset+2] != 0 {
			continue
		}
		if data[offset+3] != 0xfc {
			continue
		}
		text := strings.TrimSpace(strings.TrimRight(string(data[offset+5:offset+18]), "\x00"))
		return strings.TrimSpace(text)
	}
	return ""
}
