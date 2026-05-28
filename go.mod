module github.com/WJQSERVER/kamishell

go 1.26.0

require (
	github.com/WJQSERVER-STUDIO/go-utils/iox v0.0.3
	github.com/WJQSERVER-STUDIO/httpc v0.9.3
	github.com/WJQSERVER/readline v0.0.0-20260219133359-50c005a0d74f
	github.com/valyala/bytebufferpool v1.0.0
	golang.org/x/sys v0.43.0
)

require (
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/go-json-experiment/json v0.0.0-20260430182902-b6187a392ed4 // indirect
	github.com/mattn/go-runewidth v0.0.21 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/term v0.42.0 // indirect
)

replace github.com/WJQSERVER/readline => ./readline
