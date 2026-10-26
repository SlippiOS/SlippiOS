// Package desktop applies the display and audio preferences from settings to
// the running X/PulseAudio session.
package desktop

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ApplyDisplay sets the configured resolution and refresh rate on the
// connected output. A mode the display does not advertise is ignored; the
// preference still applies on the next boot.
func ApplyDisplay(values map[string]any) error {
	if _, err := exec.LookPath("xrandr"); err != nil {
		return nil
	}
	query, err := exec.Command("xrandr", "--query").Output()
	if err != nil {
		return err
	}
	output := ""
	for _, entry := range strings.Split(string(query), "\n") {
		if strings.Contains(entry, " connected") {
			fields := strings.Fields(entry)
			if len(fields) > 0 {
				output = fields[0]
			}
			if strings.Contains(entry, " primary") {
				break
			}
		}
	}
	if output == "" {
		return nil
	}

	display, _ := values["display"].(map[string]any)
	resolution := fmt.Sprint(display["resolution"])
	rate := display["refresh_hz"]
	arguments := []string{"--output", output}
	if resolution != "" && resolution != "auto" {
		arguments = append(arguments, "--mode", resolution)
	}
	if rate != nil && fmt.Sprint(rate) != "" && fmt.Sprint(rate) != "0" {
		arguments = append(arguments, "--rate", fmt.Sprint(rate))
	}
	if len(arguments) == 2 {
		return nil
	}
	if output, err := exec.Command("xrandr", arguments...).CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("%s", message)
	}
	return nil
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
