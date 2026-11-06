package host

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var manifestClient = &http.Client{Timeout: 30 * time.Second}

func trimSpace(value string) string {
	return strings.TrimSpace(value)
}

func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func HumanSize(bytes int64) string {
	if bytes < 0 {
		return "?"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	value := float64(bytes) / float64(div)
	suffix := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}[exp]
	if value >= 10 {
		return fmt.Sprintf("%.0f %s", value, suffix)
	}
	return fmt.Sprintf("%.1f %s", value, suffix)
}

func SHA256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func LinkOrCopy(src, dst string) error {
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	return CopyFile(src, dst)
}

func DiskFree(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}

func CheckFreeSpace(path string, need uint64) error {
	free, err := DiskFree(path)
	if err != nil {
		return err
	}
	if free < need {
		return fmt.Errorf("not enough space in %s (need %s, have %s)",
			path, HumanSize(int64(need)), HumanSize(int64(free)))
	}
	return nil
}

func ReadLine(reader io.Reader) (string, error) {
	line, err := bufio.NewReader(reader).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func PromptYesNo(reader io.Reader, prompt string) (bool, error) {
	fmt.Fprint(os.Stderr, prompt)
	answer, err := ReadLine(reader)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(answer) {
	case "y", "yes":
		return true, nil
	case "n", "no", "":
		return false, nil
	default:
		return false, nil
	}
}

func FindOne(dir, pattern string, exclude ...string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return "", err
	}
	var found []string
	for _, match := range matches {
		skip := false
		for _, suffix := range exclude {
			if strings.HasSuffix(match, suffix) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		if info, err := os.Stat(match); err == nil && info.Mode().IsRegular() {
			found = append(found, match)
		}
	}
	if len(found) == 0 {
		return "", fmt.Errorf("cannot find %s in %s", pattern, dir)
	}
	if len(found) > 1 {
		sort.Strings(found)
		return "", fmt.Errorf("expected one %s in %s, found %d: %s",
			pattern, dir, len(found), strings.Join(found, ", "))
	}
	return found[0], nil
}

func FindImages(dir, pattern string, exclude ...string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return nil, err
	}
	var found []string
	for _, match := range matches {
		skip := false
		for _, suffix := range exclude {
			if strings.HasSuffix(match, suffix) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		if info, err := os.Stat(match); err == nil && info.Mode().IsRegular() {
			found = append(found, match)
		}
	}
	sort.Strings(found)
	return found, nil
}

func Resolve(path string) string {
	value, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return value
}

func RealPath(path string) string {
	value, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return Resolve(value)
}

func RemoveIfExists(path string) {
	_ = os.Remove(path)
}

func RemoveAllIfExists(path string) {
	_ = os.RemoveAll(path)
}

func DirSize(path string) int64 {
	var total int64
	seen := map[[2]uint64]bool{}
	_ = filepath.WalkDir(path, func(_ string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			key := [2]uint64{uint64(stat.Dev), stat.Ino}
			if seen[key] {
				return nil
			}
			seen[key] = true
			total += stat.Blocks * 512
			return nil
		}
		total += info.Size()
		return nil
	})
	return total
}

func PrintDirSize(path string) {
	fmt.Printf("%s\t%s\n", HumanSize(DirSize(path)), path)
}

func digHoles(path string) {
	_ = exec.Command("fallocate", "--dig-holes", path).Run()
}

func FileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func ImageID(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("%s:%d:%d", RealPath(path), info.Size(), info.ModTime().UnixNano())
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:12], nil
}

func IsTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func ParseInt(value string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(value), 10, 64)
}

func NowUTC() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

func Glob(dir, pattern string) []string {
	matches, _ := filepath.Glob(filepath.Join(dir, pattern))
	sort.Strings(matches)
	return matches
}
