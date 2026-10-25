package dolphin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func withAPI(t *testing.T, url string) {
	t.Helper()
	saved := API
	API = url
	t.Cleanup(func() { API = saved })
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func TestLatestRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"draft": true, "tag_name": "draft", "assets": []any{
				map[string]any{"name": assetName},
			}},
			{"draft": false, "tag_name": "v1.2.3", "assets": []any{
				map[string]any{"name": "other"},
				map[string]any{"name": assetName, "size": 10,
					"browser_download_url": "http://example.test/app"},
			}},
		})
	}))
	defer server.Close()
	withAPI(t, server.URL)

	tag, asset, err := LatestRelease()
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v1.2.3" || asset.Name != assetName || asset.Size != 10 {
		t.Fatalf("tag = %q asset = %#v", tag, asset)
	}
}

func TestEnsureDownloadsAndCaches(t *testing.T) {
	content := []byte("fake appimage payload")
	requests := atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases":
			requests.Add(1)
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"tag_name": "v1",
				"assets": []any{map[string]any{
					"name":                 assetName,
					"size":                 len(content),
					"browser_download_url": "http://" + r.Host + "/asset",
					"digest":               digest(content),
				}},
			}})
		case "/asset":
			_, _ = w.Write(content)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	withAPI(t, server.URL+"/releases")

	paths := ForData(t.TempDir())
	var logs []string
	path, err := paths.Ensure(func(message string) { logs = append(logs, message) }, false)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(paths.Versions, "v1", assetName) {
		t.Fatalf("path = %s", path)
	}
	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(content) {
		t.Fatal("downloaded content differs")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}
	target, err := os.Readlink(paths.Current)
	if err != nil {
		t.Fatal(err)
	}
	if target != path {
		t.Fatalf("current = %s", target)
	}
	if tag, ok := paths.CurrentTag(); !ok || tag != "v1" {
		t.Fatalf("tag = %q %v", tag, ok)
	}
	if len(logs) == 0 || !strings.Contains(logs[0], "Downloading") {
		t.Fatalf("logs = %#v", logs)
	}

	again, err := paths.Ensure(nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if again != path {
		t.Fatalf("again = %s", again)
	}
	if requests.Load() != 1 {
		t.Fatalf("api requests = %d", requests.Load())
	}
}

func TestEnsureRejectsBadChecksum(t *testing.T) {
	content := []byte("fake appimage payload")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/releases" {
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"tag_name": "v1",
				"assets": []any{map[string]any{
					"name":                 assetName,
					"size":                 len(content),
					"browser_download_url": "http://" + r.Host + "/asset",
					"digest":               "sha256:" + strings.Repeat("0", 64),
				}},
			}})
			return
		}
		_, _ = w.Write(content)
	}))
	defer server.Close()
	withAPI(t, server.URL+"/releases")

	paths := ForData(t.TempDir())
	if _, err := paths.Ensure(nil, false); err == nil {
		t.Fatal("bad checksum accepted")
	}
	if _, err := os.Stat(filepath.Join(paths.Versions, "v1", assetName)); !os.IsNotExist(err) {
		t.Fatal("bad download still exists")
	}
	if _, err := os.Lstat(paths.Current); !os.IsNotExist(err) {
		t.Fatal("current symlink still exists")
	}
}

func TestEnsureFallsBackToCurrentOnCheckFailure(t *testing.T) {
	content := []byte("fake appimage payload")
	requests := atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/releases" {
			requests.Add(1)
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"tag_name": "v1",
				"assets": []any{map[string]any{
					"name":                 assetName,
					"size":                 len(content),
					"browser_download_url": "http://" + r.Host + "/asset",
					"digest":               digest(content),
				}},
			}})
			return
		}
		_, _ = w.Write(content)
	}))
	defer server.Close()
	withAPI(t, server.URL+"/releases")

	paths := ForData(t.TempDir())
	path, err := paths.Ensure(nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Expire the cache so the next call checks the API.
	state := readJSON(paths.State)
	state["checked_at"] = float64(1)
	if err := writeJSON(paths.State, state); err != nil {
		t.Fatal(err)
	}
	server.Close()

	var logs []string
	current, err := paths.Ensure(func(message string) { logs = append(logs, message) }, false)
	if err != nil {
		t.Fatal(err)
	}
	if current != path {
		t.Fatalf("current = %s, want %s", current, path)
	}
	if len(logs) == 0 || !strings.Contains(logs[0], "Update check failed") {
		t.Fatalf("logs = %#v", logs)
	}
	if requests.Load() != 1 {
		t.Fatalf("api requests = %d", requests.Load())
	}
}
