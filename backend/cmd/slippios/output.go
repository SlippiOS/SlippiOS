package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

var errFailed = errors.New("command failed")

type modes struct {
	json  bool
	human bool
	out   io.Writer
	err   io.Writer
}

func (m modes) stdout() io.Writer {
	if m.out != nil {
		return m.out
	}
	return os.Stdout
}

func (m modes) stderr() io.Writer {
	if m.err != nil {
		return m.err
	}
	return os.Stderr
}

func (m modes) printf(format string, args ...any) {
	fmt.Fprintf(m.stdout(), format, args...)
}

func (m modes) println(args ...any) {
	fmt.Fprintln(m.stdout(), args...)
}

func (m modes) emit(payload map[string]any, human func()) error {
	jsonMode := m.json || (!m.human && !isTTY(m.stdout()))
	if jsonMode {
		encoder := json.NewEncoder(m.stdout())
		encoder.SetIndent("", "")
		_ = encoder.Encode(payload)
		if ok, _ := payload["ok"].(bool); !ok {
			return errFailed
		}
		return nil
	}
	if ok, _ := payload["ok"].(bool); !ok {
		message, _ := payload["error"].(string)
		fmt.Fprintln(m.stderr(), "error:", message)
		return errFailed
	}
	if human != nil {
		human()
	}
	return nil
}

func (m modes) fail(message string) error {
	return m.emit(map[string]any{"ok": false, "error": message}, nil)
}

func isTTY(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
