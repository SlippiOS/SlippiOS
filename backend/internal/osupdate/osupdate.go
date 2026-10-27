// Package osupdate stages and installs SlippiOS system updates.
//
// The running system is an A/B image: systemd-sysupdate writes the staged
// store image and UKI to the inactive slot, and boot counting falls back to
// the previous system if the new one does not finish booting. Downloads use
// zsync against the active store partition when possible.
package osupdate

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultManifest = "https://github.com/SlippiOS/SlippiOS/releases/latest/download/manifest.json"
	downloadBase    = "https://github.com/SlippiOS/SlippiOS/releases/download"
	userAgent       = "SlippiOS/0.2"
)

var (
	sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	procMounts    = "/proc/mounts"
	bootIDFile    = "/proc/sys/kernel/random/boot_id"
)

// Paths describes the update state under /data.
type Paths struct {
	DataDir     string
	StageDir    string
	StatusFile  string
	Progress    string
	Latest      string
	Request     string
	Reboot      string
	Source      string
	VersionFile string
}

func DefaultPaths() Paths {
	root := os.Getenv("SLIPPIOS_DATA")
	if root == "" {
		root = "/data"
	}
	return ForData(root)
}

func ForData(root string) Paths {
	stage := filepath.Join(root, "system", "update")
	return Paths{
		DataDir:     root,
		StageDir:    stage,
		StatusFile:  filepath.Join(stage, "status.json"),
		Progress:    filepath.Join(stage, "progress.json"),
		Latest:      filepath.Join(stage, "latest.json"),
		Request:     filepath.Join(stage, "request"),
		Reboot:      filepath.Join(stage, "reboot"),
		Source:      filepath.Join(root, "system", "update-source.json"),
		VersionFile: "/etc/slippios/version",
	}
}

// Part is one downloaded payload in the release manifest.
type Part struct {
	Name   string
	SHA256 string
	Size   int64
	Zsync  string
}

// Manifest is the validated update manifest.
type Manifest struct {
	Version  string
	Tag      string
	NixStore Part
	UKI      Part
	Base     string
	URL      string
	Raw      map[string]any
}

// CheckResult is the outcome of comparing the installed version to the release.
type CheckResult struct {
	Current   string
	Latest    string
	Available bool
	Manifest  *Manifest
}

func readJSON(path string) map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{}
	}
	value := map[string]any{}
	if err := json.Unmarshal(data, &value); err != nil {
		return map[string]any{}
	}
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
	data = append(data, '\n')
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func request(url string, timeout time.Duration) (*http.Response, error) {
	httpRequest, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("User-Agent", userAgent)
	client := &http.Client{Timeout: timeout}
	response, err := client.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		response.Body.Close()
		return nil, fmt.Errorf("%s returned %s", url, response.Status)
	}
	return response, nil
}

func object(value any) (map[string]any, bool) {
	result, ok := value.(map[string]any)
	return result, ok
}

func stringValue(value any) (string, bool) {
	result, ok := value.(string)
	return result, ok
}

func intValue(value any) (int64, bool) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	result, err := strconv.ParseInt(number.String(), 10, 64)
	if err != nil {
		return 0, false
	}
	return result, true
}

// parseManifest validates the shape of a release manifest.
func parseManifest(data []byte) (*Manifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("invalid update manifest: %w", err)
	}
	raw, ok := object(value)
	if !ok {
		return nil, errors.New("invalid update manifest")
	}
	version, ok := stringValue(raw["version"])
	if !ok || version == "" {
		return nil, errors.New("update manifest has no version")
	}
	nixStoreRaw, nixStoreOK := object(raw["nix_store"])
	ukiRaw, ukiOK := object(raw["uki"])
	if !nixStoreOK || !ukiOK {
		return nil, errors.New("update manifest is missing nix_store/uki")
	}

	part := func(value map[string]any, label string) (Part, error) {
		name, nameOK := stringValue(value["name"])
		digest, digestOK := stringValue(value["sha256"])
		size, sizeOK := intValue(value["size"])
		if !nameOK || name == "" || strings.Contains(name, "/") {
			return Part{}, fmt.Errorf("update manifest has an invalid %s name", label)
		}
		if !digestOK || !sha256Pattern.MatchString(digest) {
			return Part{}, fmt.Errorf("update manifest has an invalid %s sha256", label)
		}
		if !sizeOK || size <= 0 {
			return Part{}, fmt.Errorf("update manifest has an invalid %s size", label)
		}
		zsync, _ := stringValue(value["zsync"])
		return Part{Name: name, SHA256: digest, Size: size, Zsync: zsync}, nil
	}
	nixStore, err := part(nixStoreRaw, "nix_store")
	if err != nil {
		return nil, err
	}
	uki, err := part(ukiRaw, "uki")
	if err != nil {
		return nil, err
	}
	tag := ""
	if rawTag, present := raw["tag"]; present && rawTag != nil {
		value, ok := stringValue(rawTag)
		if !ok {
			return nil, errors.New("update manifest has an invalid tag")
		}
		tag = value
	}
	return &Manifest{
		Version: version, Tag: tag, NixStore: nixStore, UKI: uki, Raw: raw,
	}, nil
}

