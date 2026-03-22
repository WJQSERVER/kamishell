package readline

import "os"

type Config struct {
	Prompt    string
	History   History
	Completer Completer
	Stdin     *os.File
	Stdout    *os.File
}

func (c *Config) Init() {
	if c.Stdin == nil {
		c.Stdin = os.Stdin
	}
	if c.Stdout == nil {
		c.Stdout = os.Stdout
	}
	if c.History == nil {
		c.History = NewHistory()
	}
}
