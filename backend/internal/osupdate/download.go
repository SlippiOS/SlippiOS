package osupdate

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func validFile(path, expected string, size int64) bool {
	info, err := os.Stat(path)
	if err != nil || info.Size() != size {
		return false
	}
	digest, err := fileSHA256(path)
	return err == nil && digest == expected
}

func (p Paths) writeProgress(phase string, done, total int64) {
	_ = writeJSON(p.Progress, map[string]any{
		"phase":      phase,
		"done":       done,
		"total":      total,
		"updated_at": time.Now().Unix(),
	})
}

func (p Paths) fail(message string) {
	_ = writeJSON(p.Progress, map[string]any{
		"phase":      "failed",
		"error":      message,
		"updated_at": time.Now().Unix(),
	})
}

func (p Paths) download(url, destination, expected string, size int64) error {
	partial := destination + ".part"
	_ = os.Remove(partial)
	p.writeProgress("downloading", 0, size)

	response, err := request(url, 60*time.Second)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer response.Body.Close()
	file, err := os.OpenFile(partial, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	done := int64(0)
	buffer := make([]byte, 1<<20)
	for {
		count, readErr := response.Body.Read(buffer)
		if count > 0 {
			if _, err := file.Write(buffer[:count]); err != nil {
				file.Close()
				_ = os.Remove(partial)
				return fmt.Errorf("download failed: %w", err)
			}
			done += int64(count)
			p.writeProgress("downloading", done, size)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			file.Close()
			_ = os.Remove(partial)
			return fmt.Errorf("download failed: %w", readErr)
		}
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(partial)
		return fmt.Errorf("download failed: %w", err)
	}
	if !validFile(partial, expected, size) {
		_ = os.Remove(partial)
		return errors.New("downloaded file failed checksum verification")
	}
	return os.Rename(partial, destination)
}

func CurrentStoreDevice() string {
	file, err := os.Open(procMounts)
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 || fields[1] != "/nix/store" {
			continue
		}
		device, err := filepath.EvalSymlinks(fields[0])
		if err != nil {
			return ""
		}
		info, err := os.Stat(device)
		if err == nil && info.Mode()&os.ModeDevice != 0 && info.Mode()&os.ModeCharDevice == 0 {
			return device
		}
		return ""
	}
	return ""
}

func (p Paths) zsync(seed, destination, rawURL, zsyncURL string, size int64) bool {
	executable, err := exec.LookPath("zsync")
	if err != nil || seed == "" {
		return false
	}
	_ = os.Remove(destination)
	logPath := filepath.Join(p.StageDir, "zsync.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		_ = os.WriteFile(filepath.Join(p.StageDir, "zsync-error"), []byte(err.Error()+"\n"), 0o644)
		return false
	}
	defer logFile.Close()

	command := exec.Command(executable, "-i", seed, "-o", destination, "-u", rawURL, zsyncURL)
	command.Stdout = logFile
	command.Stderr = logFile
	if err := command.Start(); err != nil {
		_ = os.WriteFile(filepath.Join(p.StageDir, "zsync-error"), []byte(err.Error()+"\n"), 0o644)
		return false
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			if err != nil {
				return false
			}
			info, err := os.Stat(destination)
			return err == nil && !info.IsDir()
		case <-ticker.C:
			var progress int64
			if info, err := os.Stat(destination); err == nil {
				progress = info.Size()
			}
			p.writeProgress("downloading", progress, size)
		}
	}
}

func (p Paths) stageFile(name, expected string, size int64, url, zsyncURL, seed string) (string, error) {
	destination := filepath.Join(p.StageDir, name)
	if validFile(destination, expected, size) {
		return destination, nil
	}
	_ = os.Remove(destination)
	if zsyncURL != "" && seed != "" {
		if p.zsync(seed, destination, url, zsyncURL, size) && validFile(destination, expected, size) {
			return destination, nil
		}
		_ = os.Remove(destination)
	}
	if err := p.download(url, destination, expected, size); err != nil {
		return "", err
	}
	return destination, nil
}

func removeUnmatched(pattern, keep string) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	for _, match := range matches {
		if filepath.Base(match) != keep {
			_ = os.Remove(match)
		}
	}
}
