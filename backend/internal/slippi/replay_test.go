package slippi

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func testReplay(method byte, complete bool) []byte {
	raw := []byte{0x35, 4, 0x39, 0, 6, 0x39, method, 0, 0, 0, 0, 0}
	data := append([]byte{}, rawMarker...)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(raw)))
	data = append(data, length...)
	data = append(data, raw...)
	if complete {
		data = append(data, '{', '}')
	}
	return data
}

func writeReplay(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "game.slp")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGameEndMethod(t *testing.T) {
	path := writeReplay(t, testReplay(7, true))
	method, err := GameEndMethod(path)
	if err != nil {
		t.Fatal(err)
	}
	if method != 7 {
		t.Fatalf("method = %d, want 7", method)
	}
	quit, err := IsNoContest(path)
	if err != nil || !quit {
		t.Fatalf("IsNoContest = %v, %v", quit, err)
	}
}

func TestGameEndMethodNormal(t *testing.T) {
	path := writeReplay(t, testReplay(2, true))
	quit, err := IsNoContest(path)
	if err != nil {
		t.Fatal(err)
	}
	if quit {
		t.Fatal("normal game reported as a quit")
	}
}

func TestGameEndMethodIncomplete(t *testing.T) {
	path := writeReplay(t, testReplay(7, false)[:20])
	if _, err := GameEndMethod(path); err == nil {
		t.Fatal("incomplete replay parsed without an error")
	}
}
