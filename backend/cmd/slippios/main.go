// Command slippios is the Go control-plane spike.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SlippiOS/SlippiOS/backend/internal/auth"
	"github.com/SlippiOS/SlippiOS/backend/internal/desktop"
	"github.com/SlippiOS/SlippiOS/backend/internal/dolphin"
	"github.com/SlippiOS/SlippiOS/backend/internal/netpages"
	"github.com/SlippiOS/SlippiOS/backend/internal/osupdate"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/SlippiOS/SlippiOS/backend/internal/wifi"
)

type modes struct {
	json  bool
	human bool
}

func parseModes(args []string) ([]string, modes) {
	result := make([]string, 0, len(args))
	mode := modes{}
	for _, argument := range args {
		switch argument {
		case "--json":
			mode.json = true
		case "--human":
			mode.human = true
		default:
			result = append(result, argument)
		}
	}
	return result, mode
}

func isTTY() bool {
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func emit(payload map[string]any, mode modes, human func()) {
	jsonMode := mode.json || (!mode.human && !isTTY())
	if jsonMode {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "")
		_ = encoder.Encode(payload)
		if ok, _ := payload["ok"].(bool); !ok {
			os.Exit(1)
		}
		return
	}
	if ok, _ := payload["ok"].(bool); !ok {
		message, _ := payload["error"].(string)
		fmt.Fprintln(os.Stderr, "error:", message)
		os.Exit(1)
	}
	human()
}

func fail(message string) {
	emit(map[string]any{"ok": false, "error": message}, modes{}, func() {})
	os.Exit(1)
}

func statusPayload() map[string]any {
	paths := settings.DefaultPaths()
	summary := map[string]any{"exists": false, "path": paths.SettingsFile}
	values, loadErr := paths.Load()
	if loadErr == nil {
		summary["exists"] = true
		raw, _ := values["game"].(map[string]any)["iso"].(string)
		valid, reason := paths.ISOStatus(raw)
		summary["iso"] = raw
		summary["iso_path"] = paths.ISOPath(raw)
		summary["iso_valid"] = valid
		summary["iso_reason"] = reason
		if display, ok := values["display"].(map[string]any); ok {
			summary["refresh_hz"] = display["refresh_hz"]
			summary["resolution"] = display["resolution"]
		}
	} else if _, statErr := os.Stat(paths.SettingsFile); statErr == nil {
		summary["error"] = loadErr.Error()
	}

	login, loginDetail := paths.LoginStatus(paths.DolphinUserFile)
	adapter := adapterPresent()
	currentDolphin := dolphinCurrent()
	return map[string]any{
		"ok":       true,
		"settings": summary,
		"login":    map[string]any{"signed_in": login, "detail": loginDetail},
		"dolphin":  map[string]any{"current": currentDolphin},
		"adapter":  map[string]any{"present": adapter},
	}
}

func adapterPresent() bool {
	entries, err := filepath.Glob("/sys/bus/usb/devices/*")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		vendor, err := os.ReadFile(filepath.Join(entry, "idVendor"))
		if err != nil {
			continue
		}
		product, err := os.ReadFile(filepath.Join(entry, "idProduct"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(vendor)) == "057e" &&
			strings.TrimSpace(string(product)) == "0337" {
			return true
		}
	}
	return false
}

func dolphinCurrent() string {
	if tag, ok := dolphin.DefaultPaths().CurrentTag(); ok {
		return tag
	}
	return ""
}

func printStatus(payload map[string]any) {
	summary := payload["settings"].(map[string]any)
	state := "missing"
	if summary["exists"] == true {
		state = "ok"
	}
	fmt.Printf("Settings:  %s (%s)\n", summary["path"], state)
	if summary["iso"] != nil {
		gameState := "ok"
		if summary["iso_valid"] != true {
			gameState, _ = summary["iso_reason"].(string)
		}
		fmt.Printf("Game:      %s (%s)\n", summary["iso_path"], gameState)
		if summary["refresh_hz"] != nil {
			fmt.Printf("Display:   %v Hz\n", summary["refresh_hz"])
		}
	} else {
		fmt.Println("Game:      not configured")
	}
	login := payload["login"].(map[string]any)
	if login["signed_in"] == true {
		fmt.Printf("Login:     signed in (%s)\n", login["detail"])
	} else {
		fmt.Println("Login:     not signed in")
	}
	dolphin := payload["dolphin"].(map[string]any)
	current, _ := dolphin["current"].(string)
	if current == "" {
		current = "not installed"
	}
	fmt.Printf("Dolphin:   %s\n", current)
	adapter := payload["adapter"].(map[string]any)
	if adapter["present"] == true {
		fmt.Println("Adapter:   connected")
	} else {
		fmt.Println("Adapter:   not connected")
	}
}

