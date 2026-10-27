package desktop

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

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
