package desktop

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

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
