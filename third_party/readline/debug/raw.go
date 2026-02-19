package main

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

func main() {
	// 1. 将当前终端切换到 Raw Mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	fmt.Print("\r\nWJQ Readline RAW HEX Debugger\r\n")
	fmt.Print("正在监听按键 (按下 'q' 退出)...\r\n")
	fmt.Print("-------------------------------------------\r\n")

	buf := make([]byte, 16)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("读取错误: %v\r\n", err)
			break
		}

		input := buf[:n]
		fmt.Printf("\r收到 %d 字节: Hex=%X | 预览=%q\r\n", n, input, string(input))

		if n == 1 && input[0] == 'q' {
			fmt.Print("退出中...\r\n")
			break
		}
	}
}
