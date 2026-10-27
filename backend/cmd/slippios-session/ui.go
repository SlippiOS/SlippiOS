package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

var input = bufio.NewReader(os.Stdin)

func clear() {
	fmt.Print("\033[2J\033[H")
}

func line(prompt string) (string, bool) {
	fmt.Print("                 " + prompt)
	value, err := input.ReadString('\n')
	if err != nil && value == "" {
		return "", false
	}
	return strings.TrimSpace(value), true
}

func bootAnimation() {
	frames := []string{"      ◇", "    ◇ ◇", "  ◇   ◇", "◇  SLIPPIOS  ◇"}
	for _, frame := range frames {
		clear()
		fmt.Print("\n\n\n                 " + frame + "\n")
		fmt.Print("\n                 Initializing console…\n")
		time.Sleep(550 * time.Millisecond)
	}
}

func alert(lines []string) {
	clear()
	fmt.Println()
	for _, text := range lines {
		fmt.Println("                 " + text)
	}
	fmt.Println()
	_, _ = line("Press Enter to continue…")
}
func readPassword() (string, bool) {
	// Avoid depending on a terminal package; stty is present in the image.
	previous := exec.Command("stty", "-echo")
	previous.Stdin = os.Stdin
	_ = previous.Run()
	value, ok := line("Password: ")
	restore := exec.Command("stty", "echo")
	restore.Stdin = os.Stdin
	_ = restore.Run()
	fmt.Println()
	return value, ok
}
