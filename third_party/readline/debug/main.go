package main

import (
	"fmt"
	"os"

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

	p := input.NewParser(t)

	fmt.Print("\r\nWJQ Readline Raw Key Debugger\r\n")
	fmt.Print("Press any keys to see their parsed values.\r\n")
	fmt.Print("Type 'q' or Ctrl-C to exit.\r\n")
	fmt.Print("-------------------------------------------\r\n")

	for {
		ev, err := p.NextEvent()
		if err != nil {
			fmt.Printf("\r\nError: %v\r\n", err)
			break
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
		}

		fmt.Printf("\rKey Event: ID=%d, Name=%-20s Rune=%d\r\n", ev.Key, keyName, ev.Rune)

		if ev.Key == input.KeyCtrlC || (ev.Key == input.KeyRune && ev.Rune == 'q') {
			fmt.Print("\r\nExiting...\r\n")
			break
		}
	}
}
