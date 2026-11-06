package host

import (
	"fmt"
	"os"
	"os/exec"
)

func RunTests(ws *Workspace) error {
	fmt.Println("Running Go tests...")
	runner := NewRunner(ws.Path("backend"))
	if err := runner.Run("go", "test", "./..."); err != nil {
		return err
	}
	fmt.Println("Parsing the Godot project...")
	godotArgs := []string{"--headless", "--path", "apps/shell", "--import"}
	if _, err := exec.LookPath("godot"); err == nil {
		if err := runGodot(ws, nil, godotArgs); err != nil {
			return err
		}
		return runGodot(ws, nil, []string{"--headless", "--path", "apps/shell", "--quit"})
	}
	nix := NewNix(ws)
	packages := []string{"nixpkgs#godot_4"}
	if err := nix.Shell(packages, "env", "SLIPPIOS_PARSE_ONLY=1", "godot", "--headless", "--path", "apps/shell", "--import"); err != nil {
		return err
	}
	return nix.Shell(packages, "env", "SLIPPIOS_PARSE_ONLY=1", "godot", "--headless", "--path", "apps/shell", "--quit")
}

func runGodot(ws *Workspace, extraEnv []string, args []string) error {
	env := append([]string{}, os.Environ()...)
	env = append(env, "SLIPPIOS_PARSE_ONLY=1")
	env = append(env, extraEnv...)
	return NewRunner(ws.Root).RunEnv(env, "godot", args...)
}
