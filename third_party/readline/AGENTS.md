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
- Core terminal handling relies on `golang.org/x/term` for robust cross-platform stability.

## Debugging and Testing
- **Unit Tests**: Run `go test ./...` to verify internal logic.
- **Unified Debugger**: `go run debug/main.go` (Run from `third_party/readline` directory)
  - This tool simultaneously displays **RAW HEX BYTES** and the **PARSED RESULT**.
  - Use this to diagnose any key combination issues, especially on Windows/PowerShell.
- **Example App**: `go run example/main.go` for a full feature demonstration.

## Common Issues
- **First line not showing**: Ensure `ENABLE_PROCESSED_OUTPUT` is set on Windows (see `term_windows.go`).
- **Cursor jitter**: Use CHA (`\x1b[nG`) and hide/show cursor during redraw to ensure a smooth UI.
