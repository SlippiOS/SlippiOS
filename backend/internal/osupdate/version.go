package osupdate

import (
	"os"
	"strings"
)

func currentVersion(paths Paths) string {
	data, err := os.ReadFile(paths.VersionFile)
	if err != nil {
		return "unknown"
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return "unknown"
	}
	return value
}
func validVersionChar(c byte) bool {
	return c >= '0' && c <= '9' ||
		c >= 'a' && c <= 'z' ||
		c >= 'A' && c <= 'Z' ||
		c == '~' || c == '-' || c == '^' || c == '.'
}

func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func hasVersionPrefix(s string, c byte) int {
	if len(s) > 0 && s[0] == c {
		return 1
	}
	return 0
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isAlpha(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func firstByte(s string) int {
	if len(s) == 0 {
		return 0
	}
	return int(s[0])
}

// CompareVersions orders versions the way systemd's strverscmp_improved()
// does. systemd-boot and systemd-sysupdate use that ordering, so the updater
// must agree with them before offering a version. It treats '.', '-', '^'
// and '~' as segment prefixes, compares numeric segments numerically, and
// ranks a numeric segment above an alphabetic one.
func CompareVersions(a, b string) int {
	for {
		// Unsupported characters separate segments, so drop them before
		// comparing the next segment.
		for len(a) > 0 && !validVersionChar(a[0]) {
			a = a[1:]
		}
		for len(b) > 0 && !validVersionChar(b[0]) {
			b = b[1:]
		}

		// A '~' prefix makes a pre-release older.
		if hasVersionPrefix(a, '~') != 0 || hasVersionPrefix(b, '~') != 0 {
			r := compareInt(1-hasVersionPrefix(a, '~'), 1-hasVersionPrefix(b, '~'))
			if r != 0 {
				return r
			}
			a = a[1:]
			b = b[1:]
		}

		// A longer version with more segments is newer.
		if len(a) == 0 || len(b) == 0 {
			return compareInt(firstByte(a), firstByte(b))
		}

		// '-' marks a release, which is older than '^', '.', or an
		// unprefixed segment.
		if hasVersionPrefix(a, '-') != 0 || hasVersionPrefix(b, '-') != 0 {
			r := compareInt(1-hasVersionPrefix(a, '-'), 1-hasVersionPrefix(b, '-'))
			if r != 0 {
				return r
			}
			a = a[1:]
			b = b[1:]
		}

		// '^' marks a patched release, newer than '-' but older than '.'.
		if hasVersionPrefix(a, '^') != 0 || hasVersionPrefix(b, '^') != 0 {
			r := compareInt(1-hasVersionPrefix(a, '^'), 1-hasVersionPrefix(b, '^'))
			if r != 0 {
				return r
			}
			a = a[1:]
			b = b[1:]
		}

		// '.' marks a point release, older than an unprefixed segment.
		if hasVersionPrefix(a, '.') != 0 || hasVersionPrefix(b, '.') != 0 {
			r := compareInt(1-hasVersionPrefix(a, '.'), 1-hasVersionPrefix(b, '.'))
			if r != 0 {
				return r
			}
			a = a[1:]
			b = b[1:]
		}

		aEnd := 0
		for aEnd < len(a) && isDigit(a[aEnd]) {
			aEnd++
		}
		bEnd := 0
		for bEnd < len(b) && isDigit(b[bEnd]) {
			bEnd++
		}
		if aEnd > 0 || bEnd > 0 {
			// A numeric segment is newer than an alphabetic one.
			aNumeric, bNumeric := 0, 0
			if aEnd > 0 {
				aNumeric = 1
			}
			if bEnd > 0 {
				bNumeric = 1
			}
			if r := compareInt(aNumeric, bNumeric); r != 0 {
				return r
			}
			// Ignore leading zeros, then compare length and digits.
			aStart := 0
			for aStart < aEnd && a[aStart] == '0' {
				aStart++
			}
			bStart := 0
			for bStart < bEnd && b[bStart] == '0' {
				bStart++
			}
			if r := compareInt(aEnd-aStart, bEnd-bStart); r != 0 {
				return r
			}
			if r := strings.Compare(a[aStart:aEnd], b[bStart:bEnd]); r != 0 {
				return r
			}
		} else {
			for aEnd < len(a) && isAlpha(a[aEnd]) {
				aEnd++
			}
			for bEnd < len(b) && isAlpha(b[bEnd]) {
				bEnd++
			}
			common := aEnd
			if bEnd < common {
				common = bEnd
			}
			if r := strings.Compare(a[:common], b[:common]); r != 0 {
				return r
			}
			// A longer alphabetic segment is newer.
			if r := compareInt(aEnd, bEnd); r != 0 {
				return r
			}
		}
		a = a[aEnd:]
		b = b[bEnd:]
	}
}
