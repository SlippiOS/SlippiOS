// Package desktop applies the display and audio preferences from settings to
// the running X/PulseAudio session.
package desktop

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ApplyDisplay selects the configured mode on the connected output. A custom
// mode is re-added first because X forgets runtime modes on restart.
func ApplyDisplay(values map[string]any) error {
	if _, err := exec.LookPath("xrandr"); err != nil {
		return nil
	}
	display, _ := values["display"].(map[string]any)
	if custom, _ := display["custom"].(string); custom != "" {
		return ApplyCustomMode(custom)
	}
	resolution := fmt.Sprint(display["resolution"])
	if resolution == "" || resolution == "auto" {
		return nil
	}
	rate, _ := display["rate"].(string)
	if rate == "" {
		if hz := display["refresh_hz"]; hz != nil && fmt.Sprint(hz) != "0" {
			rate = fmt.Sprint(hz)
		}
	}
	return SetMode(resolution, rate)
}

// SetMode switches the connected output to an advertised resolution and rate.
func SetMode(resolution, rate string) error {
	output, err := connectedOutput()
	if err != nil {
		return err
	}
	arguments := []string{"--output", output, "--mode", resolution}
	if rate != "" {
		arguments = append(arguments, "--rate", rate)
	}
	return runXrandr(arguments)
}

func runXrandr(arguments []string) error {
	if output, err := exec.Command("xrandr", arguments...).CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("%s", message)
	}
	return nil
}

func connectedOutput() (string, error) {
	query, err := exec.Command("xrandr", "--query").Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(query), "\n") {
		if !strings.Contains(line, " connected") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no connected display found")
}

// ApplyAudio selects the configured PulseAudio sink.
func ApplyAudio(values map[string]any) error {
	audio, _ := values["audio"].(map[string]any)
	sink, _ := audio["sink"].(string)
	if sink == "" || sink == "auto" {
		return nil
	}
	if _, err := exec.LookPath("pactl"); err != nil {
		return nil
	}
	return exec.Command("pactl", "set-default-sink", sink).Run()
}

// Sink describes one PulseAudio output.
type Sink struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type pactlSink struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Sinks lists the available PulseAudio outputs.
func Sinks() ([]Sink, error) {
	if _, err := exec.LookPath("pactl"); err != nil {
		return nil, fmt.Errorf("pactl is not installed")
	}
	output, err := exec.Command("pactl", "-f", "json", "list", "sinks").Output()
	if err != nil {
		return nil, err
	}
	var raw []pactlSink
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, err
	}
	sinks := make([]Sink, 0, len(raw))
	for _, sink := range raw {
		sinks = append(sinks, Sink{Name: sink.Name, Description: sink.Description})
	}
	return sinks, nil
}

// ModeRate is one resolution with the refresh rates the display advertises.
type ModeRate struct {
	Resolution string    `json:"resolution"`
	Rates      []float64 `json:"rates"`
	Current    bool      `json:"current"`
	Preferred  bool      `json:"preferred"`
}

// DisplayModes describes the connected output and its advertised modes.
type DisplayModes struct {
	Output   string     `json:"output"`
	Current  string     `json:"current"`
	Rate     float64    `json:"rate"`
	EDIDName string     `json:"edid_name,omitempty"`
	Modes    []ModeRate `json:"modes"`
}

// Modes lists the modes from xrandr, which is the driver's view of the monitor
// EDID. EDID is read only for the human-readable monitor name.
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

// edidName reads the monitor name descriptor from the connector's EDID.
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

// ApplyCustomMode adds and selects a CVT mode that the display may not list.
func ApplyCustomMode(spec string) error {
	width, height, rate, err := parseModeSpec(spec)
	if err != nil {
		return err
	}
	output, err := connectedOutput()
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%dx%d_%g", width, height, rate)
	cvt, err := exec.Command("cvt", strconv.Itoa(width), strconv.Itoa(height),
		fmt.Sprintf("%g", rate)).Output()
	if err != nil {
		return err
	}
	line := ""
	for _, candidate := range strings.Split(string(cvt), "\n") {
		if strings.Contains(candidate, "Modeline") {
			line = candidate
			break
		}
	}
	if line == "" {
		return fmt.Errorf("cvt produced no modeline")
	}
	modeline := strings.Fields(line)
	if len(modeline) < 3 || modeline[0] != "Modeline" {
		return fmt.Errorf("unexpected cvt output")
	}
	modeline = modeline[1:]
	modeline[0] = name
	args := append([]string{"--newmode"}, modeline...)
	if output, err := exec.Command("xrandr", args...).CombinedOutput(); err != nil &&
		!strings.Contains(string(output), "already exists") {
		return fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}
	if output, err := exec.Command("xrandr", "--addmode", output, name).CombinedOutput(); err != nil &&
		!strings.Contains(string(output), "already exists") {
		return fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}
	if output, err := exec.Command("xrandr", "--output", output, "--mode", name, "--rate",
		fmt.Sprintf("%g", rate)).CombinedOutput(); err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}
	return nil
}

func parseModeSpec(spec string) (int, int, float64, error) {
	rate := 60.0
	if at := strings.LastIndex(spec, "@"); at >= 0 {
		value, err := strconv.ParseFloat(spec[at+1:], 64)
		if err != nil || value <= 0 {
			return 0, 0, 0, fmt.Errorf("invalid refresh rate")
		}
		rate = value
		spec = spec[:at]
	}
	parts := strings.SplitN(spec, "x", 2)
	if len(parts) != 2 {
		return 0, 0, 0, fmt.Errorf("use WIDTHxHEIGHT or WIDTHxHEIGHT@RATE")
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil || width < 320 {
		return 0, 0, 0, fmt.Errorf("invalid width")
	}
	height, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || height < 240 {
		return 0, 0, 0, fmt.Errorf("invalid height")
	}
	return width, height, rate, nil
}
