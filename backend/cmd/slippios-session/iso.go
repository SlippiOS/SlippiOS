package main

import (
	"fmt"
	"strconv"

	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

func chooseISO(paths settings.Paths) (string, bool) {
	found := paths.FindISOs()
	valid := []string{}
	for _, entry := range found {
		if entry.Valid {
			valid = append(valid, entry.Path)
		}
	}
	if len(valid) > 0 {
		fmt.Print("\n                 Found game images:\n\n")
		for index, path := range valid {
			fmt.Printf("                 [%d] %s\n", index+1, path)
		}
		choice, ok := line("\n                 Select [1]: ")
		if ok {
			if choice == "" {
				return valid[0], true
			}
			if number, err := strconv.Atoi(choice); err == nil && number >= 1 && number <= len(valid) {
				return valid[number-1], true
			}
			fmt.Println("                 Unrecognized choice.")
		}
	} else {
		fmt.Println("\n                 No valid Melee image found.")
		fmt.Printf("                 Put an NTSC 1.02 ISO in %s/\n", paths.ISODir)
	}

	for {
		path, ok := line("\n                 Path to Melee ISO (blank to skip): ")
		if !ok || path == "" {
			return "", false
		}
		if valid, reason := paths.ISOStatus(path); valid {
			return path, true
		} else {
			fmt.Printf("                 %s\n", reason)
		}
	}
}
func anyValidISO(paths settings.Paths) bool {
	for _, entry := range paths.FindISOs() {
		if entry.Valid {
			return true
		}
	}
	return false
}
