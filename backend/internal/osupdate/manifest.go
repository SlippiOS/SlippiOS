package osupdate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

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
