package osupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sha(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func manifestFor(version string, raw, uki []byte) map[string]any {
	return map[string]any{
		"version": version,
		"tag":     "v" + version,
		"nix_store": map[string]any{
			"name":   "slippios_" + version + ".nix-store.raw",
			"sha256": sha(raw),
			"size":   len(raw),
		},
		"uki": map[string]any{
			"name":   "slippios_" + version + ".efi",
			"sha256": sha(uki),
			"size":   len(uki),
		},
	}
}

func updateServer(t *testing.T, manifest map[string]any, files map[string][]byte) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			_ = json.NewEncoder(w).Encode(manifest)
			return
		}
		content, ok := files[filepath.Base(r.URL.Path)]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(content)
	}))
	t.Cleanup(server.Close)
	return server
}

func testPaths(t *testing.T, version string) Paths {
	t.Helper()
	paths := ForData(t.TempDir())
	paths.VersionFile = filepath.Join(paths.DataDir, "version")
	if err := os.WriteFile(paths.VersionFile, []byte(version+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return paths
}

func writeSource(t *testing.T, paths Paths, base string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(paths.Source), 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]any{"base": base})
	if err := os.WriteFile(paths.Source, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestManifestValidation(t *testing.T) {
	raw := []byte("raw")
	uki := []byte("uki")
	valid := manifestFor("200", raw, uki)
	if _, err := parseManifest(mustJSON(t, valid)); err != nil {
		t.Fatal(err)
	}

	cases := map[string]map[string]any{
		"missing version": {"nix_store": valid["nix_store"], "uki": valid["uki"]},
		"missing parts":   {"version": "200"},
		"empty name": {"version": "200", "nix_store": map[string]any{
			"name": "", "sha256": sha(raw), "size": len(raw),
		}, "uki": valid["uki"]},
		"path name": {"version": "200", "nix_store": map[string]any{
			"name": "dir/raw", "sha256": sha(raw), "size": len(raw),
		}, "uki": valid["uki"]},
		"uppercase digest": {"version": "200", "nix_store": map[string]any{
			"name": "raw", "sha256": strings.ToUpper(sha(raw)), "size": len(raw),
		}, "uki": valid["uki"]},
		"zero size": {"version": "200", "nix_store": map[string]any{
			"name": "raw", "sha256": sha(raw), "size": 0,
		}, "uki": valid["uki"]},
		"bad tag": {"version": "200", "tag": 5,
			"nix_store": valid["nix_store"], "uki": valid["uki"]},
	}
	for name, manifest := range cases {
		name, manifest := name, manifest
		t.Run(name, func(t *testing.T) {
			if _, err := parseManifest(mustJSON(t, manifest)); err == nil {
				t.Fatal("manifest accepted")
			}
		})
	}
	if _, err := parseManifest([]byte("[]")); err == nil {
		t.Fatal("array accepted")
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestVersionKey(t *testing.T) {
	cases := map[string]int64{
		"1792939455-b16a03c": 1792939455,
		"42":                 42,
		"unknown":            0,
		"":                   0,
	}
	for value, expected := range cases {
		if got := VersionKey(value); got != expected {
			t.Fatalf("VersionKey(%q) = %d, want %d", value, got, expected)
		}
	}
}

func TestCheckAndStage(t *testing.T) {
	rawContent := []byte("nix store image")
	ukiContent := []byte("unified kernel image")
	manifest := manifestFor("200", rawContent, ukiContent)
	files := map[string][]byte{
		"slippios_200.nix-store.raw": rawContent,
		"slippios_200.efi":           ukiContent,
	}
	server := updateServer(t, manifest, files)
	paths := testPaths(t, "100")
	writeSource(t, paths, server.URL+"/")

	checked, err := paths.Check()
	if err != nil {
		t.Fatal(err)
	}
	if !checked.Available || checked.Current != "100" || checked.Latest != "200" {
		t.Fatalf("check = %#v", checked)
	}
	cached := readJSON(paths.Latest)
	if cached["latest"] != "200" || cached["update_available"] != true || cached["base"] != server.URL+"/" {
		t.Fatalf("latest = %#v", cached)
	}

	var logs []string
	result, err := paths.Stage(func(message string) { logs = append(logs, message) })
	if err != nil {
		t.Fatal(err)
	}
	if result["updated"] != true || result["phase"] != "installing" || result["latest"] != "200" {
		t.Fatalf("stage = %#v", result)
	}
	if len(logs) != 1 || !strings.Contains(logs[0], "Staging SlippiOS 200") {
		t.Fatalf("logs = %#v", logs)
	}
	for name, content := range files {
		stored, err := os.ReadFile(filepath.Join(paths.StageDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(stored) != string(content) {
			t.Fatalf("%s content differs", name)
		}
	}
	request, err := os.ReadFile(paths.Request)
	if err != nil {
		t.Fatal(err)
	}
	if string(request) != "200\n" {
		t.Fatalf("request = %q", request)
	}
	progress := readJSON(paths.Progress)
	if progress["phase"] != "installing" {
		t.Fatalf("progress = %#v", progress)
	}
}

func TestStageNoUpdate(t *testing.T) {
	rawContent := []byte("raw")
	ukiContent := []byte("uki")
	manifest := manifestFor("100", rawContent, ukiContent)
	server := updateServer(t, manifest, nil)
	paths := testPaths(t, "100")
	writeSource(t, paths, server.URL+"/")

	result, err := paths.Stage(nil)
	if err != nil {
		t.Fatal(err)
	}
	if result["updated"] != false || result["current"] != "100" {
		t.Fatalf("stage = %#v", result)
	}
	if _, err := os.Stat(paths.Request); !os.IsNotExist(err) {
		t.Fatal("request written for up-to-date system")
	}
	if progress := readJSON(paths.Progress); progress["phase"] != "idle" {
		t.Fatalf("progress = %#v", progress)
	}
}

func TestStageChecksumFailure(t *testing.T) {
	rawContent := []byte("nix store image")
	ukiContent := []byte("unified kernel image")
	manifest := manifestFor("200", rawContent, ukiContent)
	raw := manifest["nix_store"].(map[string]any)
	raw["sha256"] = strings.Repeat("0", 64)
	files := map[string][]byte{
		"slippios_200.nix-store.raw": rawContent,
		"slippios_200.efi":           ukiContent,
	}
	server := updateServer(t, manifest, files)
	paths := testPaths(t, "100")
	writeSource(t, paths, server.URL+"/")

	if _, err := paths.Stage(nil); err == nil {
		t.Fatal("bad checksum accepted")
	}
	progress := readJSON(paths.Progress)
	if progress["phase"] != "failed" || progress["error"] == nil {
		t.Fatalf("progress = %#v", progress)
	}
	if _, err := os.Stat(paths.Request); !os.IsNotExist(err) {
		t.Fatal("request written after failure")
	}
	if _, err := os.Stat(filepath.Join(paths.StageDir, "slippios_200.nix-store.raw.part")); !os.IsNotExist(err) {
		t.Fatal("partial file still exists")
	}
}

func TestStatusClearsInstalledUpdate(t *testing.T) {
	paths := testPaths(t, "200")
	if err := os.MkdirAll(paths.StageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(paths.Latest, map[string]any{
		"latest": "200", "update_available": true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(paths.StatusFile, map[string]any{"ok": true, "phase": "installed"}); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(paths.Progress, map[string]any{"phase": "installing"}); err != nil {
		t.Fatal(err)
	}

	status := paths.Status()
	if status["phase"] != "idle" || status["service_ok"] != nil {
		t.Fatalf("status = %#v", status)
	}
	for _, path := range []string{paths.StatusFile, paths.Progress} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still exists", path)
		}
	}
}

func TestStatusClearsFailedUpdateAfterBoot(t *testing.T) {
	paths := testPaths(t, "200")
	if err := os.MkdirAll(paths.StageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(paths.Latest, map[string]any{
		"latest": "200", "update_available": true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(paths.StatusFile, map[string]any{
		"ok": false, "phase": "failed", "error": "systemd-sysupdate failed",
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(paths.Progress, map[string]any{"phase": "installing"}); err != nil {
		t.Fatal(err)
	}

	status := paths.Status()
	if status["phase"] != "idle" || status["error"] != nil {
		t.Fatalf("status = %#v", status)
	}
	if status["update_available"] != false {
		t.Fatalf("stale update available flag survived: %#v", status)
	}
	for _, path := range []string{paths.StatusFile, paths.Progress} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still exists", path)
		}
	}
}

func TestStatusKeepsInstalledBeforeReboot(t *testing.T) {
	paths := testPaths(t, "100")
	oldBootID := bootIDFile
	bootIDFile = filepath.Join(paths.DataDir, "boot_id")
	defer func() { bootIDFile = oldBootID }()
	if err := os.WriteFile(bootIDFile, []byte("boot-1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.StageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(paths.Latest, map[string]any{"latest": "200"}); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(paths.StatusFile, map[string]any{
		"ok": true, "phase": "installed", "version": "200", "boot_id": "boot-1",
	}); err != nil {
		t.Fatal(err)
	}

	status := paths.Status()
	if status["phase"] != "installed" {
		t.Fatalf("status = %#v", status)
	}
	if _, err := os.Stat(paths.StatusFile); err != nil {
		t.Fatalf("installed status removed before reboot: %v", err)
	}
}

func TestStatusDetectsRollbackAfterBoot(t *testing.T) {
	paths := testPaths(t, "100")
	oldBootID := bootIDFile
	bootIDFile = filepath.Join(paths.DataDir, "boot_id")
	defer func() { bootIDFile = oldBootID }()
	if err := os.WriteFile(bootIDFile, []byte("boot-2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.StageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(paths.Latest, map[string]any{"latest": "200", "update_available": true}); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(paths.StatusFile, map[string]any{
		"ok": true, "phase": "installed", "version": "200", "boot_id": "boot-1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(paths.Progress, map[string]any{"phase": "installing"}); err != nil {
		t.Fatal(err)
	}

	status := paths.Status()
	if status["phase"] != "failed" {
		t.Fatalf("status = %#v", status)
	}
	if status["error"] != "The updated system did not boot; the previous system was restored." {
		t.Fatalf("error = %v", status["error"])
	}
	if status["update_available"] != true {
		t.Fatalf("update should be retryable: %#v", status)
	}
	if _, err := os.Stat(paths.Progress); !os.IsNotExist(err) {
		t.Fatal("progress file still exists")
	}
	persisted := readJSON(paths.StatusFile)
	if persisted["phase"] != "failed" {
		t.Fatalf("rollback not persisted: %#v", persisted)
	}
}

func TestStatusReportsFailure(t *testing.T) {
	paths := testPaths(t, "100")
	if err := os.MkdirAll(paths.StageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(paths.Progress, map[string]any{
		"phase": "failed", "error": "checksum mismatch",
	}); err != nil {
		t.Fatal(err)
	}
	status := paths.Status()
	if status["phase"] != "failed" || status["error"] != "checksum mismatch" {
		t.Fatalf("status = %#v", status)
	}
}

func TestRequestReboot(t *testing.T) {
	paths := ForData(t.TempDir())
	if err := paths.RequestReboot(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(paths.Reboot)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) == "" {
		t.Fatal("reboot request is empty")
	}
}
