module kamishell

go 1.26.0

require (
	github.com/WJQSERVER/readline v0.0.0-20260219133359-50c005a0d74f
	github.com/chzyer/readline v1.5.1
)

require (
	github.com/clipperhouse/uax29/v2 v2.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.20 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/term v0.40.0 // indirect
)

replace github.com/WJQSERVER/readline => ./third_party/readline
