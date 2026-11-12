package host

import (
	"encoding/json"
	"os"
)

type Asset struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Zsync  string `json:"zsync,omitempty"`
}

type Manifest struct {
	Version       string `json:"version"`
	Tag           string `json:"tag"`
	Commit        string `json:"commit"`
	Date          string `json:"date"`
	FullImage     *Asset `json:"full_image,omitempty"`
	UnsignedImage *Asset `json:"unsigned_image,omitempty"`
	NixStore      *Asset `json:"nix_store,omitempty"`
	UKI           *Asset `json:"uki,omitempty"`
}

func WriteManifest(path string, manifest Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func ReadManifest(path string) (Manifest, error) {
	var manifest Manifest
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest, err
	}
	err = json.Unmarshal(data, &manifest)
	return manifest, err
}

func decodeManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	err := json.Unmarshal(data, &manifest)
	return manifest, err
}
