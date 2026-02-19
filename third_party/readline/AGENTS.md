# Readline Library Implementation Notes

## Architecture
- **internal/term**: Handles raw mode and window size (using `golang.org/x/term`).
- **internal/buffer**: Manages the line content as `[]rune`.
- **internal/input**: Parses ANSI sequences into events.
- **internal/render**: Draws the line and handles the cursor.

## Unicode Support
- Use `github.com/mattn/go-runewidth` for visual width calculation.
- Always handle characters as `rune`.

## Multi-platform
- Core terminal logic now relies on `golang.org/x/term` for robust cross-platform support.

## Debugging and Testing
- **Unit Tests**: Run `go test ./...` to verify basic logic.
- **Parsed Debugger**: `go run debug/main.go`
  - Shows how the library interprets keys after parsing ANSI sequences.
- **Raw HEX Debugger**: `go run debug/raw.go`
  - Directly captures every byte from Stdin.
  - **Crucial for Windows/PowerShell**: Use this to find the exact Hex sequence (e.g., `1B 64`) for any key combination.

## Common Issues
- **First line not showing**: Ensure `ENABLE_PROCESSED_OUTPUT` is set on Windows.
- **Cursor jitter**: Use CHA (`\x1b[nG`) and hide/show cursor during redraw.
