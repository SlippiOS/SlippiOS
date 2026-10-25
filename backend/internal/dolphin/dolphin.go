// Package dolphin fetches and verifies the newest Slippi Mainline Dolphin
// release. A failed check or download leaves the current build usable.
package dolphin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const (
	assetName   = "Slippi_Netplay_Mainline-x86_64.AppImage"
	userAgent   = "SlippiOS/0.1"
	checkPeriod = 15 * time.Minute
)

var (
	// API lists recent upstream releases. Tests replace it with a local server.
	API = "https://api.github.com/repos/project-slippi/dolphin/releases?per_page=10"

	tagPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

// Paths describes the on-disk Dolphin installation.
type Paths struct {
	Base     string
	Versions string
	Current  string
	State    string
	Lock     string
}

func DefaultPaths() Paths {
	root := os.Getenv("SLIPPIOS_DATA")
	if root == "" {
		root = "/data"
	}
	return ForData(root)
}

func ForData(root string) Paths {
	base := filepath.Join(root, "system", "dolphin")
	return Paths{
		Base:     base,
		Versions: filepath.Join(base, "versions"),
		Current:  filepath.Join(base, "current"),
		State:    filepath.Join(base, "state.json"),
		Lock:     filepath.Join(base, "lock"),
	}
}

// Asset is one GitHub release asset.
type Asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

type release struct {
	Draft   bool    `json:"draft"`
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

func get(url string, timeout time.Duration) (*http.Response, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: timeout}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		response.Body.Close()
		return nil, fmt.Errorf("%s returned %s", url, response.Status)
	}
	return response, nil
}

// LatestRelease returns the newest non-draft release carrying the AppImage.
func LatestRelease() (string, Asset, error) {
	response, err := get(API, 20*time.Second)
	if err != nil {
		return "", Asset{}, err
	}
	defer response.Body.Close()
	var releases []release
	if err := json.NewDecoder(response.Body).Decode(&releases); err != nil {
		return "", Asset{}, fmt.Errorf("cannot decode releases: %w", err)
	}
	for _, entry := range releases {
		if entry.Draft {
			continue
		}
		for _, asset := range entry.Assets {
			if asset.Name != assetName {
				continue
			}
			if !tagPattern.MatchString(entry.TagName) {
				return "", Asset{}, fmt.Errorf("unexpected release tag %q", entry.TagName)
			}
			return entry.TagName, asset, nil
		}
	}
	return "", Asset{}, errors.New("no Slippi release with a Linux AppImage found")
}

// CurrentPath returns the AppImage the current symlink points at.
func (p Paths) CurrentPath() (string, bool) {
	target, err := os.Readlink(p.Current)
	if err != nil {
		return "", false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(p.Current), target)
	}
	info, err := os.Stat(target)
	if err != nil || info.IsDir() {
		return "", false
	}
	return target, true
}

// CurrentTag returns the version directory of the current AppImage.
func (p Paths) CurrentTag() (string, bool) {
	path, ok := p.CurrentPath()
	if !ok {
		return "", false
	}
	return filepath.Base(filepath.Dir(path)), true
}

func readJSON(path string) map[string]any {
	value := map[string]any{}
	data, err := os.ReadFile(path)
	if err != nil {
		return value
	}
	_ = json.Unmarshal(data, &value)
	return value
}

func writeJSON(path string, value map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func checkedAt(state map[string]any) int64 {
	value, ok := state["checked_at"].(float64)
	if !ok {
		return 0
	}
	return int64(value)
}

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

func download(asset Asset, destination string, log func(string)) error {
	log(fmt.Sprintf("Downloading %s (%d MB)…", asset.Name, asset.Size/(1024*1024)))

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

// Ensure returns a verified AppImage, downloading the newest release if needed.
func (p Paths) Ensure(log func(string), force bool) (string, error) {
	if log == nil {
		log = func(string) {}
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
		if err := download(asset, destination, log); err != nil {
			return "", fmt.Errorf("cannot download Slippi Dolphin: %w", err)
		}
		if !digestOK(destination, asset.Digest) {
			_ = os.Remove(destination)
			return "", errors.New("downloaded AppImage failed checksum verification")
		}
	}
	if err := os.Chmod(destination, 0o755); err != nil {
		return "", err
	}
	if err := switchCurrent(p, tag, destination); err != nil {
		return "", err
	}
	return destination, nil
}
