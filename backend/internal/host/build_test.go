package host

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRewriteResultPrefersProjectStore(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, ".cache")
	physical := filepath.Join(cache, ".nix-portable", "nix", "store", "abc-pkg")
	if err := os.MkdirAll(physical, 0o755); err != nil {
		t.Fatal(err)
	}
	result := filepath.Join(root, "result-test")
	if err := os.Symlink("/nix/store/abc-pkg", result); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{Root: root, Cache: cache}
	target, err := rewriteResult(ws, result)
	if err != nil {
		t.Fatal(err)
	}
	if target != physical {
		t.Fatalf("target = %q, want %q", target, physical)
	}
	linked, err := os.Readlink(result)
	if err != nil {
		t.Fatal(err)
	}
	if linked != physical {
		t.Fatalf("link = %q, want %q", linked, physical)
	}
}
