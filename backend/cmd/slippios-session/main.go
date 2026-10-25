// Command slippios-session is the terminal fallback when the graphical shell
// fails to start. It configures, updates, and launches Dolphin.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/SlippiOS/SlippiOS/backend/internal/auth"
	"github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

var input = bufio.NewReader(os.Stdin)

func clear() {
	fmt.Print("\033[2J\033[H")
}

func line(prompt string) (string, bool) {
	fmt.Print("                 " + prompt)
	value, err := input.ReadString('\n')
	if err != nil && value == "" {
		return "", false
	}
	return strings.TrimSpace(value), true
}

func bootAnimation() {
	frames := []string{"      ◇", "    ◇ ◇", "  ◇   ◇", "◇  SLIPPIOS  ◇"}
	for _, frame := range frames {
		clear()
		fmt.Print("\n\n\n                 " + frame + "\n")
		fmt.Print("\n                 Initializing console…\n")
		time.Sleep(550 * time.Millisecond)
	}
}

func alert(lines []string) {
	clear()
	fmt.Println()
	for _, text := range lines {
		fmt.Println("                 " + text)
	}
	fmt.Println()
	_, _ = line("Press Enter to continue…")
}

func chooseISO(paths settings.Paths) (string, bool) {
	found := paths.FindISOs()
	valid := []string{}
	for _, entry := range found {
		if entry.Valid {
			valid = append(valid, entry.Path)
		}
	}
	if len(valid) > 0 {
		fmt.Print("\n                 Found game images:\n\n")
		for index, path := range valid {
			fmt.Printf("                 [%d] %s\n", index+1, path)
		}
		choice, ok := line("\n                 Select [1]: ")
		if ok {
			if choice == "" {
				return valid[0], true
			}
			if number, err := strconv.Atoi(choice); err == nil && number >= 1 && number <= len(valid) {
				return valid[number-1], true
			}
			fmt.Println("                 Unrecognized choice.")
		}
	} else {
		fmt.Println("\n                 No valid Melee image found.")
		fmt.Printf("                 Put an NTSC 1.02 ISO in %s/\n", paths.ISODir)
	}

	for {
		path, ok := line("\n                 Path to Melee ISO (blank to skip): ")
		if !ok || path == "" {
			return "", false
		}
		if valid, reason := paths.ISOStatus(path); valid {
			return path, true
		} else {
			fmt.Printf("                 %s\n", reason)
		}
	}
}

func configureNetwork(values map[string]any) {
	network, _ := values["network"].(map[string]any)
	ssid, _ := network["ssid"].(string)
	ssid = strings.TrimSpace(ssid)
	psk, _ := network["psk"].(string)
	if ssid == "" {
		return
	}
	if _, err := exec.LookPath("nmcli"); err != nil {
		return
	}
	if output, err := exec.Command("nmcli", "-t", "-f", "STATE", "general").Output(); err == nil {
		if strings.HasPrefix(strings.TrimSpace(string(output)), "connected") {
			return
		}
	}

	fmt.Printf("\n                 Connecting to Wi-Fi “%s”…\n", ssid)
	arguments := []string{"--wait", "20", "device", "wifi", "connect", ssid}
	if psk != "" {
		arguments = append(arguments, "password", psk)
	}
	command := exec.Command("nmcli", arguments...)
	_ = command.Run()
}