func settingsCommand(args []string, mode modes) {
	paths := settings.DefaultPaths()
	if len(args) == 0 || args[0] == "show" {
		values, err := paths.Load()
		if err != nil {
			if _, statErr := os.Stat(paths.SettingsFile); statErr != nil {
				values = settings.Defaults()
				err = nil
			}
		}
		if err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		emit(map[string]any{"ok": true, "settings": values}, mode, func() {
			fmt.Printf("Game:      %v\n", values["game"].(map[string]any)["iso"])
			display := values["display"].(map[string]any)
			fmt.Printf("Display:   %v Hz, %v\n", display["refresh_hz"], display["resolution"])
			fmt.Printf("Audio:     %v\n", values["audio"].(map[string]any)["sink"])
			fmt.Printf("Network:   %v\n", values["network"].(map[string]any)["ssid"])
		})
		return
	}
	if args[0] != "set" || len(args) < 3 {
		fail("usage: slippios settings set section.key value")
	}
	key, value := args[1], args[2]
	section, name, ok := strings.Cut(key, ".")
	if !ok {
		fail("setting must be section.key")
	}
	values, err := paths.Load()
	if err != nil {
		if _, statErr := os.Stat(paths.SettingsFile); statErr == nil {
			fail(err.Error())
		}
		values = settings.Defaults()
	}
	valuesSection, ok := values[section].(map[string]any)
	if !ok {
		fail("unknown setting " + key)
	}
	if name == "iso" {
		path := paths.ISOPath(value)
		if relative, err := filepath.Rel(paths.ISODir, path); err == nil &&
			relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			value = relative
		} else {
			value = path
		}
	}
	if name == "refresh_hz" {
		if value == "auto" {
			valuesSection[name] = int64(0)
		} else {
			number, err := strconv.Atoi(value)
			if err != nil {
				fail("refresh_hz must be a number or auto")
			}
			valuesSection[name] = int64(number)
		}
	} else {
		valuesSection[name] = value
	}
	if err := paths.Save(values); err != nil {
		fail(err.Error())
	}
	emit(map[string]any{"ok": true, "settings": values}, mode, func() {
		fmt.Println("Saved", key)
	})
}

func isosCommand(mode modes) {
	paths := settings.DefaultPaths()
	entries := paths.FindISOs()
	payload := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		payload = append(payload, map[string]any{
			"path": entry.Path, "name": entry.Name,
			"valid": entry.Valid, "reason": entry.Reason,
		})
	}
	emit(map[string]any{"ok": true, "isos": payload}, mode, func() {
		if len(entries) == 0 {
			fmt.Println("No game images found in /data/isos.")
			return
		}
		for _, entry := range entries {
			state := "ok"
			if !entry.Valid {
				state = entry.Reason
			}
			fmt.Printf("%s: %s\n", entry.Name, state)
		}
	})
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func readCredentials(path string, fromStdin bool) (credentials, error) {
	var data []byte
	var err error
	if path != "" {
		data, err = os.ReadFile(path)
		if err != nil {
			return credentials{}, fmt.Errorf("cannot read credentials: %w", err)
		}
	} else if fromStdin {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return credentials{}, fmt.Errorf("cannot read credentials: %w", err)
		}
	} else {
		return credentials{}, nil
	}
	var value credentials
	if err := json.Unmarshal(data, &value); err != nil {
		if path != "" {
			return credentials{}, fmt.Errorf("cannot read credentials: %w", err)
		}
		return credentials{}, fmt.Errorf("invalid JSON: %w", err)
	}
	return value, nil
}

