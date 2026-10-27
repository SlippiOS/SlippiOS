// Package wifi wraps the NetworkManager CLI for the console setup screen.
package wifi

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

type Network struct {
	SSID     string `json:"ssid"`
	Signal   int    `json:"signal"`
	Security string `json:"security"`
	Active   bool   `json:"active"`
}

type Status struct {
	Connected bool   `json:"connected"`
	SSID      string `json:"ssid,omitempty"`
}

func nmcli(args ...string) (string, error) {
	path, err := exec.LookPath("nmcli")
	if err != nil {
		return "", errors.New("NetworkManager CLI is not available")
	}
	command := exec.Command(path, args...)
	command.Env = append(os.Environ(), "LC_ALL=C")
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", errors.New(message)
	}
	return string(output), nil
}

func splitTerse(line string) []string {
	fields := []string{}
	current := strings.Builder{}
	escaped := false
	for _, character := range line {
		switch {
		case escaped:
			current.WriteRune(character)
			escaped = false
		case character == '\\':
			escaped = true
		case character == ':':
			fields = append(fields, current.String())
			current.Reset()
		default:
			current.WriteRune(character)
		}
	}
	fields = append(fields, current.String())
	return fields
}

func Scan() ([]Network, error) {
	output, err := nmcli("-t", "-f", "IN-USE,SSID,SIGNAL,SECURITY",
		"device", "wifi", "list", "--rescan", "yes")
	if err != nil {
		return nil, err
	}
	networks := map[string]Network{}
	for _, line := range strings.Split(output, "\n") {
		fields := splitTerse(line)
		if len(fields) < 4 || fields[1] == "" {
			continue
		}
		signal, _ := strconv.Atoi(fields[2])
		security := strings.TrimSpace(fields[3])
		if security == "--" {
			security = ""
		}
		candidate := Network{
			SSID:     fields[1],
			Signal:   signal,
			Security: security,
			Active:   strings.TrimSpace(fields[0]) == "*",
		}
		existing, ok := networks[candidate.SSID]
		if !ok || candidate.Signal > existing.Signal || (candidate.Active && !existing.Active) {
			networks[candidate.SSID] = candidate
		}
	}
	result := make([]Network, 0, len(networks))
	for _, network := range networks {
		result = append(result, network)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Active != result[j].Active {
			return result[i].Active
		}
		if result[i].Signal != result[j].Signal {
			return result[i].Signal > result[j].Signal
		}
		return strings.ToLower(result[i].SSID) < strings.ToLower(result[j].SSID)
	})
	return result, nil
}

func StatusNow() Status {
	output, err := nmcli("-t", "-f", "IN-USE,SSID", "device", "wifi", "list")
	if err != nil {
		return Status{}
	}
	for _, line := range strings.Split(output, "\n") {
		fields := splitTerse(line)
		if len(fields) >= 2 && strings.TrimSpace(fields[0]) == "*" && fields[1] != "" {
			return Status{Connected: true, SSID: fields[1]}
		}
	}
	return Status{}
}

func Connect(paths settings.Paths, ssid, password string) (Status, error) {
	if ssid == "" {
		return Status{}, errors.New("no network selected")
	}
	args := []string{"device", "wifi", "connect", ssid}
	if password != "" {
		args = append(args, "password", password)
	}
	if _, err := nmcli(args...); err != nil {
		return Status{}, err
	}

	values, err := paths.Load()
	if err != nil {
		values = settings.Defaults()
	}
	if network, ok := values["network"].(map[string]any); ok {
		network["ssid"] = ssid
	} else {
		values["network"] = map[string]any{"ssid": ssid, "psk": ""}
	}
	if err := paths.Save(values); err != nil {
		return Status{}, fmt.Errorf("connected, but could not save settings: %w", err)
	}
	return Status{Connected: true, SSID: ssid}, nil
}

func WaitForConnect(timeout time.Duration) Status {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if status := StatusNow(); status.Connected {
			return status
		}
		time.Sleep(500 * time.Millisecond)
	}
	return Status{}
}