func onboarding(paths settings.Paths, reason string) (map[string]any, bool) {
	clear()
	fmt.Print("\n                 SLIPPIOS SETUP\n\n")
	fmt.Println("                 Melee 1.02 (NTSC) is required. SlippiOS does not")
	fmt.Print("                 include or download game images.\n\n")
	if reason != "" {
		fmt.Print("                 " + reason + "\n\n")
	}

	iso, ok := chooseISO(paths)
	if !ok {
		return nil, false
	}
	ssid, _ := line("\n                 Wi-Fi SSID (blank to skip): ")
	psk := ""
	if ssid != "" {
		psk, _ = line("                 Wi-Fi password: ")
	}

	values := settings.Defaults()
	game := values["game"].(map[string]any)
	if relative, err := filepath.Rel(paths.ISODir, iso); err == nil &&
		relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		game["iso"] = relative
	} else {
		game["iso"] = iso
	}
	network := values["network"].(map[string]any)
	network["ssid"] = ssid
	network["psk"] = psk
	if err := paths.Save(values); err != nil {
		alert([]string{"Could not save settings.", err.Error()})
		return nil, false
	}
	fmt.Printf("\n                 Wrote %s\n", paths.SettingsFile)
	time.Sleep(time.Second)
	return values, true
}

func applyDisplay(values map[string]any) {
	if _, err := exec.LookPath("xrandr"); err != nil {
		return
	}
	query, err := exec.Command("xrandr", "--query").Output()
	if err != nil {
		return
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
		return
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
		return
	}
	_ = exec.Command("xrandr", arguments...).Run()
}

func applyAudio(values map[string]any) {
	audio, _ := values["audio"].(map[string]any)
	sink, _ := audio["sink"].(string)
	if sink == "" || sink == "auto" {
		return
	}
	if _, err := exec.LookPath("pactl"); err != nil {
		return
	}
	_ = exec.Command("pactl", "set-default-sink", sink).Run()
}

func readPassword() (string, bool) {
	// Avoid depending on a terminal package; stty is present in the image.
	previous := exec.Command("stty", "-echo")
	previous.Stdin = os.Stdin
	_ = previous.Run()
	value, ok := line("Password: ")
	restore := exec.Command("stty", "echo")
	restore.Stdin = os.Stdin
	_ = restore.Run()
	fmt.Println()
	return value, ok
}

func ensureLogin(paths settings.Paths, values map[string]any) (bool, string) {
	if ok, detail := paths.InstallLogin(values); ok {
		return true, detail
	}
	for attempt := 0; attempt < 3; attempt++ {
		clear()
		fmt.Print("\n                 SLIPPI ACCOUNT\n\n")
		email, ok := line("Email (blank to skip): ")
		if !ok || email == "" {
			return false, "skipped"
		}
		password, ok := readPassword()
		if !ok {
			return false, "skipped"
		}
		user, err := auth.Login(email, password)
		if err != nil {
			fmt.Printf("\n                 %s\n", err)
			time.Sleep(2 * time.Second)
			continue
		}

		target := filepath.Join(paths.ProfileDir, settings.DefaultUserJSON)
		if err := os.MkdirAll(paths.ProfileDir, 0o755); err != nil {
			alert([]string{"Could not save the login.", err.Error()})
			return false, "sign-in failed"
		}
		data, err := jsonMarshalIndent(user)
		if err != nil {
			alert([]string{"Could not save the login.", err.Error()})
			return false, "sign-in failed"
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			alert([]string{"Could not save the login.", err.Error()})
			return false, "sign-in failed"
		}
		_ = os.Chmod(target, 0o600)
		return paths.InstallLogin(values)
	}
	return false, "sign-in failed"
}

func jsonMarshalIndent(value any) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}

func ensureDolphin() (string, bool) {
	clear()
	fmt.Print("\n\n                 Checking for Slippi Dolphin updates…\n\n")
	appimage, err := dolphin.DefaultPaths().Ensure(func(message string) {
		fmt.Println("                 " + message)
	}, false)
	if err != nil {
		alert([]string{"Could not prepare Slippi Dolphin.", err.Error()})
		return "", false
	}
	return appimage, true
}

func environmentWithRuntime() []string {
	environment := os.Environ()
	runtime := fmt.Sprintf("/run/user/%d", os.Getuid())
	if info, err := os.Stat(runtime); err == nil && info.IsDir() {
		for _, value := range environment {
			if strings.HasPrefix(value, "XDG_RUNTIME_DIR=") {
				return environment
			}
		}
		environment = append(environment, "XDG_RUNTIME_DIR="+runtime)
	}
	return environment
}