func loginCommand(args []string, mode modes) {
	flags := flag.NewFlagSet("login", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	email := flags.String("email", "", "Slippi account email")
	password := flags.String("password", "", "Slippi account password")
	fromStdin := flags.Bool("stdin", false, "read {email,password} JSON from stdin")
	file := flags.String("file", "", "read {email,password} JSON from a file")
	if err := flags.Parse(args); err != nil {
		fail(err.Error())
	}

	credentials, err := readCredentials(*file, *fromStdin)
	if err != nil {
		fail(err.Error())
	}
	if credentials.Email == "" {
		credentials.Email = *email
	}
	if credentials.Password == "" {
		credentials.Password = *password
	}
	if credentials.Email == "" || credentials.Password == "" {
		fail("email and password are required")
	}

	user, err := auth.Login(credentials.Email, credentials.Password)
	if err != nil {
		fail(err.Error())
	}

	paths := settings.DefaultPaths()
	if err := os.MkdirAll(paths.ProfileDir, 0o755); err != nil {
		fail(err.Error())
	}
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		fail(err.Error())
	}
	target := paths.Resolve(settings.DefaultUserJSON)
	if err := os.WriteFile(target, data, 0o600); err != nil {
		fail(fmt.Sprintf("cannot write %s: %v", target, err))
	}
	_ = os.Chmod(target, 0o600)

	values, err := paths.Load()
	if err != nil {
		values = settings.Defaults()
	}
	installed, detail := paths.InstallLogin(values)
	payload := map[string]any{
		"ok":           installed,
		"connect_code": user.ConnectCode,
		"display_name": user.DisplayName,
		"detail":       detail,
	}
	if !installed {
		payload["error"] = detail
	}
	emit(payload, mode, func() {
		fmt.Printf("Signed in as %s (%s)\n", user.DisplayName, user.ConnectCode)
	})
}

