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

const defaultUpdateManifest = "https://pub-ac47e58c86e44a728518e30a6eb756da.r2.dev/manifest.json"

type FetchOptions struct {
	Offline  bool
	DryRun   bool
	NoRun    bool
	DiskSize string
}

func FetchImage(ws *Workspace, opts FetchOptions) error {
	manifestURL := os.Getenv("SLIPPIOS_UPDATE_MANIFEST")
	if manifestURL == "" {
		manifestURL = defaultUpdateManifest
	}
	resp, err := manifestClient.Get(manifestURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch manifest: %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	manifest, err := decodeManifest(data)
	if err != nil {
		return err
	}
	if manifest.Version == "" {
		return fmt.Errorf("release manifest is missing version")
	}
	if manifest.FullImage == nil || manifest.FullImage.Name == "" || manifest.FullImage.SHA256 == "" {
		return fmt.Errorf("release manifest is missing full_image name/sha256")
	}
	if err := os.MkdirAll(ws.Cache, 0o755); err != nil {
		return err
	}

	releaseURL := fmt.Sprintf("https://sourceforge.net/projects/slippi-os/files/v%s/", manifest.Version)
	if tag := manifest.Tag; tag != "" {
		releaseURL = "https://github.com/SlippiOS/SlippiOS/releases/tag/" + tag
	}
	fmt.Printf("Release: %s\n", manifest.Version)
	fmt.Printf("URL:     %s\n", releaseURL)
	fmt.Printf("Image:   %s\n", manifest.FullImage.Name)
	fmt.Printf("SHA256:  %s\n", manifest.FullImage.SHA256)

	compressed := ws.CachePath(manifest.FullImage.Name)
	raw := ws.CachePath(strings.TrimSuffix(filepath.Base(manifest.FullImage.Name), ".xz") + "-" + manifest.FullImage.SHA256[:12] + ".raw")

	if opts.DryRun {
		fmt.Println("dry run: not downloading")
		return nil
	}

	verified := false
	if info, err := os.Stat(compressed); err == nil && info.Size() == manifest.FullImage.Size {
		if sum, err := SHA256File(compressed); err == nil && sum == manifest.FullImage.SHA256 {
			verified = true
		}
	}
	if verified {
		fmt.Println("Using cached image.")
	} else if opts.Offline {
		return fmt.Errorf("offline: no verified cached image")
	} else {
		if _, err := os.Stat(compressed); err == nil {
			fmt.Fprintln(os.Stderr, "Cached file is incomplete or corrupt; starting over.")
			_ = os.Remove(compressed)
		}
		if err := CheckFreeSpace(ws.Cache, 20_000_000*1024); err != nil {
			return err
		}
		if _, err := exec.LookPath("curl"); err != nil {
			return fmt.Errorf("curl is required")
		}
		downloadURL := fmt.Sprintf("https://sourceforge.net/projects/slippi-os/files/v%s/%s/download",
			manifest.Version, manifest.FullImage.Name)
		tmp := compressed + ".tmp"
		_ = os.Remove(tmp)
		if err := NewRunner(ws.Root).Run("curl", "--fail", "--location", "--progress-bar", "--output", tmp, downloadURL); err != nil {
			_ = os.Remove(tmp)
			return err
		}
		sum, err := SHA256File(tmp)
		if err != nil {
			_ = os.Remove(tmp)
			return err
		}
		if sum != manifest.FullImage.SHA256 {
			_ = os.Remove(tmp)
			return fmt.Errorf("sha256 mismatch for %s", manifest.FullImage.Name)
		}
		if err := os.Rename(tmp, compressed); err != nil {
			return err
		}
	}

	if _, err := os.Stat(raw); err != nil {
		if _, err := exec.LookPath("xz"); err != nil {
			return fmt.Errorf("xz is required")
		}
		fmt.Println("Decompressing image...")
		if err := decompressXZ(compressed, raw); err != nil {
			return err
		}
	}
	if opts.NoRun {
		fmt.Printf("Image: %s\n", raw)
		return nil
	}
	fmt.Printf("Booting %s (%s)\n", raw, opts.DiskSize)
	return RunImage(ws, RunOptions{Image: raw, DiskSize: opts.DiskSize})
}
