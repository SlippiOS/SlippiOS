package dolphin

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func digestOK(path, digest string) bool {
	if !strings.HasPrefix(digest, "sha256:") {
		return false
	}
	expected := strings.ToLower(strings.TrimPrefix(digest, "sha256:"))
	if len(expected) != 64 {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return false
	}
	return hex.EncodeToString(hasher.Sum(nil)) == expected
}

// Progress reports downloaded bytes and the expected total.
type Progress func(done, total int64)

func download(asset Asset, destination string, log func(string), progress Progress) error {
	log(fmt.Sprintf("Downloading %s (%d MB)…", asset.Name, asset.Size/(1024*1024)))
	if progress != nil {
		progress(0, asset.Size)
	}

	partial := destination + ".part"
	response, err := get(asset.BrowserDownloadURL, 60*time.Second)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	file, err := os.OpenFile(partial, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	done := int64(0)
	reported := int64(-1)
	buffer := make([]byte, 1<<20)
	for {
		count, readErr := response.Body.Read(buffer)
		if count > 0 {
			if _, err := file.Write(buffer[:count]); err != nil {
				file.Close()
				_ = os.Remove(partial)
				return err
			}
			done += int64(count)
			if progress != nil {
				progress(done, asset.Size)
			}
			current := done >> 20
			if current >= reported+16 {
				reported = current
				suffix := ""
				if asset.Size > 0 {
					suffix = fmt.Sprintf(" (%d%%)", done*100/asset.Size)
				}
				log(fmt.Sprintf("  %d MB%s", current, suffix))
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			file.Close()
			_ = os.Remove(partial)
			return readErr
		}
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(partial)
		return err
	}
	return os.Rename(partial, destination)
}

func switchCurrent(paths Paths, tag, destination string) error {
	temporary := filepath.Join(paths.Base, "current.tmp")
	_ = os.Remove(temporary)
	if err := os.Symlink(destination, temporary); err != nil {
		return err
	}
	if err := os.Rename(temporary, paths.Current); err != nil {
		return err
	}
	state := readJSON(paths.State)
	state["tag"] = tag
	state["checked_at"] = time.Now().Unix()
	return writeJSON(paths.State, state)
}

func (p Paths) Ensure(log func(string), force bool) (string, error) {
	return p.EnsureWithProgress(log, nil, force)
}

func (p Paths) EnsureWithProgress(log func(string), progress Progress, force bool) (string, error) {
	if log == nil {
		log = func(string) {}
	}
	// The release image seeds the beta AppImage. Use it without contacting
	// GitHub; the download path is for dev images and repairs.
	if info, err := os.Stat(BundledPath()); err == nil && !info.IsDir() {
		return BundledPath(), nil
	}
	if err := os.MkdirAll(p.Base, 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(p.Versions, 0o755); err != nil {
		return "", err
	}

	lock, err := os.OpenFile(p.Lock, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return "", err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return "", err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	existing, hasExisting := p.CurrentPath()
	state := readJSON(p.State)
	fresh := checkedAt(state) > time.Now().Add(-checkPeriod).Unix()
	if hasExisting && fresh && !force {
		return existing, nil
	}

	tag, asset, err := LatestRelease()
	if err != nil {
		if hasExisting {
			current, _ := p.CurrentTag()
			log(fmt.Sprintf("Update check failed, using %s: %v", current, err))
			return existing, nil
		}
		return "", fmt.Errorf("cannot check for Slippi updates: %w", err)
	}

	if hasExisting && !force {
		if current, ok := p.CurrentTag(); ok && current == tag {
			state["checked_at"] = time.Now().Unix()
			if err := writeJSON(p.State, state); err != nil {
				return "", err
			}
			return existing, nil
		}
	}

	directory := filepath.Join(p.Versions, tag)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	destination := filepath.Join(directory, assetName)
	if info, err := os.Stat(destination); err == nil && !info.IsDir() && digestOK(destination, asset.Digest) {
		log(fmt.Sprintf("Using cached %s.", tag))
	} else {
		if err := download(asset, destination, log, progress); err != nil {
			return "", fmt.Errorf("cannot download Slippi Dolphin: %w", err)
		}
		if !digestOK(destination, asset.Digest) {
			_ = os.Remove(destination)
			return "", errors.New("downloaded AppImage failed checksum verification")
		}
	}
	if progress != nil {
		progress(asset.Size, asset.Size)
	}
	if err := os.Chmod(destination, 0o755); err != nil {
		return "", err
	}
	if err := switchCurrent(p, tag, destination); err != nil {
		return "", err
	}
	return destination, nil
}
