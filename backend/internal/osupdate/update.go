package osupdate

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

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

func (p Paths) RequestReboot() error {
	if err := os.MkdirAll(p.StageDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(p.Reboot, []byte(strconv.FormatInt(time.Now().Unix(), 10)+"\n"), 0o644)
}
