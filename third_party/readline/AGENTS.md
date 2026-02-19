# Readline Library Implementation Notes

## Architecture
- **internal/term**: Handles raw mode and window size.
- **internal/buffer**: Manages the line content as `[]rune`.
- **internal/input**: Parses ANSI sequences into events.
- **internal/render**: Draws the line and handles the cursor.

## Unicode Support
- Use `github.com/mattn/go-runewidth` for visual width calculation.
- Always handle characters as `rune`.

## Multi-platform
- Unix: Uses `termios` via `golang.org/x/sys/unix`.
- Windows: Uses `Console API` via `golang.org/x/sys/windows`.

## Debugging and Testing
- **Unit Tests**: Run `go test ./...` to verify basic logic.
- **Interactive Debugger**: A raw key debugger is available in `debug/main.go`.
  Run it with: `go run debug/main.go`
  Use this to verify if ANSI escape sequences (arrows, home, etc.) are being correctly parsed in your terminal.
- **Example App**: `go run example/main.go` for a full feature demonstration.

## Common Issues
- If arrow keys are not working, check `internal/input/input.go` to see if the escape sequence matches your terminal.
- Use CHA (`\x1b[nG`) for cursor movement in the renderer for better stability.
