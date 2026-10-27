package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var meleeID = regexp.MustCompile(`^GAL[A-Z0-9]01`)

func (p Paths) ISOStatus(value string) (bool, string) {
	iso := p.ISOPath(value)
	file, err := os.Open(iso)
	if err != nil {
		return false, fmt.Sprintf("not found: %s", iso)
	}
	defer file.Close()
	header := make([]byte, 6)
	if _, err := file.Read(header); err != nil {
		return false, fmt.Sprintf("cannot read %s: %v", iso, err)
	}
	if !meleeID.Match(header) {
		return false, fmt.Sprintf("%s is not a Melee image (game id %q)", iso, header)
	}
	return true, "ok"
}

type ISO struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Valid  bool   `json:"valid"`
	Reason string `json:"reason"`
}

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
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			nested, err := os.ReadDir(filepath.Join(base, entry.Name()))
			if err != nil {
				continue
			}
			for _, child := range nested {
				if child.IsDir() {
					dirs = append(dirs, filepath.Join(base, entry.Name(), child.Name()))
				}
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
	return p.FindISOsIn(p.SearchDirs())
}

func (p Paths) FindISOsIn(dirs []string) []ISO {
	seen := map[string]bool{}
	result := []ISO{}
	for _, root := range dirs {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".iso") {
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
