package slippi

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

type event struct {
	command byte
	payload []byte
}

func writeReplay(t *testing.T, events []event) string {
	t.Helper()
	sizes := map[byte]int{}
	for _, item := range events {
		sizes[item.command] = len(item.payload)
	}
	tableLength := 1 + len(sizes)*3
	var raw []byte
	raw = append(raw, 0x35, byte(tableLength))
	for command, size := range sizes {
		raw = append(raw, command, byte(size>>8), byte(size))
	}
	for _, item := range events {
		raw = append(raw, item.command)
		raw = append(raw, item.payload...)
	}

	data := append([]byte{}, rawMarker...)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(raw)))
	data = append(data, length...)
	data = append(data, raw...)

	path := filepath.Join(t.TempDir(), "test.slp")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func gameStartPayload(size int, stage int, players []Player) []byte {
	payload := make([]byte, size)
	binary.BigEndian.PutUint16(payload[0x12:0x14], uint16(stage))
	for index := 0; index < 4; index++ {
		base := 0x64 + index*0x24
		payload[base+1] = 3
	}
	for index, player := range players {
		base := 0x64 + index*0x24
		payload[base] = byte(player.CharacterID)
		payload[base+1] = byte(player.Type)
		payload[0x6D+index*0x24] = byte(player.TeamID)
		copy(payload[0x160+index*0x10:], player.Nametag)
		if size >= 0x1A4+index*0x1F+len(player.DisplayName) {
			copy(payload[0x1A4+index*0x1F:], player.DisplayName)
		}
		if size >= 0x220+index*0xA+len(player.ConnectCode) {
			copy(payload[0x220+index*0xA:], player.ConnectCode)
		}
	}
	return payload
}

func framePayload(size int, frame int) []byte {
	payload := make([]byte, size)
	binary.BigEndian.PutUint32(payload[0:4], uint32(int32(frame)))
	return payload
}

func TestSummaryUsesDisplayNames(t *testing.T) {
	path := writeReplay(t, []event{{
		command: gameStartCommand,
		payload: gameStartPayload(0x2F8, 31, []Player{
			{CharacterID: 20, DisplayName: "tag1", ConnectCode: "ABC#1"},
			{CharacterID: 15, DisplayName: "tag2", ConnectCode: "DEF#2"},
		}),
	}})
	summary, err := Summary(path)
	if err != nil {
		t.Fatal(err)
	}
	const want = "Falco (tag1) vs Jigglypuff (tag2) on Battlefield"
	if summary != want {
		t.Fatalf("summary = %q, want %q", summary, want)
	}
}

func TestSummaryFallsBackToNametag(t *testing.T) {
	path := writeReplay(t, []event{{
		command: gameStartCommand,
		// Old replays only carry the 420-byte game-start payload.
		payload: gameStartPayload(0x1A4, 31, []Player{
			{CharacterID: 20, Nametag: "one"},
			{CharacterID: 15, Nametag: "two"},
		}),
	}})
	summary, err := Summary(path)
	if err != nil {
		t.Fatal(err)
	}
	const want = "Falco (one) vs Jigglypuff (two) on Battlefield"
	if summary != want {
		t.Fatalf("summary = %q, want %q", summary, want)
	}
}

func TestSummaryGroupsTeams(t *testing.T) {
	payload := gameStartPayload(0x2F8, 31, []Player{
		{CharacterID: 20, DisplayName: "a", TeamID: 0},
		{CharacterID: 15, DisplayName: "b", TeamID: 1},
		{CharacterID: 2, DisplayName: "c", TeamID: 0},
		{CharacterID: 9, DisplayName: "d", TeamID: 1},
	})
	payload[0x0C] = 1
	path := writeReplay(t, []event{{command: gameStartCommand, payload: payload}})
	summary, err := Summary(path)
	if err != nil {
		t.Fatal(err)
	}
	const want = "Falco (a) & Fox (c) vs Jigglypuff (b) & Marth (d) on Battlefield"
	if summary != want {
		t.Fatalf("summary = %q, want %q", summary, want)
	}
}

func TestLastFrame(t *testing.T) {
	path := writeReplay(t, []event{
		{command: gameStartCommand, payload: gameStartPayload(0x1A4, 31, nil)},
		{command: preFrameCommand, payload: framePayload(63, -123)},
		{command: preFrameCommand, payload: framePayload(63, 0)},
		{command: postFrameCommand, payload: framePayload(72, 100)},
		{command: postFrameCommand, payload: framePayload(72, 101)},
		{command: gameEndCommand, payload: []byte{0, 1}},
	})
	frame, err := LastFrame(path)
	if err != nil {
		t.Fatal(err)
	}
	if frame != 101 {
		t.Fatalf("last frame = %d, want 101", frame)
	}
}

func TestLastFrameRejectsEmptyReplay(t *testing.T) {
	path := writeReplay(t, []event{{
		command: gameStartCommand,
		payload: gameStartPayload(0x1A4, 31, nil),
	}})
	if _, err := LastFrame(path); err == nil {
		t.Fatal("expected an error for a replay without frames")
	}
}
