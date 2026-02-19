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

## Verification
- Run `go test ./...` to verify basic logic.
- Run `go build -o example_bin example/main.go` to ensure it compiles.
