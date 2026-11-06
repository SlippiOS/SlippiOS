package host

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const devSettings = `schema_version = 1

[game]
iso = "test-melee.iso"

[display]
refresh_hz = 60
resolution = "auto"
rate = ""
custom = ""

[audio]
sink = "auto"

[network]
ssid = ""
psk = ""

[dolphin]
backend = "Vulkan"
resolution = 3
aspect = 3
vsync = false
msaa = 1
audio_backend = "Cubeb"
audio_latency = 20
controller = "adapter"
profile = ""

[expert]
enabled = false

[slippi]
user_json = "slippi-user.json"
`

func DevCLI(ws *Workspace, args []string) error {
	data, bin, err := setupDev(ws)
	if err != nil {
		return err
	}
	if err := buildCLI(ws, filepath.Join(bin, "slippios"), false); err != nil {
		return err
	}
	env := append(os.Environ(),
		"SLIPPIOS_DATA="+data,
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	return Exec(ws.Root, filepath.Join(bin, "slippios"), args, env)
}

func DevSSH(ws *Workspace) error {
	port := envDefault("SLIPPI_SSH_PORT", "2222")
	fmt.Println("password: slippios")
	args := []string{
		"-p", port,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"player@localhost",
	}
	return Exec(ws.Root, "ssh", args, os.Environ())
}

func DevPush(ws *Workspace) error {
	_, bin, err := setupDev(ws)
	if err != nil {
		return err
	}
	fmt.Println("Building CLI...")
	if err := buildCLI(ws, filepath.Join(bin, "slippios"), true); err != nil {
		return err
	}
	fmt.Println("Exporting shell...")
	nix := NewNix(ws)
	if err := nix.Run("build", ".#shell-export", "-o", "result-shell-export"); err != nil {
		return err
	}
	target, err := rewriteResult(ws, ws.Path("result-shell-export"))
	if err != nil {
		return err
	}
	shellSource := filepath.Join(target, "bin", "slippios-shell")
	shellTarget := filepath.Join(bin, "slippios-shell")
	if err := CopyFile(shellSource, shellTarget); err != nil {
		return err
	}
	_ = os.Chmod(filepath.Join(bin, "slippios"), 0o755)
	_ = os.Chmod(shellTarget, 0o755)

	port := envDefault("SLIPPI_SSH_PORT", "2222")
	askpass := filepath.Join(ws.CachePath("dev"), "askpass")
	if err := os.WriteFile(askpass, []byte("#!/bin/sh\necho slippios\n"), 0o700); err != nil {
		return err
	}
	env := append(os.Environ(),
		"SSH_ASKPASS="+askpass,
		"SSH_ASKPASS_REQUIRE=force",
		"DISPLAY=:0",
	)
	sshArgs := []string{
		"-p", port,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "PubkeyAuthentication=no",
		"-o", "PreferredAuthentications=password",
		"player@localhost",
	}
	remote := func(args ...string) error {
		return runDetached(env, "ssh", append(sshArgs, args...)...)
	}
	scp := func(args ...string) error {
		base := []string{
			"-P", port,
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "PubkeyAuthentication=no",
			"-o", "PreferredAuthentications=password",
		}
		return runDetached(env, "scp", append(base, args...)...)
	}

	ready := false
	for i := 0; i < 30; i++ {
		if err := remote("true"); err == nil {
			ready = true
			break
		}
		time.Sleep(time.Second)
	}
	if !ready {
		return fmt.Errorf("dev VM is not reachable on port %s (boot a dev image first)", port)
	}
	if err := remote("mkdir -p /data/dev"); err != nil {
		return err
	}
	if err := scp(filepath.Join(bin, "slippios"), "player@localhost:/data/dev/slippios.new"); err != nil {
		return err
	}
	if err := scp(shellTarget, "player@localhost:/data/dev/slippios-shell.new"); err != nil {
		return err
	}
	if err := remote("mv -f /data/dev/slippios.new /data/dev/slippios && mv -f /data/dev/slippios-shell.new /data/dev/slippios-shell && chmod 0755 /data/dev/slippios /data/dev/slippios-shell"); err != nil {
		return err
	}
	fmt.Println("Restarting the graphical session...")
	if err := remote("sudo -n /run/current-system/sw/bin/systemctl restart display-manager"); err != nil {
		return err
	}
	time.Sleep(3 * time.Second)
	fmt.Println("Pushed. Use 'mise run dev:ssh' for logs.")
	return nil
}

func setupDev(ws *Workspace) (string, string, error) {
	root := ws.CachePath("dev")
	data := filepath.Join(root, "data")
	bin := filepath.Join(root, "bin")
	for _, dir := range []string{
		filepath.Join(data, "isos"),
		filepath.Join(data, "system"),
		filepath.Join(data, "users", "default"),
		bin,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", "", err
		}
	}
	iso := filepath.Join(data, "isos", "test-melee.iso")
	if !fileExists(iso) {
		if err := makeTestISO(iso); err != nil {
			return "", "", err
		}
	}
	settings := filepath.Join(data, "users", "default", "settings.toml")
	if !fileExists(settings) {
		if err := os.WriteFile(settings, []byte(devSettings), 0o644); err != nil {
			return "", "", err
		}
	}
	return data, bin, nil
}

func buildCLI(ws *Workspace, output string, static bool) error {
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	runner := NewRunner(ws.Path("backend"))
	env := append([]string{}, os.Environ()...)
	if static {
		env = append(env, "CGO_ENABLED=0")
	}
	args := []string{"build"}
	if static {
		args = append(args, "-trimpath")
	}
	args = append(args, "-o", output, "./cmd/slippios")
	return runner.RunEnv(env, "go", args...)
}

func makeTestISO(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Truncate(64 << 20); err != nil {
		return err
	}
	_, err = file.WriteAt([]byte("GALE01"), 0)
	return err
}

func runDetached(env []string, name string, args ...string) error {
	cmd := exec.Command("setsid", append([]string{name}, args...)...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = nil
	return cmd.Run()
}
