package settings

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GameCube disc header: the game ID is at offset 0, but mods may change it.
// The magic word at 0x1C is what identifies a GameCube disc image.
var gameCubeMagic = []byte{0xC2, 0x33, 0x9F, 0x3D}

const gameCubeMagicOffset = 0x1C

func (p Paths) ISOStatus(value string) (bool, string) {
	if value == "" {
		return false, "not configured"
	}
	iso := p.ISOPath(value)
	file, err := os.Open(iso)
	if err != nil {
		return false, fmt.Sprintf("not found: %s", iso)
	}
	defer file.Close()
	header := make([]byte, gameCubeMagicOffset+len(gameCubeMagic))
	if _, err := file.ReadAt(header, 0); err != nil {
		return false, fmt.Sprintf("cannot read %s: %v", iso, err)
	}
	if !bytes.Equal(header[gameCubeMagicOffset:], gameCubeMagic) {
		return false, fmt.Sprintf("%s is not a GameCube image (magic %x)", iso,
			header[gameCubeMagicOffset:])
	}
	return true, "ok"
}

type ISO struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Valid  bool   `json:"valid"`
	Reason string `json:"reason"`
}

// maxISOSearchDepth bounds the walk for auto-discovered mounts. Some systems
// (notably WSL) mount enormous filesystems under /mnt, where an unbounded
// walk can take minutes.
const maxISOSearchDepth = 2

func (p Paths) SearchDirs() []string {
	dirs := []string{p.ISODir}
	for _, base := range []string{"/run/media", "/media", "/mnt"} {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				dirs = append(dirs, filepath.Join(base, entry.Name()))
			}
		}
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if !seen[dir] {
			seen[dir] = true
			result = append(result, dir)
		}
	}
	return result
}

func (p Paths) FindISOs() []ISO {
	return p.findISOs(p.SearchDirs(), maxISOSearchDepth)
}

func (p Paths) FindISOsIn(dirs []string) []ISO {
	return p.findISOs(dirs, -1)
}

func (p Paths) findISOs(dirs []string, maxDepth int) []ISO {
	seen := map[string]bool{}
	result := []ISO{}
	for _, root := range dirs {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() {
				if maxDepth >= 0 {
					relative, relErr := filepath.Rel(root, path)
					if relErr == nil && relative != "." &&
						strings.Count(filepath.ToSlash(relative), "/")+1 > maxDepth {
						return filepath.SkipDir
					}
				}
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(entry.Name()), ".iso") {
				return nil
			}
			if seen[path] {
				return nil
			}
			seen[path] = true
			valid, reason := p.ISOStatus(path)
			result = append(result, ISO{Path: path, Name: filepath.Base(path), Valid: valid, Reason: reason})
			return nil
		})
		if err != nil {
			continue
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}
