// Package slippi reads the small parts of a Slippi replay needed by the
// shell: the game-start summary and the last frame.
package slippi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	gameStartCommand = 0x36
	preFrameCommand  = 0x37
	postFrameCommand = 0x38
	gameEndCommand   = 0x39
)

// The raw event array starts after this UBJSON lead-up.
var rawMarker = []byte{
	0x7B, 0x55, 0x03, 0x72, 0x61, 0x77,
	0x5B, 0x24, 0x55, 0x23, 0x6C,
}

type eventReader struct {
	data       []byte
	start, end int
	sizes      map[byte]int
}

func openEventReader(path string) (*eventReader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return newEventReader(data)
}

// openEventReaderPrefix reads only the start of the file. The game-start
// summary lives in the first event, so listing a folder of replays does not
// need to read every multi-megabyte file in full.
func openEventReaderPrefix(path string, limit int64) (*eventReader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if limit <= 0 || limit > size {
		limit = size
	}
	data := make([]byte, limit)
	read, err := io.ReadFull(file, data)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	return newEventReader(data[:read])
}

func newEventReader(data []byte) (*eventReader, error) {
	index := bytes.Index(data, rawMarker)
	if index < 0 || index+len(rawMarker)+4 > len(data) {
		return nil, fmt.Errorf("no Slippi raw event stream")
	}
	start := index + len(rawMarker)
	rawLength := int(binary.BigEndian.Uint32(data[start : start+4]))
	start += 4
	end := len(data)
	if rawLength > 0 && start+rawLength < end {
		end = start + rawLength
	}
	if start >= end || data[start] != 0x35 || start+2 > end {
		return nil, fmt.Errorf("missing Slippi event payload table")
	}
	tableStart := start + 1
	tableLength := int(data[tableStart])
	entriesStart := tableStart + 1
	entriesEnd := tableStart + tableLength
	if tableLength < 1 || entriesEnd > end {
		return nil, fmt.Errorf("truncated Slippi event payload table")
	}
	sizes := map[byte]int{}
	for offset := entriesStart; offset+3 <= entriesEnd; offset += 3 {
		sizes[data[offset]] = int(binary.BigEndian.Uint16(data[offset+1 : offset+3]))
	}
	if _, ok := sizes[gameEndCommand]; !ok {
		sizes[gameEndCommand] = 2
	}
	return &eventReader{data: data, start: entriesEnd, end: end, sizes: sizes}, nil
}

func (r *eventReader) each(callback func(byte, []byte) bool) error {
	position := r.start
	for position < r.end {
		command := r.data[position]
		position++
		size, ok := r.sizes[command]
		if !ok {
			if command == 0x7B {
				return nil // metadata starts here
			}
			return fmt.Errorf("unknown Slippi event 0x%X", command)
		}
		if position+size > r.end {
			return fmt.Errorf("truncated Slippi event 0x%X", command)
		}
		if callback(command, r.data[position:position+size]) {
			return nil
		}
		position += size
	}
	return nil
}

type Player struct {
	Port        int
	CharacterID int
	Type        int
	TeamID      int
	DisplayName string
	ConnectCode string
	Nametag     string
}

type GameStart struct {
	StageID int
	Teams   bool
	Players []Player
}

func ReadGameStart(path string) (*GameStart, error) {
	reader, err := openEventReaderPrefix(path, 64<<10)
	if err != nil {
		return nil, err
	}
	var start *GameStart
	err = reader.each(func(command byte, payload []byte) bool {
		if command != gameStartCommand {
			return false
		}
		start = parseGameStart(payload)
		return true
	})
	if err != nil {
		return nil, err
	}
	if start == nil {
		return nil, fmt.Errorf("Slippi game start not found")
	}
	return start, nil
}

func parseGameStart(payload []byte) *GameStart {
	// The payload excludes the command byte, while the published field
	// offsets include it. Subtract one from every offset.
	start := &GameStart{}
	if len(payload) >= 0x14 {
		start.StageID = int(binary.BigEndian.Uint16(payload[0x12:0x14]))
		start.Teams = payload[0x0C] != 0
	}
	for index := 0; index < 4; index++ {
		base := 0x64 + index*0x24
		if base+2 > len(payload) {
			break
		}
		player := Player{
			Port:        index + 1,
			CharacterID: int(payload[base]),
			Type:        int(payload[base+1]),
			TeamID:      int(payload[0x6D+index*0x24]),
		}
		if player.Type == 3 {
			continue
		}
		player.Nametag = decodeName(payload, 0x160+index*0x10, 0x10)
		player.DisplayName = decodeName(payload, 0x1A4+index*0x1F, 0x1F)
		player.ConnectCode = decodeName(payload, 0x220+index*0xA, 0xA)
		start.Players = append(start.Players, player)
	}
	return start
}

func decodeName(payload []byte, offset, length int) string {
	if offset >= len(payload) {
		return ""
	}
	end := offset + length
	if end > len(payload) {
		end = len(payload)
	}
	value := payload[offset:end]
	if zero := bytes.IndexByte(value, 0); zero >= 0 {
		value = value[:zero]
	}
	var text []rune
	for _, character := range value {
		if character >= 0x20 && character < 0x7F {
			text = append(text, rune(character))
		}
	}
	return strings.TrimSpace(string(text))
}

func (g *GameStart) Summary() string {
	players := make([]string, 0, len(g.Players))
	teams := map[int][]string{}
	teamOrder := make([]int, 0, len(g.Players))
	for _, player := range g.Players {
		if player.CharacterID < 0 {
			continue
		}
		character := characterNames[player.CharacterID]
		if character == "" {
			character = fmt.Sprintf("Character %d", player.CharacterID)
		}
		tag := player.DisplayName
		if tag == "" {
			tag = player.ConnectCode
		}
		if tag == "" {
			tag = player.Nametag
		}
		if tag != "" {
			character = fmt.Sprintf("%s (%s)", character, tag)
		}
		players = append(players, character)
		if _, ok := teams[player.TeamID]; !ok {
			teamOrder = append(teamOrder, player.TeamID)
		}
		teams[player.TeamID] = append(teams[player.TeamID], character)
	}
	if len(players) == 0 {
		return "Unknown replay"
	}
	stage := stageNames[g.StageID]
	if stage == "" {
		stage = "Unknown Stage"
	}
	if g.Teams {
		groups := make([]string, 0, len(teamOrder))
		for _, team := range teamOrder {
			groups = append(groups, strings.Join(teams[team], " & "))
		}
		return strings.Join(groups, " vs ") + " on " + stage
	}
	return strings.Join(players, " vs ") + " on " + stage
}

func Summary(path string) (string, error) {
	start, err := ReadGameStart(path)
	if err != nil {
		return "", err
	}
	return start.Summary(), nil
}

// LastFrame returns the last frame the playback build reports as
// [CURRENT_FRAME]. It is the largest frame number in the pre- and
// post-frame events, which starts at -123 for the countdown.
func LastFrame(path string) (int, error) {
	reader, err := openEventReader(path)
	if err != nil {
		return 0, err
	}
	last := 0
	err = reader.each(func(command byte, payload []byte) bool {
		if command != preFrameCommand && command != postFrameCommand {
			return false
		}
		if len(payload) < 4 {
			return false
		}
		frame := int(int32(binary.BigEndian.Uint32(payload[0:4])))
		if frame > last {
			last = frame
		}
		return false
	})
	if err != nil {
		return 0, err
	}
	if last == 0 {
		return 0, fmt.Errorf("Slippi frames not found")
	}
	return last, nil
}
