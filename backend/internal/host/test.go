package host

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func RunTests(ws *Workspace) error {
	fmt.Println("Running Go tests...")
	runner := NewRunner(ws.Path("backend"))
	if err := runner.Run("go", "test", "./..."); err != nil {
		return err
	}
	fmt.Println("Parsing the Godot project...")
	parseArgs := []string{"--headless", "--path", "apps/shell", "--import"}
	quitArgs := []string{"--headless", "--path", "apps/shell", "--quit"}
	if _, err := exec.LookPath("godot"); err == nil {
		if err := runGodot(ws, parseArgs); err != nil {
			return err
		}
		return runGodot(ws, quitArgs)
	}
	nix := NewNix(ws)
	packages := []string{"nixpkgs#godot_4"}
	if err := runGodotNix(nix, packages, parseArgs); err != nil {
		return err
	}
	return runGodotNix(nix, packages, quitArgs)
}

func runGodot(ws *Workspace, args []string) error {
	cmd := exec.Command("godot", args...)
	cmd.Dir = ws.Root
	cmd.Env = append(os.Environ(), "SLIPPIOS_PARSE_ONLY=1")
	return runGodotCommand(cmd)
}

func runGodotNix(nix *Nix, packages, args []string) error {
	shell := []string{"shell", "--inputs-from", "."}
	shell = append(shell, packages...)
	shell = append(shell, "-c", "godot")
	shell = append(shell, args...)
	cmd, err := nix.Command(shell...)
	if err != nil {
		return err
	}
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	cmd.Env = append(cmd.Env, "SLIPPIOS_PARSE_ONLY=1")
	return runGodotCommand(cmd)
}

// Godot exits 0 even when it cannot parse a script, so inspect the output.
func runGodotCommand(cmd *exec.Cmd) error {
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	out, err := cmd.CombinedOutput()
	os.Stdout.Write(out)
	if strings.Contains(string(out), "SCRIPT ERROR") ||
		strings.Contains(string(out), "Parse Error") {
		return fmt.Errorf("Godot reported a script parse error")
	}
	return err
}