func source(paths Paths) (string, string) {
	base := ""
	manifest := defaultManifest
	config := readJSON(paths.Source)
	if configured, ok := stringValue(config["base"]); ok && configured != "" {
		base = strings.TrimRight(configured, "/") + "/"
		manifest = base + "manifest.json"
		if custom, ok := stringValue(config["manifest"]); ok && custom != "" {
			manifest = custom
		}
	} else if custom, ok := stringValue(config["manifest"]); ok && custom != "" {
		manifest = custom
	}
	return base, manifest
}

func assetURL(base, tag, name string) (string, error) {
	if base != "" {
		return base + name, nil
	}
	if tag == "" {
		return "", errors.New("release manifest has no tag")
	}
	return downloadBase + "/" + tag + "/" + name, nil
}

func fetchManifest(paths Paths) (*Manifest, error) {
	base, url := source(paths)
	response, err := request(url, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch update manifest: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch update manifest: %w", err)
	}
	manifest, err := parseManifest(data)
	if err != nil {
		return nil, err
	}
	manifest.Base = base
	manifest.URL = url
	return manifest, nil
}

func currentVersion(paths Paths) string {
	data, err := os.ReadFile(paths.VersionFile)
	if err != nil {
		return "unknown"
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return "unknown"
	}
	return value
}