func netCommand(args []string) {
	flags := flag.NewFlagSet("net", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	token := flags.String("token", "", "URL token")
	port := flags.Int("port", 8080, "listen port")
	mode := flags.String("mode", "setup", "setup, iso, or login")
	if err := flags.Parse(args); err != nil {
		fail(err.Error())
	}
	if *token == "" {
		fail("--token is required")
	}
	switch *mode {
	case "setup", "iso", "login":
	default:
		fail("unknown net mode: " + *mode)
	}
	server := netpages.New(*token, *port, settings.DefaultPaths())
	server.Mode = *mode
	url, err := server.Start()
	if err != nil {
		fail(err.Error())
	}
	line, _ := json.Marshal(map[string]any{"ok": true, "url": url})
	fmt.Println(string(line))
	if err := server.Serve(); err != nil {
		fail(err.Error())
	}
}

func updateCommand(args []string, mode modes) {
	flags := flag.NewFlagSet("update", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	check := flags.Bool("check", false, "only check for a new release")
	if err := flags.Parse(args); err != nil {
		fail(err.Error())
	}
	paths := dolphin.DefaultPaths()
	if *check {
		tag, _, err := dolphin.LatestRelease()
		if err != nil {
			fail(err.Error())
		}
		var current any
		if value, ok := paths.CurrentTag(); ok {
			current = value
		}
		payload := map[string]any{
			"ok": true, "check": true, "current": current, "latest": tag,
		}
		emit(payload, mode, func() {
			shown := "not installed"
			if current != nil {
				shown = fmt.Sprint(current)
			}
			fmt.Printf("Current:   %s\nLatest:    %s\n", shown, tag)
		})
		return
	}

	path, err := paths.Ensure(func(message string) {
		fmt.Fprintln(os.Stderr, message)
	}, false)
	if err != nil {
		fail(err.Error())
	}
	current, _ := paths.CurrentTag()
	emit(map[string]any{
		"ok": true, "check": false, "current": current, "path": path,
	}, mode, func() {
		fmt.Printf("Dolphin:   %s\nPath:      %s\n", current, path)
	})
}

func osCommand(args []string, mode modes) {
	if len(args) == 0 {
		fail("os action required")
	}
	paths := osupdate.DefaultPaths()
	switch args[0] {
	case "check":
		checked, err := paths.Check()
		if err != nil {
			fail(err.Error())
		}
		emit(map[string]any{
			"ok": true, "current": checked.Current, "latest": checked.Latest,
			"update_available": checked.Available,
		}, mode, func() {
			fmt.Printf("Current:   %s\nLatest:    %s\n", checked.Current, checked.Latest)
			if checked.Available {
				fmt.Println("Update:    available")
			} else {
				fmt.Println("Update:    up to date")
			}
		})
	case "status":
		status := paths.Status()
		emit(status, mode, func() {
			fmt.Printf("Current:   %v\nLatest:    %v\n", status["current"], status["latest"])
			fmt.Printf("Phase:     %v\n", status["phase"])
			if status["error"] != nil {
				fmt.Printf("Error:     %v\n", status["error"])
			}
		})
	case "update":
		result, err := paths.Stage(func(message string) {
			fmt.Fprintln(os.Stderr, message)
		})
		if err != nil {
			fail(err.Error())
		}
		emit(result, mode, func() {
			if result["updated"] == true {
				fmt.Printf("Staged SlippiOS %v. Restart to install.\n", result["latest"])
			} else {
				fmt.Printf("SlippiOS %v is up to date.\n", result["current"])
			}
		})
	case "reboot":
		if err := paths.RequestReboot(); err != nil {
			fail(err.Error())
		}
		emit(map[string]any{"ok": true}, mode, func() {
			fmt.Println("Restarting…")
		})
	default:
		fail("unknown os action")
	}
}

func playCommand(args []string, mode modes) {
	flags := flag.NewFlagSet("play", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	requireLogin := flags.Bool("require-login", false, "refuse to start without a saved login")
	if err := flags.Parse(args); err != nil {
		fail(err.Error())
	}

	paths := settings.DefaultPaths()
	if installed, _ := paths.InstallLogin(settings.Defaults()); !installed && *requireLogin {
		fail("not signed in")
	}
	appimage, err := dolphin.DefaultPaths().Ensure(func(message string) {
		fmt.Fprintln(os.Stderr, message)
	}, false)
	if err != nil {
		fail(err.Error())
	}

	values, err := paths.Load()
	if err != nil {
		fail("no game image")
	}
	game, _ := values["game"].(map[string]any)
	raw, _ := game["iso"].(string)
	if raw == "" {
		fail("no game image")
	}
	valid, reason := paths.ISOStatus(raw)
	if !valid {
		fail(reason)
	}
	iso := paths.ISOPath(raw)

	returncode, err := runDolphin(paths, appimage, []string{
		"-C", "Dolphin.Display.Fullscreen=True",
		"-C", "Graphics.Hardware.VSync=False",
		"-b", "-e", iso,
	})
	if err != nil {
		fail(err.Error())
	}
	emit(map[string]any{
		"ok": returncode == 0, "iso": iso, "returncode": returncode,
	}, mode, func() {
		if returncode == 0 {
			fmt.Println("Dolphin exited cleanly.")
		} else {
			fmt.Printf("Dolphin exited with status %d.\n", returncode)
		}
	})
}

// runDolphin starts the Slippi AppImage with the shared user directory and
// returns its exit status.
func runDolphin(paths settings.Paths, appimage string, arguments []string) (int, error) {
	environment := os.Environ()
	runtime := fmt.Sprintf("/run/user/%d", os.Getuid())
	if info, err := os.Stat(runtime); err == nil && info.IsDir() {
		hasRuntime := false
		for _, value := range environment {
			if strings.HasPrefix(value, "XDG_RUNTIME_DIR=") {
				hasRuntime = true
				break
			}
		}
		if !hasRuntime {
			environment = append(environment, "XDG_RUNTIME_DIR="+runtime)
		}
	}

	args := append([]string{appimage, "-u", paths.DolphinDir}, arguments...)
	command := exec.Command("slippi-appimage-run", args...)
	command.Env = environment
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return exitError.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}

// dolphinConfigCommand opens Dolphin's own settings window so users can bind
// controllers or change graphics options.
func dolphinConfigCommand(mode modes) {
	paths := settings.DefaultPaths()
	appimage, err := dolphin.DefaultPaths().Ensure(func(message string) {
		fmt.Fprintln(os.Stderr, message)
	}, false)
	if err != nil {
		fail(err.Error())
	}
	returncode, err := runDolphin(paths, appimage, []string{
		"-C", "Graphics.Hardware.VSync=False",
	})
	if err != nil {
		fail(err.Error())
	}
	emit(map[string]any{"ok": returncode == 0, "returncode": returncode}, mode, func() {
		fmt.Println("Dolphin closed.")
	})
}

func displayCommand(args []string, mode modes) {
	if len(args) == 0 || args[0] != "apply" {
		fail("usage: slippios display apply")
	}
	values, err := settings.DefaultPaths().Load()
	if err != nil {
		emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
		return
	}
	if err := desktop.ApplyDisplay(values); err != nil {
		emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
		return
	}
	// Audio is best effort: a missing sink must not fail the display change.
	_ = desktop.ApplyAudio(values)
	emit(map[string]any{"ok": true}, mode, func() {
		fmt.Println("Display settings applied.")
	})
}

func audioCommand(args []string, mode modes) {
	if len(args) == 0 || args[0] != "sinks" {
		fail("usage: slippios audio sinks")
	}
	sinks, err := desktop.Sinks()
	if err != nil {
		emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
		return
	}
	list := make([]map[string]any, 0, len(sinks))
	for _, sink := range sinks {
		list = append(list, map[string]any{
			"name":        sink.Name,
			"description": sink.Description,
		})
	}
	emit(map[string]any{"ok": true, "sinks": list}, mode, func() {
		for _, sink := range sinks {
			fmt.Println(sink.Name)
		}
	})
}

func poweroffCommand(mode modes) {
	if err := exec.Command("systemctl", "poweroff", "--no-block").Run(); err != nil {
		fail(err.Error())
	}
	emit(map[string]any{"ok": true}, mode, func() {
		fmt.Println("Powering off.")
	})
}

func wifiCommand(args []string, mode modes) {
	if len(args) == 0 {
		fail("usage: slippios wifi scan|status|connect")
	}
	switch args[0] {
	case "scan":
		networks, err := wifi.Scan()
		if err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		payload := make([]map[string]any, 0, len(networks))
		for _, network := range networks {
			payload = append(payload, map[string]any{
				"ssid": network.SSID, "signal": network.Signal,
				"security": network.Security, "active": network.Active,
			})
		}
		emit(map[string]any{"ok": true, "networks": payload}, mode, func() {
			for _, network := range networks {
				active := " "
				if network.Active {
					active = "*"
				}
				security := network.Security
				if security == "" {
					security = "open"
				}
				fmt.Printf("%s %s  %d%%  %s\n", active, network.SSID, network.Signal, security)
			}
		})
	case "status":
		status := wifi.StatusNow()
		emit(map[string]any{"ok": true, "connected": status.Connected, "ssid": status.SSID}, mode, func() {
			if status.Connected {
				fmt.Printf("Wi-Fi: connected to %s\n", status.SSID)
			} else {
				fmt.Println("Wi-Fi: not connected")
			}
		})
	case "connect":
		flags := flag.NewFlagSet("wifi connect", flag.ExitOnError)
		ssid := flags.String("ssid", "", "network name")
		password := flags.String("password", "", "network password")
		_ = flags.Parse(args[1:])
		status, err := wifi.Connect(settings.DefaultPaths(), *ssid, *password)
		if err != nil {
			emit(map[string]any{"ok": false, "error": err.Error()}, mode, func() {})
			return
		}
		emit(map[string]any{"ok": true, "ssid": status.SSID}, mode, func() {
			fmt.Printf("Wi-Fi: connected to %s\n", status.SSID)
		})
	default:
		fail("unknown wifi action")
	}
}

func main() {
	args, mode := parseModes(os.Args[1:])
	if len(args) == 0 {
		fail("command required")
	}
	switch args[0] {
	case "status":
		payload := statusPayload()
		emit(payload, mode, func() { printStatus(payload) })
	case "settings":
		settingsCommand(args[1:], mode)
	case "isos":
		isosCommand(mode)
	case "login":
		loginCommand(args[1:], mode)
	case "net":
		netCommand(args[1:])
	case "update":
		updateCommand(args[1:], mode)
	case "os":
		osCommand(args[1:], mode)
	case "play":
		playCommand(args[1:], mode)
	case "dolphin-config":
		dolphinConfigCommand(mode)
	case "display":
		displayCommand(args[1:], mode)
	case "audio":
		audioCommand(args[1:], mode)
	case "poweroff":
		poweroffCommand(mode)
	case "wifi":
		wifiCommand(args[1:], mode)
	default:
		fail("unknown command: " + args[0])
	}
}
