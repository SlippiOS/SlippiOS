package host

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const nixPortableURL = "https://github.com/DavHau/nix-portable/releases/download/v012/nix-portable-x86_64"

type Nix struct {
	WS     *Workspace
	Runner Runner
}

func NewNix(ws *Workspace) *Nix {
	return &Nix{WS: ws, Runner: NewRunner(ws.Root)}
}

func (n *Nix) hasSystemNix() bool {
	_, err := exec.LookPath("nix")
	return err == nil
}

func (n *Nix) portablePath() string {
	if value := os.Getenv("NIX_PORTABLE"); value != "" {
		return value
	}
	return n.WS.CachePath("nix-portable-bin")
}

func (n *Nix) ensurePortable() error {
	path := n.portablePath()
	if info, err := os.Stat(path); err == nil && info.Mode()&0o111 != 0 {
		return nil
	}
	if err := os.MkdirAll(n.WS.Cache, 0o755); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Nix is not installed; downloading nix-portable to %s\n", path)
	if err := downloadFile(nixPortableURL, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o755)
}

func (n *Nix) env() []string {
	env := append([]string{}, os.Environ()...)
	env = append(env,
		"NP_LOCATION="+n.WS.Cache,
		"XDG_CACHE_HOME="+n.WS.CachePath("xdg"),
		"NP_RUNTIME=bwrap",
	)
	return env
}

func (n *Nix) Command(args ...string) (*exec.Cmd, error) {
	if n.hasSystemNix() {
		return n.Runner.Command("nix", args...), nil
	}
	if err := n.ensurePortable(); err != nil {
		return nil, err
	}
	full := append([]string{"nix", "--option", "sandbox", "false"}, args...)
	cmd := n.Runner.Command(n.portablePath(), full...)
	cmd.Env = n.env()
	return cmd, nil
}

func (n *Nix) Run(args ...string) error {
	cmd, err := n.Command(args...)
	if err != nil {
		return err
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nix %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func (n *Nix) Output(args ...string) (string, error) {
	cmd, err := n.Command(args...)
	if err != nil {
		return "", err
	}
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("nix %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (n *Nix) Shell(packages []string, command ...string) error {
	args := []string{"shell", "--inputs-from", "."}
	args = append(args, packages...)
	args = append(args, "-c")
	args = append(args, command...)
	return n.Run(args...)
}

func (n *Nix) StoreDelete(path string) error {
	store := n.WS.CachePath("nix", "nix", "store") + string(filepath.Separator)
	if !strings.HasPrefix(path, store) {
		return nil
	}
	arg := path
	if !n.hasSystemNix() {
		// nix-portable exposes the project store at /nix/store inside its
		// sandbox. Passing the host path makes Nix treat it as an installable.
		arg = "/nix/store/" + filepath.Base(path)
	}
	return n.Run("store", "delete", arg)
}

func (n *Nix) Exec(args ...string) error {
	cmd, err := n.Command(args...)
	if err != nil {
		return err
	}
	path := cmd.Path
	if !filepath.IsAbs(path) {
		path, err = exec.LookPath(path)
		if err != nil {
			return err
		}
	}
	return Exec(n.WS.Root, path, cmd.Args[1:], cmd.Env)
}

func ReexecInShell(ws *Workspace, packages []string, args []string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	self, err = filepath.Abs(self)
	if err != nil {
		return err
	}
	command := append([]string{self}, args...)
	return NewNix(ws).Shell(packages, command...)
}

// EnsureTools re-executes the current command inside a Nix shell when any of
// the required tools are missing. The command must define a hidden --in-shell
// flag. done is true when a child process ran; its error is returned as-is.
func EnsureTools(ws *Workspace, tools, packages []string, inShell bool) (bool, error) {
	var missing []string
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	if len(missing) == 0 || inShell {
		return false, nil
	}
	args := append([]string{}, os.Args[1:]...)
	args = append(args, "--in-shell")
	fmt.Fprintf(os.Stderr, "Missing %s; running inside a Nix shell\n", strings.Join(missing, ", "))
	return true, ReexecInShell(ws, packages, args)
}

func downloadFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}
	tmp := path + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	progress := newProgress(resp.ContentLength, filepath.Base(path), os.Stderr)
	buffer := make([]byte, 1<<20)
	_, copyErr := io.CopyBuffer(io.MultiWriter(file, progress), resp.Body, buffer)
	progress.Done()
	if syncErr := file.Sync(); copyErr == nil {
		copyErr = syncErr
	}
	closeErr := file.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, path)
}
