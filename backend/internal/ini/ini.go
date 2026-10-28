// Package ini is a small line-preserving INI reader/writer for Dolphin
// configuration files. It keeps unknown sections and raw lines intact.
package ini

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type File struct {
	Preamble []string
	Sections []*Section
}

type Section struct {
	Name  string
	Lines []string
}

func Parse(data []byte) *File {
	file := &File{}
	var current *Section
	for _, raw := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line := strings.TrimRight(raw, "\r")
		if name, ok := sectionName(line); ok {
			if current != nil {
				current.Lines = trimTrailingBlank(current.Lines)
			}
			current = file.Ensure(name)
			continue
		}
		if current == nil {
			file.Preamble = append(file.Preamble, line)
			continue
		}
		current.Lines = append(current.Lines, line)
	}
	if current != nil {
		current.Lines = trimTrailingBlank(current.Lines)
	}
	file.Preamble = trimTrailingBlank(file.Preamble)
	return file
}

func trimTrailingBlank(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func sectionName(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[") {
		return "", false
	}
	end := strings.Index(trimmed, "]")
	if end < 0 {
		return "", false
	}
	return trimmed[1:end], true
}

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &File{}, nil
	}
	if err != nil {
		return nil, err
	}
	return Parse(data), nil
}

func (f *File) Get(name string) *Section {
	for _, section := range f.Sections {
		if section.Name == name {
			return section
		}
	}
	return nil
}

func (f *File) Ensure(name string) *Section {
	if section := f.Get(name); section != nil {
		return section
	}
	section := &Section{Name: name}
	f.Sections = append(f.Sections, section)
	return section
}

func (f *File) SetLines(name string, lines []string) {
	section := f.Ensure(name)
	section.Lines = append([]string(nil), lines...)
}

func (f *File) Delete(name string) {
	for i, section := range f.Sections {
		if section.Name == name {
			f.Sections = append(f.Sections[:i], f.Sections[i+1:]...)
			return
		}
	}
}

func (f *File) Lines(name string) []string {
	section := f.Get(name)
	if section == nil {
		return nil
	}
	return section.Lines
}

func (f *File) Bytes() []byte {
	var out strings.Builder
	for _, line := range f.Preamble {
		out.WriteString(line)
		out.WriteByte('\n')
	}
	for _, section := range f.Sections {
		out.WriteByte('[')
		out.WriteString(section.Name)
		out.WriteString("]\n")
		for _, line := range section.Lines {
			out.WriteString(line)
			out.WriteByte('\n')
		}
		out.WriteByte('\n')
	}
	return []byte(out.String())
}

func Save(path string, file *File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ini-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(file.Bytes()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("save %s: %w", path, err)
	}
	return nil
}
