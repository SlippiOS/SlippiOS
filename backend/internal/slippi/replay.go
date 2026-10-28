// Package slippi reads the small part of a Slippi replay needed to tell why a
// game ended. It intentionally does not parse the rest of the replay.
package slippi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

const (
	gameEndCommand = 0x39
	noContest      = 7
)

// The raw event array starts after this UBJSON lead-up. The object starts with
// {U\x03raw[$U#l; the array data follows the four-byte raw length.
var rawMarker = []byte{
	0x7B, 0x55, 0x03, 0x72, 0x61, 0x77,
	0x5B, 0x24, 0x55, 0x23, 0x6C,
}

// GameEndMethod returns the game-end method byte from a .slp file. The Slippi
// spec defines 1 = TIME!, 2 = GAME!, and 7 = No Contest. A file that is still
// being written, or that has not reached its game-end event, returns an error.
func GameEndMethod(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	start := bytes.Index(data, rawMarker)
	if start < 0 || start+len(rawMarker)+4 > len(data) {
		return 0, fmt.Errorf("no Slippi raw event stream")
	}
	start += len(rawMarker)
	rawLength := int(binary.BigEndian.Uint32(data[start : start+4]))
	start += 4
	end := len(data)
	if rawLength > 0 && start+rawLength < end {
		end = start + rawLength
	}
	if start >= end {
		return 0, fmt.Errorf("empty Slippi event stream")
	}

	// The first event lists the payload size for every command that follows.
	// Its length byte counts the length byte itself plus the 3-byte entries.
	if data[start] != 0x35 || start+2 > end {
		return 0, fmt.Errorf("missing Slippi event payload table")
	}
	tableStart := start + 1
	tableLength := int(data[tableStart])
	entriesStart := tableStart + 1
	entriesEnd := tableStart + tableLength
	if tableLength < 1 || entriesEnd > end {
		return 0, fmt.Errorf("truncated Slippi event payload table")
	}
	sizes := map[byte]int{}
	for offset := entriesStart; offset+3 <= entriesEnd; offset += 3 {
		sizes[data[offset]] = int(binary.BigEndian.Uint16(data[offset+1 : offset+3]))
	}

	position := entriesEnd
	for position < end {
		command := data[position]
		position++
		size, ok := sizes[command]
		if !ok {
			if command == gameEndCommand {
				size = 6
			} else {
				return 0, fmt.Errorf("unknown Slippi event 0x%X", command)
			}
		}
		if size < 0 || position+size > end {
			return 0, fmt.Errorf("truncated Slippi event 0x%X", command)
		}
		if command == gameEndCommand {
			if size < 1 {
				return 0, fmt.Errorf("empty Slippi game-end event")
			}
			return int(data[position]), nil
		}
		position += size
	}
	return 0, fmt.Errorf("Slippi game-end event not found")
}

// IsNoContest reports whether a game ended by quitting rather than by a normal
// game or time result.
func IsNoContest(path string) (bool, error) {
	method, err := GameEndMethod(path)
	if err != nil {
		return false, err
	}
	return method == noContest, nil
}
