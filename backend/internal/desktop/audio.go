package desktop

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

func ApplyAudio(values map[string]any) error {
	audio, _ := values["audio"].(map[string]any)
	sink, _ := audio["sink"].(string)
	volume := 100
	switch value := audio["volume"].(type) {
	case int64:
		volume = int(value)
	case int:
		volume = value
	case float64:
		volume = int(value)
	}
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	if _, err := exec.LookPath("pactl"); err != nil {
		return nil
	}
	if sink != "" && sink != "auto" {
		if err := exec.Command("pactl", "set-default-sink", sink).Run(); err != nil {
			return err
		}
	}
	return exec.Command("pactl", "set-sink-volume", "@DEFAULT_SINK@",
		fmt.Sprintf("%d%%", volume)).Run()
}

type Sink struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type pactlSink struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

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