// currentBootID identifies the running boot, so an install can tell a
// completed reboot from a rollback to the previous system.
func currentBootID() string {
	data, err := os.ReadFile(bootIDFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func validVersionChar(c byte) bool {
	return c >= '0' && c <= '9' ||
		c >= 'a' && c <= 'z' ||
		c >= 'A' && c <= 'Z' ||
		c == '~' || c == '-' || c == '^' || c == '.'
}

func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func hasVersionPrefix(s string, c byte) int {
	if len(s) > 0 && s[0] == c {
		return 1
	}
	return 0
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isAlpha(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func firstByte(s string) int {
	if len(s) == 0 {
		return 0
	}
	return int(s[0])
}

// CompareVersions orders versions the way systemd's strverscmp_improved()
// does. systemd-boot and systemd-sysupdate use that ordering, so the updater
// must agree with them before offering a version. It treats '.', '-', '^'
// and '~' as segment prefixes, compares numeric segments numerically, and
// ranks a numeric segment above an alphabetic one.
func CompareVersions(a, b string) int {
	for {
		// Unsupported characters separate segments, so drop them before
		// comparing the next segment.
		for len(a) > 0 && !validVersionChar(a[0]) {
			a = a[1:]
		}
		for len(b) > 0 && !validVersionChar(b[0]) {
			b = b[1:]
		}

		// A '~' prefix makes a pre-release older.
		if hasVersionPrefix(a, '~') != 0 || hasVersionPrefix(b, '~') != 0 {
			r := compareInt(1-hasVersionPrefix(a, '~'), 1-hasVersionPrefix(b, '~'))
			if r != 0 {
				return r
			}
			a = a[1:]
			b = b[1:]
		}

		// A longer version with more segments is newer.
		if len(a) == 0 || len(b) == 0 {
			return compareInt(firstByte(a), firstByte(b))
		}

		// '-' marks a release, which is older than '^', '.', or an
		// unprefixed segment.
		if hasVersionPrefix(a, '-') != 0 || hasVersionPrefix(b, '-') != 0 {
			r := compareInt(1-hasVersionPrefix(a, '-'), 1-hasVersionPrefix(b, '-'))
			if r != 0 {
				return r
			}
			a = a[1:]
			b = b[1:]
		}

		// '^' marks a patched release, newer than '-' but older than '.'.
		if hasVersionPrefix(a, '^') != 0 || hasVersionPrefix(b, '^') != 0 {
			r := compareInt(1-hasVersionPrefix(a, '^'), 1-hasVersionPrefix(b, '^'))
			if r != 0 {
				return r
			}
			a = a[1:]
			b = b[1:]
		}

		// '.' marks a point release, older than an unprefixed segment.
		if hasVersionPrefix(a, '.') != 0 || hasVersionPrefix(b, '.') != 0 {
			r := compareInt(1-hasVersionPrefix(a, '.'), 1-hasVersionPrefix(b, '.'))
			if r != 0 {
				return r
			}
			a = a[1:]
			b = b[1:]
		}

		aEnd := 0
		for aEnd < len(a) && isDigit(a[aEnd]) {
			aEnd++
		}
		bEnd := 0
		for bEnd < len(b) && isDigit(b[bEnd]) {
			bEnd++
		}
		if aEnd > 0 || bEnd > 0 {
			// A numeric segment is newer than an alphabetic one.
			aNumeric, bNumeric := 0, 0
			if aEnd > 0 {
				aNumeric = 1
			}
			if bEnd > 0 {
				bNumeric = 1
			}
			if r := compareInt(aNumeric, bNumeric); r != 0 {
				return r
			}
			// Ignore leading zeros, then compare length and digits.
			aStart := 0
			for aStart < aEnd && a[aStart] == '0' {
				aStart++
			}
			bStart := 0
			for bStart < bEnd && b[bStart] == '0' {
				bStart++
			}
			if r := compareInt(aEnd-aStart, bEnd-bStart); r != 0 {
				return r
			}
			if r := strings.Compare(a[aStart:aEnd], b[bStart:bEnd]); r != 0 {
				return r
			}
		} else {
			for aEnd < len(a) && isAlpha(a[aEnd]) {
				aEnd++
			}
			for bEnd < len(b) && isAlpha(b[bEnd]) {
				bEnd++
			}
			common := aEnd
			if bEnd < common {
				common = bEnd
			}
			if r := strings.Compare(a[:common], b[:common]); r != 0 {
				return r
			}
			// A longer alphabetic segment is newer.
			if r := compareInt(aEnd, bEnd); r != 0 {
				return r
			}
		}
		a = a[aEnd:]
		b = b[bEnd:]
	}
}

// Check fetches the release manifest and records whether it is newer.
func (p Paths) Check() (*CheckResult, error) {
	current := currentVersion(p)
	manifest, err := fetchManifest(p)
	if err != nil {
		return nil, err
	}
	available := manifest.Version != current &&
		CompareVersions(manifest.Version, current) > 0
	if err := os.MkdirAll(p.StageDir, 0o755); err != nil {
		return nil, err
	}
	latest := map[string]any{
		"checked_at":       time.Now().Unix(),
		"current":          current,
		"latest":           manifest.Version,
		"update_available": available,
		"manifest":         manifest.Raw,
	}
	if manifest.Base != "" {
		latest["base"] = manifest.Base
	} else {
		latest["base"] = nil
	}
	if err := writeJSON(p.Latest, latest); err != nil {
		return nil, err
	}
	return &CheckResult{
		Current: current, Latest: manifest.Version, Available: available, Manifest: manifest,
	}, nil
}

func numberValue(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case json.Number:
		number, _ := strconv.ParseInt(typed.String(), 10, 64)
		return number
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		return 0
	}
}

func errorValue(progress, service map[string]any) any {
	if value, ok := progress["error"]; ok && value != nil && value != "" {
		return value
	}
	if value, ok := service["error"]; ok && value != nil && value != "" {
		return value
	}
	return nil
}

// Status reports the installed version and any staged update.
func (p Paths) Status() map[string]any {
	current := currentVersion(p)
	cached := readJSON(p.Latest)
	service := readJSON(p.StatusFile)
	progress := readJSON(p.Progress)
	latest := cached["latest"]
	available := false
	if latestVersion, ok := latest.(string); ok && latestVersion != "" {
		available = latestVersion != current && CompareVersions(latestVersion, current) > 0
	}

	phase := "idle"
	if value, ok := progress["phase"].(string); ok && value != "" {
		phase = value
	}
	if value, ok := service["phase"].(string); ok && value != "" {
		phase = value
	}
	latestVersion, _ := latest.(string)
	target, _ := service["version"].(string)
	if target == "" {
		// Status written by an older installer does not name its target.
		target = latestVersion
	}

	switch phase {
	case "installed":
		if target != "" && target == current {
			// The staged update is now running.
			service = map[string]any{}
			_ = os.Remove(p.StatusFile)
			_ = os.Remove(p.Progress)
			phase = "idle"
		} else if target != "" && target != current {
			recorded, _ := service["boot_id"].(string)
			if recorded != "" && recorded != currentBootID() {
				// The console rebooted but came back on another version:
				// boot assessment rolled the update back.
				service = map[string]any{
					"ok":    false,
					"phase": "failed",
					"error": "The updated system did not boot; the previous system was restored.",
				}
				phase = "failed"
				_ = writeJSON(p.StatusFile, service)
				_ = os.Remove(p.Progress)
			}
		}
	case "failed":
		// Once the running version is the one the last update targeted, the
		// failure is stale (for example after a manual install).
		if latestVersion != "" && latestVersion == current {
			service = map[string]any{}
			_ = os.Remove(p.StatusFile)
			_ = os.Remove(p.Progress)
			phase = "idle"
		}
	}

	var serviceOK any
	if value, ok := service["ok"]; ok {
		serviceOK = value
	}
	return map[string]any{
		"ok":               true,
		"current":          current,
		"latest":           latest,
		"update_available": available,
		"phase":            phase,
		"done":             numberValue(progress["done"]),
		"total":            numberValue(progress["total"]),
		"error":            errorValue(progress, service),
		"service_ok":       serviceOK,
	}
}

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

// CurrentStoreDevice returns the block device currently mounted as /nix/store.
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

// Stage downloads and verifies the newest update, then asks the root service
// to install it with systemd-sysupdate.
func (p Paths) Stage(log func(string)) (map[string]any, error) {
	if log == nil {
		log = func(string) {}
	}
	result, err := p.stage(log)
	if err != nil {
		p.fail(err.Error())
		return nil, err
	}
	return result, nil
}

func (p Paths) stage(log func(string)) (map[string]any, error) {
	if err := os.MkdirAll(p.StageDir, 0o755); err != nil {
		return nil, err
	}
	_ = os.Remove(p.Request)
	_ = os.Remove(p.StatusFile)
	_ = os.Remove(p.Progress)

	checked, err := p.Check()
	if err != nil {
		return nil, err
	}
	if !checked.Available {
		p.writeProgress("idle", 0, 0)
		return map[string]any{
			"ok": true, "updated": false,
			"current": checked.Current, "latest": checked.Latest,
		}, nil
	}

	manifest := checked.Manifest
	version := manifest.Version
	raw := manifest.NixStore
	uki := manifest.UKI
	rawURL, err := assetURL(manifest.Base, manifest.Tag, raw.Name)
	if err != nil {
		return nil, err
	}
	ukiURL, err := assetURL(manifest.Base, manifest.Tag, uki.Name)
	if err != nil {
		return nil, err
	}
	zsyncURL := ""
	if raw.Zsync != "" {
		zsyncURL, err = assetURL(manifest.Base, manifest.Tag, raw.Zsync)
		if err != nil {
			return nil, err
		}
	}

	removeUnmatched(filepath.Join(p.StageDir, "slippios_*.nix-store.raw"), raw.Name)
	removeUnmatched(filepath.Join(p.StageDir, "slippios_*.efi"), uki.Name)

	log("Staging SlippiOS " + version)
	p.writeProgress("downloading", 0, raw.Size)
	if _, err := p.stageFile(raw.Name, raw.SHA256, raw.Size, rawURL, zsyncURL, CurrentStoreDevice()); err != nil {
		return nil, err
	}
	p.writeProgress("downloading", 0, uki.Size)
	if _, err := p.stageFile(uki.Name, uki.SHA256, uki.Size, ukiURL, "", ""); err != nil {
		return nil, err
	}

	p.writeProgress("installing", 0, 0)
	if err := os.WriteFile(p.Request, []byte(version+"\n"), 0o644); err != nil {
		return nil, err
	}
	return map[string]any{
		"ok": true, "updated": true,
		"current": checked.Current, "latest": version, "phase": "installing",
	}, nil
}

// RequestReboot asks the root service to restart the console.
func (p Paths) RequestReboot() error {
	if err := os.MkdirAll(p.StageDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(p.Reboot, []byte(strconv.FormatInt(time.Now().Unix(), 10)+"\n"), 0o644)
}
