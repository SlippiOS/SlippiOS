package host

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// ErrCancelled is returned when the user declines a destructive prompt.
var ErrCancelled = errors.New("cancelled")

type Runner struct {
	Dir    string
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func NewRunner(dir string) Runner {
	return Runner{
		Dir:    dir,
		Env:    os.Environ(),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func (r Runner) Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Dir = r.Dir
	if r.Env != nil {
		cmd.Env = r.Env
	}
	cmd.Stdin = r.Stdin
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	return cmd
}

func (r Runner) Run(name string, args ...string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s is required", name)
	}
	if err := r.Command(name, args...).Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (r Runner) Output(name string, args ...string) (string, error) {
	if _, err := exec.LookPath(name); err != nil {
		return "", fmt.Errorf("%s is required", name)
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = r.Dir
	if r.Env != nil {
		cmd.Env = r.Env
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return "", fmt.Errorf("%s: %w: %s", name, err, message)
		}
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (r Runner) RunQuiet(name string, args ...string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s is required", name)
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = r.Dir
	if r.Env != nil {
		cmd.Env = r.Env
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func (r Runner) RunEnv(env []string, name string, args ...string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s is required", name)
	}
	cmd := r.Command(name, args...)
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// Exec replaces the current process. It is used for commands that should hand
// control to another interactive program (the CLI sandbox, QEMU, Nix).
func Exec(dir, name string, args, env []string) error {
	if dir != "" {
		if err := os.Chdir(dir); err != nil {
			return err
		}
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("%s is required", name)
	}
	if env == nil {
		env = os.Environ()
	}
	return syscall.Exec(path, append([]string{path}, args...), env)
}

func ExecSelf(ws *Workspace, args ...string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	self, err = filepath.Abs(self)
	if err != nil {
		return err
	}
	return Exec(ws.Root, self, args, os.Environ())
}

func RunSelf(ws *Workspace, args ...string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	return NewRunner(ws.Root).Run(self, args...)
}