func launch(paths settings.Paths, values map[string]any, appimage string) {
	game, _ := values["game"].(map[string]any)
	raw, _ := game["iso"].(string)
	iso := paths.ISOPath(raw)
	clear()
	fmt.Print("\n\n                 Launching Slippi Dolphin…\n\n")
	fmt.Println("                 " + iso)

	logPath := filepath.Join(paths.ProfileDir, "logs", "dolphin.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		alert([]string{"Could not create the Dolphin log.", err.Error()})
		return
	}
	logFile, err := os.Create(logPath)
	if err != nil {
		alert([]string{"Could not create the Dolphin log.", err.Error()})
		return
	}
	defer logFile.Close()

	command := exec.Command(
		"slippi-appimage-run", appimage,
		"-u", paths.DolphinDir,
		"-C", "Dolphin.Display.Fullscreen=True",
		"-C", "Graphics.Hardware.VSync=False",
		"-b", "-e", iso,
	)
	command.Env = environmentWithRuntime()
	command.Stdout = logFile
	command.Stderr = logFile
	runErr := command.Run()
	if errors.Is(runErr, exec.ErrNotFound) {
		alert([]string{"slippi-appimage-run is not installed in this image."})
		return
	}
	if runErr == nil {
		return
	}
	var exitError *exec.ExitError
	code := -1
	if errors.As(runErr, &exitError) {
		code = exitError.ExitCode()
	}
	tail := []string{}
	if data, err := os.ReadFile(logPath); err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) > 4 {
			lines = lines[len(lines)-4:]
		}
		tail = lines
	}
	alert(append([]string{
		fmt.Sprintf("Dolphin exited with status %d.", code),
		"Log: " + logPath,
	}, tail...))
}

func anyValidISO(paths settings.Paths) bool {
	for _, entry := range paths.FindISOs() {
		if entry.Valid {
			return true
		}
	}
	return false
}

func main() {
	bootAnimation()
	paths := settings.DefaultPaths()

	if paths.CheckHardwareBinding() {
		alert([]string{
			"This cartridge was used on different hardware.",
			"The saved Slippi login has been removed.",
			"Sign in again to play online.",
		})
	}

	_, statErr := os.Stat(paths.SettingsFile)
	haveSettings := statErr == nil
	var values map[string]any
	var loadErr error
	if haveSettings {
		values, loadErr = paths.Load()
	}

	switch {
	case !haveSettings && !anyValidISO(paths):
		values, _ = onboarding(paths, "First-time setup: no settings and no game image.")
	case !haveSettings:
		alert([]string{
			"Settings file is missing.",
			"Expected " + paths.SettingsFile,
			"An existing game image was found, but SlippiOS is not configured.",
		})
		values, _ = onboarding(paths, "Setup: settings file missing.")
	case loadErr != nil:
		alert([]string{"Settings file is invalid.", loadErr.Error()})
		values, _ = onboarding(paths, "Setup: settings file invalid.")
	default:
		game, _ := values["game"].(map[string]any)
		raw, _ := game["iso"].(string)
		if ok, reason := paths.ISOStatus(raw); !ok {
			alert([]string{"Game image problem.", reason})
			values, _ = onboarding(paths, "Setup: game image missing or invalid.")
		}
	}

	if values == nil {
		alert([]string{"Setup was not completed.", "Falling back to a shell."})
		return
	}

	configureNetwork(values)
	applyDisplay(values)
	applyAudio(values)

	if loggedIn, detail := ensureLogin(paths, values); loggedIn {
		fmt.Printf("\n                 Slippi login: %s\n", detail)
	} else {
		fmt.Print("\n                 No Slippi login; online play unavailable.\n")
	}

	appimage, ok := ensureDolphin()
	if !ok {
		return
	}
	launch(paths, values, appimage)
}
