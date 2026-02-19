package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/WJQSERVER/readline/internal/input"
	"github.com/WJQSERVER/readline/internal/term"
)

func main() {
	t, err := term.NewTerminal(os.Stdin, os.Stdout)
	if err != nil {
		fmt.Printf("Error creating terminal: %v\n", err)
		return
	}

	restore, err := t.SetRaw()
	if err != nil {
		fmt.Printf("Error setting raw mode: %v\n", err)
		return
	}
	defer restore()

	fmt.Print("\r\nWJQ Readline Raw Key Debugger\r\n")
	printConsoleModes()
	fmt.Print("Commands: 'q' to quit, 'r' to toggle RAW BYTE mode\r\n")
	fmt.Print("-------------------------------------------\r\n")

	rawMode := false
	p := input.NewParser(t)

	for {
		if rawMode {
			var buf [16]byte
			n, err := os.Stdin.Read(buf[:])
			if err != nil {
				fmt.Printf("\r\nRead error: %v\r\n", err)
				break
			}
			fmt.Printf("\rRAW BYTES: ")
			for i := 0; i < n; i++ {
				fmt.Printf("0x%02x ", buf[i])
			}
			fmt.Print("\r\n")

			// Check for 'q' in raw mode
			for i := 0; i < n; i++ {
				if buf[i] == 'q' {
					fmt.Print("\r\nExiting...\r\n")
					return
				}
				if buf[i] == 'r' {
					rawMode = false
					fmt.Print("\r\nSwitched to PARSED mode\r\n")
					// Re-init parser because we consumed from stdin
					p = input.NewParser(t)
					break
				}
			}
			continue
		}

		ev, err := p.NextEvent()
		if err != nil {
			fmt.Printf("\r\nError: %v\r\n", err)
			break
		}

		if ev.Key == input.KeyRune && ev.Rune == 'r' {
			rawMode = true
			fmt.Print("\r\nSwitched to RAW mode (Direct Stdin Read)\r\n")
			continue
		}

		keyName := "Unknown"
		switch ev.Key {
		case input.KeyRune:
			keyName = fmt.Sprintf("Rune('%c')", ev.Rune)
		case input.KeyEnter:
			keyName = "Enter"
		case input.KeyBackspace:
			keyName = "Backspace"
		case input.KeyDelete:
			keyName = "Delete"
		case input.KeyLeft:
			keyName = "Left"
		case input.KeyRight:
			keyName = "Right"
		case input.KeyUp:
			keyName = "Up"
		case input.KeyDown:
			keyName = "Down"
		case input.KeyHome:
			keyName = "Home"
		case input.KeyEnd:
			keyName = "End"
		case input.KeyTab:
			keyName = "Tab"
		case input.KeyCtrlA:
			keyName = "Ctrl-A"
		case input.KeyCtrlB:
			keyName = "Ctrl-B"
		case input.KeyCtrlC:
			keyName = "Ctrl-C"
		case input.KeyCtrlD:
			keyName = "Ctrl-D"
		case input.KeyCtrlE:
			keyName = "Ctrl-E"
		case input.KeyCtrlF:
			keyName = "Ctrl-F"
		case input.KeyCtrlK:
			keyName = "Ctrl-K"
		case input.KeyCtrlL:
			keyName = "Ctrl-L"
		case input.KeyCtrlN:
			keyName = "Ctrl-N"
		case input.KeyCtrlP:
			keyName = "Ctrl-P"
		case input.KeyCtrlR:
			keyName = "Ctrl-R"
		case input.KeyCtrlU:
			keyName = "Ctrl-U"
		case input.KeyCtrlW:
			keyName = "Ctrl-W"
		case input.KeyEsc:
			keyName = "Esc"
		case input.KeyCtrlLeft:
			keyName = "Ctrl-Left / Alt-b"
		case input.KeyCtrlRight:
			keyName = "Ctrl-Right / Alt-f"
		case input.KeyCtrlDelete:
			keyName = "Ctrl-Delete"
		}

		fmt.Printf("\rKey Event: ID=%-2d, Name=%-20s Rune=%-5d (0x%04x)\r\n", ev.Key, keyName, ev.Rune, ev.Rune)

		if ev.Key == input.KeyCtrlC || (ev.Key == input.KeyRune && ev.Rune == 'q') {
			fmt.Print("\r\nExiting...\r\n")
			break
		}
	}
}

func printConsoleModes() {
	if runtime.GOOS == "windows" {
		fmt.Print("Platform: Windows\r\n")
	} else {
		fmt.Print("Platform: Unix-like\r\n")
	}
}
