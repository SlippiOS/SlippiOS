package dolphin

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"time"
)

// QuitMarker is sent by the in-game Quit code through the Slippi EXI device.
const QuitMarker = "SLIPPIOS_QUIT"

// WatchQuit watches Dolphin's captured output for the marker the in-game Quit
// code sends. The console listener writes to stderr without buffering, so the
// line is visible immediately.
func WatchQuit(path string, pid int, done <-chan struct{}) {
	var file *os.File
	for {
		f, err := os.Open(path)
		if err == nil {
			file = f
			break
		}
		select {
		case <-done:
			return
		case <-time.After(200 * time.Millisecond):
		}
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	for {
		select {
		case <-done:
			return
		default:
		}
		line, err := reader.ReadString('\n')
		if strings.Contains(line, QuitMarker) {
			StopProcessGroup(pid, done)
			return
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return
			}
			select {
			case <-done:
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
}

// QuitRequested reports whether the captured output contains the in-game quit
// marker. A quit is a clean exit even though the launcher stops Dolphin.
func QuitRequested(path string) bool {
	data, err := os.ReadFile(path)
	return err == nil && bytes.Contains(data, []byte(QuitMarker))
}

// StopProcessGroup sends SIGTERM to Dolphin's process group and escalates to
// SIGKILL if it does not exit.
func StopProcessGroup(pid int, done <-chan struct{}) {
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	select {
	case <-done:
		return
	case <-time.After(3 * time.Second):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}
}
